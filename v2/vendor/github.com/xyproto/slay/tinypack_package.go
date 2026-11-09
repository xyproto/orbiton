package slay

import (
	"bytes"
	"debug/elf"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xyproto/env/v2"
	lua "github.com/yuin/gopher-lua"
)

var tinyCategories = map[string][]string{
	"doc":    {"usr/share/doc", "usr/share/gtk-doc", "usr/share/help", "usr/share/devhelp"},
	"man":    {"usr/share/man", "usr/share/info"},
	"locale": {"usr/share/locale"},
	"dev":    {"usr/include", "usr/lib/pkgconfig", "usr/share/pkgconfig", "usr/lib/cmake", "usr/share/aclocal", "usr/share/gir-1.0", "usr/share/vala"},
}

var tinyCategoryDesc = map[string]string{"doc": "documentation", "man": "manual pages", "locale": "translations", "dev": "development files"}

func (b *TinyBuilder) option(name string, def bool) bool {
	for o := range strings.FieldsSeq(env.Str("TINYPACK_OPTIONS")) {
		if o == name {
			def = true
		} else if o == "!"+name {
			def = false
		}
	}
	return b.R.HasOption(name, def)
}

type tinyOutput struct {
	Name    string
	Desc    string
	Dir     string
	Arch    string
	Depends []string
	Sub     string
}

func (b *TinyBuilder) finish() ([]string, error) {
	b.logf("Tidying install")
	if err := b.unSysroot(b.PkgDir); err != nil {
		return nil, err
	}
	if err := b.tidy(b.PkgDir); err != nil {
		return nil, err
	}
	outputs := []*tinyOutput{{Name: b.R.Name, Desc: b.R.Desc, Dir: b.PkgDir, Arch: b.R.PkgArch(), Depends: append([]string{}, b.R.Depends...)}}
	for _, sub := range b.R.Subpackages {
		dir := filepath.Join(b.Work, "pkg-"+sub)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		o := &tinyOutput{Name: b.R.Name + "-" + sub, Dir: dir, Arch: b.R.PkgArch(), Sub: sub}
		if f := b.R.Function("package_" + sub); f != nil {
			bt := b.luaBuilder()
			bt.RawSetString("pkgdir", lua.LString(dir))
			savedPkg := b.PkgDir
			b.PkgDir = dir
			b.cwd = b.buildDir()
			err := b.callLua(f, bt)
			b.PkgDir = savedPkg
			if err != nil {
				return nil, err
			}
			if err := b.tidy(dir); err != nil {
				return nil, err
			}
		} else if paths, ok := tinyCategories[sub]; ok {
			for _, p := range paths {
				if err := moveTree(filepath.Join(b.PkgDir, p), filepath.Join(dir, p)); err != nil {
					return nil, err
				}
			}
			if sub == "dev" {
				if err := moveDevLibs(b.PkgDir, dir); err != nil {
					return nil, err
				}
			}
		} else {
			return nil, fmt.Errorf("subpackage %q needs a package_%s function, or must be one of doc, man, locale, dev", sub, sub)
		}
		desc := tinyCategoryDesc[sub]
		if desc == "" {
			desc = sub
		}
		o.Desc = b.R.Desc + " (" + desc + ")"
		if sub == "dev" {
			o.Depends = []string{b.R.Name + "=" + b.R.FullVersion()}
		} else if sub != "doc" && sub != "man" && sub != "locale" {
			o.Depends = nil
		}
		outputs = append(outputs, o)
	}
	var files []string
	var problems []string
	for _, o := range outputs {
		auto, errs := b.check(o)
		problems = append(problems, errs...)
		for _, d := range auto {
			if !hasDep(o.Depends, d) && d != o.Name {
				o.Depends = append(o.Depends, d)
			}
		}
	}
	if len(problems) > 0 {
		for _, p := range problems {
			fmt.Fprintln(os.Stderr, "tinypack: "+p)
		}
		if env.Str("TINYPACK_CHECK", "error") != "warn" {
			return nil, fmt.Errorf("%d check(s) failed", len(problems))
		}
	}
	if err := os.MkdirAll(b.PkgDest, 0o755); err != nil {
		return nil, err
	}
	for _, o := range outputs {
		file, err := b.writePackage(o)
		if err != nil {
			return nil, err
		}
		b.logf("Created %s", file)
		files = append(files, file)
	}
	return files, nil
}

