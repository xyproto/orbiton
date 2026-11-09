package slay

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type tinyRepoPkg struct {
	File     string
	Name     string
	Version  string
	Arch     string
	Depends  []string
	Provides []string
}

type TinySysroot struct {
	Dir      string
	Packages []*tinyRepoPkg
	Owners   map[string]string
}

var tinyBase = []string{"gcc-libs", "bash", "zlib", "bzip2", "xz", "zstd", "lz4", "ncurses", "readline", "openssl", "libxcrypt",
	"expat", "libffi", "util-linux-libs", "attr", "acl", "pcre2", "gmp", "libcap", "brotli", "libelf"}

var pkgSuffixes = []string{".pkg.tar.zst", ".pkg.tar.xz", ".pkg.tar.gz", ".pkg.tar"}

func pkgFileArch(name string) string {
	for _, suf := range pkgSuffixes {
		if before, ok := strings.CutSuffix(name, suf); ok {
			base := before
			if i := strings.LastIndex(base, "-"); i >= 0 {
				return base[i+1:]
			}
		}
	}
	return ""
}

func depName(dep string) string {
	if i := strings.IndexAny(dep, "<>=:"); i >= 0 {
		dep = dep[:i]
	}
	return strings.TrimSpace(dep)
}

func readPkgInfo(file string) (*tinyRepoPkg, error) {
	out, err := exec.Command("bsdtar", "-xOf", file, ".PKGINFO").Output()
	if err != nil {
		return nil, fmt.Errorf("%s: could not read .PKGINFO: %w", file, err)
	}
	p := &tinyRepoPkg{File: file}
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		key, value, ok := strings.Cut(sc.Text(), " = ")
		if !ok {
			continue
		}
		switch key {
		case "pkgname":
			p.Name = value
		case "pkgver":
			p.Version = value
		case "arch":
			p.Arch = value
		case "depend":
			p.Depends = append(p.Depends, value)
		case "provides":
			p.Provides = append(p.Provides, value)
		}
	}
	return p, nil
}

func vercmp(a, b string) int {
	if a == b {
		return 0
	}
	out, err := exec.Command("vercmp", a, b).Output()
	if err != nil {
		return strings.Compare(a, b)
	}
	switch strings.TrimSpace(string(out)) {
	case "-1":
		return -1
	case "1":
		return 1
	}
	return 0
}

func scanTinyRepos(repos []string, carch string) (map[string]*tinyRepoPkg, map[string][]*tinyRepoPkg, error) {
	var files []string
	for _, repo := range repos {
		entries, err := os.ReadDir(repo)
		if err != nil {
			return nil, nil, err
		}
		for _, e := range entries {
			if a := pkgFileArch(e.Name()); a == carch || a == "any" {
				files = append(files, filepath.Join(repo, e.Name()))
			}
		}
	}
	pkgs := make([]*tinyRepoPkg, len(files))
	errs := make([]error, len(files))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 16)
	for i, f := range files {
		wg.Add(1)
		go func(i int, f string) {
			defer wg.Done()
			sem <- struct{}{}
			pkgs[i], errs[i] = readPkgInfo(f)
			<-sem
		}(i, f)
	}
	wg.Wait()
	byName := map[string]*tinyRepoPkg{}
	for i, p := range pkgs {
		if errs[i] != nil {
			return nil, nil, errs[i]
		}
		if old, ok := byName[p.Name]; !ok || vercmp(p.Version, old.Version) > 0 {
			byName[p.Name] = p
		}
	}
	providers := map[string][]*tinyRepoPkg{}
	for _, p := range byName {
		for _, prov := range p.Provides {
			n := depName(prov)
			providers[n] = append(providers[n], p)
		}
	}
	return byName, providers, nil
}

func resolveTinyDeps(want []string, byName map[string]*tinyRepoPkg, providers map[string][]*tinyRepoPkg) ([]*tinyRepoPkg, []string) {
	seen := map[string]bool{}
	var order []*tinyRepoPkg
	var missing []string
	queue := append([]string{}, want...)
	for len(queue) > 0 {
		name := depName(queue[0])
		queue = queue[1:]
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		p := byName[name]
		if p == nil {
			if cands := providers[name]; len(cands) > 0 {
				sort.Slice(cands, func(i, j int) bool { return cands[i].Name < cands[j].Name })
				p = cands[0]
			}
		}
		if p == nil {
			missing = append(missing, name)
			continue
		}
		if seen["pkg:"+p.Name] {
			continue
		}
		seen["pkg:"+p.Name] = true
		order = append(order, p)
		queue = append(queue, p.Depends...)
	}
	return order, missing
}

func BuildTinySysroot(dir string, repos []string, carch string, want, soft []string) (*TinySysroot, error) {
	byName, providers, err := scanTinyRepos(repos, carch)
	if err != nil {
		return nil, err
	}
	base := []string{"glibc", "linux-api-headers"}
	extras := strings.Fields(os.Getenv("TINYPACK_BASE"))
	if len(extras) == 0 {
		extras = tinyBase
	}
	for _, extra := range extras {
		if _, ok := byName[extra]; ok {
			base = append(base, extra)
		}
	}
	pkgs, missing := resolveTinyDeps(append(append(base, want...), soft...), byName, providers)
	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "tinypack: not in the %s repositories, skipped: %s\n", carch, strings.Join(missing, ", "))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &TinySysroot{Dir: dir, Packages: pkgs, Owners: map[string]string{}}
	for _, p := range pkgs {
		cmd := exec.Command("bsdtar", "-x", "-p", "--no-xattrs", "--no-acls", "--no-fflags", "-f", p.File, "-C", dir,
			"--exclude", ".PKGINFO", "--exclude", ".BUILDINFO", "--exclude", ".MTREE", "--exclude", ".INSTALL", "--exclude", ".CHANGELOG")
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("extracting %s: %v: %s", filepath.Base(p.File), err, out)
		}
		list, err := exec.Command("bsdtar", "-tf", p.File).Output()
		if err != nil {
			return nil, err
		}
		for line := range strings.SplitSeq(string(list), "\n") {
			if strings.HasPrefix(line, "usr/lib/") && strings.Contains(line, ".so") {
				s.Owners[filepath.Base(line)] = p.Name
			}
		}
	}
	if err := relativizeLinks(dir); err != nil {
		return nil, err
	}
	return s, nil
}

func relativizeLinks(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.Type()&fs.ModeSymlink == 0 {
			return err
		}
		target, err := os.Readlink(path)
		if err != nil || !filepath.IsAbs(target) {
			return err
		}
		rel, err := filepath.Rel(filepath.Dir(path), filepath.Join(root, target))
		if err != nil {
			return err
		}
		if err := os.Remove(path); err != nil {
			return err
		}
		return os.Symlink(rel, path)
	})
}

func hostPackageOwner(path string) string {
	out, err := exec.Command("pacman", "-Qqo", path).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