func (b *TinyBuilder) unSysroot(dir string) error {
	if !b.usesSysroot() {
		return nil
	}
	leaked := filepath.Join(dir, b.Sysroot)
	if _, err := os.Lstat(leaked); err != nil {
		return nil
	}
	b.logf("Moving files that were installed below the sysroot path")
	err := filepath.WalkDir(leaked, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(leaked, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.Rename(path, target)
	})
	if err != nil {
		return err
	}
	top, _, _ := strings.Cut(strings.TrimPrefix(b.Sysroot, "/"), "/")
	if top != "" && top != "usr" && top != "etc" && top != "opt" && top != "var" {
		return os.RemoveAll(filepath.Join(dir, top))
	}
	return nil
}

func hasDep(deps []string, name string) bool {
	for _, d := range deps {
		if depName(d) == depName(name) {
			return true
		}
	}
	return false
}

func moveTree(src, dest string) error {
	if _, err := os.Lstat(src); err != nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.Rename(src, dest)
}

func moveDevLibs(pkg, dev string) error {
	dir := filepath.Join(pkg, "usr/lib")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		n := e.Name()
		if strings.HasSuffix(n, ".a") || (strings.HasSuffix(n, ".so") && e.Type()&fs.ModeSymlink != 0) {
			if err := moveTree(filepath.Join(dir, n), filepath.Join(dev, "usr/lib", n)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *TinyBuilder) tidy(dir string) error {
	os.Remove(filepath.Join(dir, "usr/share/info/dir"))
	if !b.option("docs", true) {
		for _, d := range []string{"usr/share/doc", "usr/share/gtk-doc", "usr/doc", "usr/local/share/doc"} {
			os.RemoveAll(filepath.Join(dir, d))
		}
	}
	keepLibtool := b.option("libtool", false)
	keepStatic := b.option("staticlibs", false)
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		name := d.Name()
		switch {
		case name == "perllocal.pod" || name == ".packlist":
			return os.Remove(path)
		case strings.HasSuffix(name, ".la") && !keepLibtool:
			return os.Remove(path)
		case strings.HasSuffix(name, ".a") && !keepStatic:
			if _, err := os.Lstat(strings.TrimSuffix(path, ".a") + ".so"); err == nil {
				return os.Remove(path)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if b.option("zipman", true) {
		if err := zipMan(dir); err != nil {
			return err
		}
	}
	if b.option("strip", true) {
		b.stripTree(dir)
	}
	if !b.option("emptydirs", true) {
		removeEmptyDirs(dir)
	}
	return nil
}

func zipMan(dir string) error {
	for _, root := range []string{"usr/share/man", "usr/share/info"} {
		top := filepath.Join(dir, root)
		if _, err := os.Stat(top); err != nil {
			continue
		}
		var links []string
		err := filepath.WalkDir(top, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || strings.HasSuffix(path, ".gz") {
				return err
			}
			if d.Type()&fs.ModeSymlink != 0 {
				links = append(links, path)
				return nil
			}
			if !d.Type().IsRegular() {
				return nil
			}
			if out, err := exec.Command("gzip", "-9", "-n", "-f", path).CombinedOutput(); err != nil {
				return fmt.Errorf("gzip %s: %v: %s", path, err, out)
			}
			return nil
		})
		if err != nil {
			return err
		}
		for _, l := range links {
			target, err := os.Readlink(l)
			if err != nil {
				continue
			}
			os.Remove(l)
			if !strings.HasSuffix(target, ".gz") {
				target += ".gz"
			}
			if err := os.Symlink(target, l+".gz"); err != nil {
				return err
			}
		}
	}
	return nil
}

func removeEmptyDirs(dir string) {
	var dirs []string
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() && path != dir {
			dirs = append(dirs, path)
		}
		return nil
	})
	sort.Sort(sort.Reverse(sort.StringSlice(dirs)))
	for _, d := range dirs {
		os.Remove(d)
	}
}

func (b *TinyBuilder) stripTree(dir string) {
	strip := b.tools["strip"]
	if strip == "" {
		return
	}
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return nil
		}
		f, err := elf.Open(path)
		if err != nil {
			return nil
		}
		typ := f.Type
		interp := false
		for _, p := range f.Progs {
			if p.Type == elf.PT_INTERP {
				interp = true
			}
		}
		f.Close()
		var flag string
		switch {
		case typ == elf.ET_EXEC || (typ == elf.ET_DYN && interp):
			flag = "--strip-all"
		case typ == elf.ET_DYN:
			flag = "--strip-unneeded"
		case typ == elf.ET_REL && strings.HasSuffix(path, ".ko"):
			flag = "--strip-debug"
		default:
			return nil
		}
		info, _ := os.Stat(path)
		exec.Command(strip, flag, path).Run()
		if info != nil {
			os.Chmod(path, info.Mode().Perm())
		}
		return nil
	})
}

func maxVersion(names []string, prefix string) string {
	best := ""
	for _, n := range names {
		if strings.HasPrefix(n, prefix) && compareDotted(strings.TrimPrefix(n, prefix), strings.TrimPrefix(best, prefix)) > 0 {
			best = n
		}
	}
	return best
}

func compareDotted(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) || i < len(bs); i++ {
		var x, y int
		if i < len(as) {
			x, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			y, _ = strconv.Atoi(bs[i])
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}

func (b *TinyBuilder) definedVersions(lib string) []string {
	path := filepath.Join(b.Sysroot, "usr/lib", lib)
	f, err := elf.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	vs, err := f.DynamicVersions()
	if err != nil {
		return nil
	}
	var out []string
	for _, v := range vs {
		out = append(out, v.Name)
	}
	return out
}

func (b *TinyBuilder) check(o *tinyOutput) ([]string, []string) {
	var problems []string
	autoSet := map[string]bool{}
	provided := map[string]bool{}
	filepath.WalkDir(o.Dir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			provided[d.Name()] = true
		}
		return nil
	})
	limits := map[string]string{}
	for lib, prefix := range map[string]string{"libc.so.6": "GLIBC_", "libstdc++.so.6": "GLIBCXX_"} {
		if v := maxVersion(b.definedVersions(lib), prefix); v != "" {
			limits[prefix] = v
		}
	}
	work := []byte(b.Work)
	filepath.WalkDir(o.Dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return nil
		}
		rel, _ := filepath.Rel(o.Dir, path)
		if info, err := d.Info(); err == nil && info.Size() < 64<<20 {
			if data, err := os.ReadFile(path); err == nil && bytes.Contains(data, work) {
				fmt.Fprintf(os.Stderr, "tinypack: warning: %s contains the build directory path\n", rel)
			}
		}
		f, err := elf.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		if o.Arch == "any" {
			problems = append(problems, fmt.Sprintf("%s: ELF file in an arch=any package", rel))
			return nil
		}
		if f.Machine != b.T.Machine || f.Class != b.T.Class {
			problems = append(problems, fmt.Sprintf("%s: built for %s/%s, not %s", rel, f.Machine, f.Class, b.R.Carch))
			return nil
		}
		if f.Type != elf.ET_EXEC && f.Type != elf.ET_DYN {
			return nil
		}
		needed, _ := f.ImportedLibraries()
		for _, lib := range needed {
			switch {
			case provided[lib]:
			case b.Sys != nil && b.Sys.Owners[lib] != "":
				autoSet[b.Sys.Owners[lib]] = true
			case b.Sysroot == "/":
				if owner := hostPackageOwner(filepath.Join("/usr/lib", lib)); owner != "" {
					autoSet[owner] = true
				} else if _, err := os.Stat(filepath.Join("/usr/lib", lib)); err != nil {
					problems = append(problems, fmt.Sprintf("%s: needs %s, which nothing provides", rel, lib))
				}
			default:
				if _, err := os.Stat(filepath.Join(b.Sysroot, "usr/lib", lib)); err != nil {
					problems = append(problems, fmt.Sprintf("%s: needs %s, which is not in libdepends or the package", rel, lib))
				}
			}
		}
		syms, _ := f.ImportedSymbols()
		for prefix, limit := range limits {
			var used []string
			for _, s := range syms {
				if strings.HasPrefix(s.Version, prefix) {
					used = append(used, s.Version)
				}
			}
			if need := maxVersion(used, prefix); need != "" && compareDotted(strings.TrimPrefix(need, prefix), strings.TrimPrefix(limit, prefix)) > 0 {
				problems = append(problems, fmt.Sprintf("%s: needs %s, but the target only has %s", rel, need, limit))
			}
		}
		return nil
	})
	var auto []string
	for d := range autoSet {
		auto = append(auto, d)
	}
	sort.Strings(auto)
	return auto, problems
}

func dirSize(dir string) int64 {
	var total int64
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			if info, err := d.Info(); err == nil {
				total += info.Size()
			}
		}
		return nil
	})
	return total
}

func (b *TinyBuilder) pkgInfo(o *tinyOutput) string {
	var s strings.Builder
	w := func(k, v string) {
		if v != "" {
			s.WriteString(k + " = " + v + "\n")
		}
	}
	s.WriteString("# Generated by slay tinypack\n")
	w("pkgname", o.Name)
	w("pkgbase", b.R.Name)
	if len(b.R.Subpackages) > 0 {
		w("xdata", "pkgtype=split")
	} else {
		w("xdata", "pkgtype=pkg")
	}
	w("pkgver", b.R.FullVersion())
	w("pkgdesc", o.Desc)
	w("url", b.R.URL)
	w("builddate", strconv.FormatInt(b.Epoch, 10))
	w("packager", env.Str("PACKAGER", "Unknown Packager"))
	w("size", strconv.FormatInt(dirSize(o.Dir), 10))
	w("arch", o.Arch)
	for _, l := range b.R.License {
		w("license", l)
	}
	main := o.Sub == ""
	if main {
		for _, v := range b.R.Replaces {
			w("replaces", v)
		}
		for _, v := range b.R.Groups {
			w("group", v)
		}
		for _, v := range b.R.Conflicts {
			w("conflict", v)
		}
		for _, v := range b.R.Provides {
			w("provides", v)
		}
		for _, v := range b.R.Backup {
			w("backup", strings.TrimPrefix(v, "/"))
		}
	}
	for _, v := range o.Depends {
		w("depend", v)
	}
	if main {
		for _, v := range b.R.OptDepends {
			w("optdepend", v)
		}
	}
	for _, v := range append(append([]string{}, b.R.MakeDepends...), b.R.LibDepends...) {
		w("makedepend", v)
	}
	return s.String()
}

func clampTimes(dir string, epoch int64) {
	t := time.Unix(epoch, 0)
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.Type()&fs.ModeSymlink == 0 {
			if info, err := d.Info(); err == nil && info.ModTime().After(t) {
				os.Chtimes(path, t, t)
			}
		}
		return nil
	})
}

func (b *TinyBuilder) writePackage(o *tinyOutput) (string, error) {
	if err := os.WriteFile(filepath.Join(o.Dir, ".PKGINFO"), []byte(b.pkgInfo(o)), 0o644); err != nil {
		return "", err
	}
	clampTimes(o.Dir, b.Epoch)
	entries, err := os.ReadDir(o.Dir)
	if err != nil {
		return "", err
	}
	var names []string
	for _, e := range entries {
		if n := e.Name(); n != ".PKGINFO" && n != ".MTREE" {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	owner := []string{"--uid", "0", "--gid", "0", "--uname", "root", "--gname", "root"}
	mtree := exec.Command("bsdtar", append(append([]string{"-czf", ".MTREE", "--format=mtree", "--options=!all,use-set,type,uid,gid,mode,time,size,md5,sha256,link"}, owner...), append([]string{".PKGINFO"}, names...)...)...)
	mtree.Dir = o.Dir
	mtree.Env = append(os.Environ(), "LANG=C")
	if out, err := mtree.CombinedOutput(); err != nil {
		return "", fmt.Errorf("creating .MTREE: %v: %s", err, out)
	}
	os.Chtimes(filepath.Join(o.Dir, ".MTREE"), time.Unix(b.Epoch, 0), time.Unix(b.Epoch, 0))
	file := filepath.Join(b.PkgDest, o.Name+"-"+b.R.FullVersion()+"-"+o.Arch+".pkg.tar.zst")
	part := file + ".part"
	tar := exec.Command("bsdtar", append(append([]string{"-cf", "-"}, owner...), append([]string{".MTREE", ".PKGINFO"}, names...)...)...)
	tar.Dir = o.Dir
	tar.Env = append(os.Environ(), "LANG=C")
	tar.Stderr = os.Stderr
	zst := exec.Command("zstd", "-c", "-T0", "-19", "-q")
	pipe, err := tar.StdoutPipe()
	if err != nil {
		return "", err
	}
	zst.Stdin = pipe
	out, err := os.Create(part)
	if err != nil {
		return "", err
	}
	zst.Stdout = out
	zst.Stderr = os.Stderr
	if err := zst.Start(); err != nil {
		out.Close()
		return "", err
	}
	if err := tar.Run(); err != nil {
		out.Close()
		zst.Wait()
		os.Remove(part)
		return "", fmt.Errorf("bsdtar: %w", err)
	}
	if err := zst.Wait(); err != nil {
		out.Close()
		os.Remove(part)
		return "", fmt.Errorf("zstd: %w", err)
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	return file, os.Rename(part, file)
}
