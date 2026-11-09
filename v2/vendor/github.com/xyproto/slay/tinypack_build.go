package slay

import (
	"crypto/sha256"
	"crypto/sha512"
	"debug/elf"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/xyproto/env/v2"
)

type tinyTarget struct {
	Triple      string
	MesonCPU    string
	GoArch      string
	CargoTarget string
	Machine     elf.Machine
	Class       elf.Class
	Qemu        string
}

var tinyTargets = map[string]tinyTarget{
	"x86_64":  {"x86_64-pc-linux-gnu", "x86_64", "amd64", "x86_64-unknown-linux-gnu", elf.EM_X86_64, elf.ELFCLASS64, "qemu-x86_64"},
	"x32":     {"x86_64-pc-linux-gnux32", "x86_64", "", "x86_64-unknown-linux-gnux32", elf.EM_X86_64, elf.ELFCLASS32, ""},
	"i686":    {"i686-pc-linux-gnu", "x86", "386", "i686-unknown-linux-gnu", elf.EM_386, elf.ELFCLASS32, "qemu-i386"},
	"aarch64": {"aarch64-linux-gnu", "aarch64", "arm64", "aarch64-unknown-linux-gnu", elf.EM_AARCH64, elf.ELFCLASS64, "qemu-aarch64"},
	"armv7h":  {"arm-linux-gnueabihf", "arm", "arm", "armv7-unknown-linux-gnueabihf", elf.EM_ARM, elf.ELFCLASS32, "qemu-arm"},
	"riscv64": {"riscv64-linux-gnu", "riscv64", "riscv64", "riscv64gc-unknown-linux-gnu", elf.EM_RISCV, elf.ELFCLASS64, "qemu-riscv64"},
}

type TinyBuilder struct {
	R         *TinyRecipe
	T         tinyTarget
	HostT     tinyTarget
	Cross     bool
	Work      string
	SrcDir    string
	PkgDir    string
	BinDir    string
	HomeDir   string
	OutDir    string
	SrcDest   string
	PkgDest   string
	Sysroot   string
	Sys       *TinySysroot
	Repos     []string
	Jobs      int
	Epoch     int64
	Sandbox   bool
	cwd       string
	emulating bool
	extraEnv  map[string]string
	tools     map[string]string
}

func TinyPack(path string) ([]string, error) {
	carch := env.Str("CARCH", HostCarch())
	r, err := LoadTinyRecipe(path, carch)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	b, err := NewTinyBuilder(r)
	if err != nil {
		return nil, err
	}
	return b.Run()
}

func tinyCacheDir() string {
	if d := os.Getenv("XDG_CACHE_HOME"); d != "" {
		return filepath.Join(d, "slay")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "slay")
}

func NewTinyBuilder(r *TinyRecipe) (*TinyBuilder, error) {
	t, ok := tinyTargets[r.Carch]
	if !ok {
		return nil, fmt.Errorf("unknown target architecture %q", r.Carch)
	}
	host := HostCarch()
	b := &TinyBuilder{R: r, T: t, HostT: tinyTargets[host], Cross: r.Carch != host, extraEnv: map[string]string{}, tools: map[string]string{}}
	cache := tinyCacheDir()
	b.Work = filepath.Join(cache, "tinypack", r.Name+"-"+r.Carch)
	b.SrcDir = filepath.Join(b.Work, "src")
	b.PkgDir = filepath.Join(b.Work, "pkg")
	b.BinDir = filepath.Join(b.Work, "bin")
	b.HomeDir = filepath.Join(b.Work, "home")
	b.OutDir = filepath.Join(b.Work, "out")
	b.SrcDest = env.Str("SRCDEST", filepath.Join(cache, "sources"))
	b.PkgDest = env.Str("PKGDEST", r.Dir)
	for repo := range strings.SplitSeq(env.Str("REPO"), ":") {
		if repo != "" {
			b.Repos = append(b.Repos, repo)
		}
	}
	b.Sysroot = env.Str("SYSROOT")
	b.Jobs = env.Int("JOBS", runtime.NumCPU())
	b.Epoch = time.Now().Unix()
	if s := env.Str("SOURCE_DATE_EPOCH"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			b.Epoch = n
		}
	}
	_, err := exec.LookPath("bwrap")
	b.Sandbox = err == nil && env.Str("TINYPACK_SANDBOX", "1") != "0"
	if b.Cross && r.Cross == "no" {
		return nil, fmt.Errorf("%s can not be cross compiled (cross = \"no\")", r.Name)
	}
	if b.Cross && b.Sysroot == "" && len(b.Repos) == 0 {
		return nil, fmt.Errorf("cross compiling for %s needs REPO=dir[:dir...] with %s packages, or SYSROOT=dir", r.Carch, r.Carch)
	}
	return b, nil
}

func (b *TinyBuilder) logf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "==> "+format+"\n", args...)
}

func (b *TinyBuilder) Run() ([]string, error) {
	if !strings.Contains(b.Work, string(filepath.Separator)+"tinypack"+string(filepath.Separator)) {
		return nil, fmt.Errorf("refusing to use %s as a work directory", b.Work)
	}
	syscall.Umask(0o022)
	os.RemoveAll(b.Work)
	for _, d := range []string{b.SrcDir, b.PkgDir, b.BinDir, b.HomeDir, b.OutDir, b.SrcDest} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}
	b.logf("Making package: %s %s (%s)", b.R.Name, b.R.FullVersion(), b.R.Carch)
	srcs, err := b.fetchSources()
	if err != nil {
		return nil, err
	}
	if err := b.setupSysroot(); err != nil {
		return nil, err
	}
	if err := b.setupToolchain(); err != nil {
		return nil, err
	}
	if err := b.extractSources(srcs); err != nil {
		return nil, err
	}
	b.cwd = b.buildDir()
	bt := b.luaBuilder()
	if f := b.R.Function("prepare"); f != nil {
		b.logf("Starting prepare()")
		if err := b.callLua(f, bt); err != nil {
			return nil, err
		}
	} else {
		for _, p := range b.R.Patches {
			if err := b.patch(p); err != nil {
				return nil, err
			}
		}
	}
	if err := b.fetchDeps(); err != nil {
		return nil, err
	}
	b.cwd = b.buildDir()
	b.logf("Starting build()")
	if f := b.R.Function("build"); f != nil {
		if err := b.callLua(f, bt); err != nil {
			return nil, err
		}
	} else if err := b.styleBuild(); err != nil {
		return nil, err
	}
	b.cwd = b.buildDir()
	b.logf("Starting package()")
	if f := b.R.Function("package"); f != nil {
		if err := b.callLua(f, bt); err != nil {
			return nil, err
		}
	} else if err := b.styleInstall(); err != nil {
		return nil, err
	}
	b.cwd = b.buildDir()
	for _, in := range b.R.Install {
		if err := b.install(in.Src, in.Dest, in.Mode); err != nil {
			return nil, err
		}
	}
	for _, lic := range b.R.InstallLicense {
		if err := b.install(lic, "usr/share/licenses/"+b.R.Name+"/", 0o644); err != nil {
			return nil, err
		}
	}
	return b.finish()
}

type tinySource struct {
	Name string
	URL  string
	Kind string
	Ref  string
	Path string
	Sums map[string]string
}

func (b *TinyBuilder) parseSource(s string, i int) tinySource {
	src := tinySource{}
	name, url, hasName := strings.Cut(s, "::")
	if !hasName {
		url = s
		name = ""
	}
	src.Sums = map[string]string{}
	for algo, sums := range map[string][]string{"sha256": b.R.SHA256, "sha512": b.R.SHA512, "b2": b.R.B2} {
		if i < len(sums) && !strings.EqualFold(sums[i], "SKIP") {
			src.Sums[algo] = sums[i]
		}
	}
	switch {
	case strings.HasPrefix(url, "git+"):
		src.Kind = "git"
		url = strings.TrimPrefix(url, "git+")
		if base, frag, ok := strings.Cut(url, "#"); ok {
			url = base
			src.Ref = frag
		}
		if name == "" {
			name = strings.TrimSuffix(filepath.Base(url), ".git")
		}
	case strings.Contains(url, "://"):
		src.Kind = "http"
		if name == "" {
			name = filepath.Base(strings.SplitN(url, "?", 2)[0])
		}
	default:
		src.Kind = "file"
		if name == "" {
			name = filepath.Base(url)
		}
	}
	src.Name = name
	src.URL = url
	return src
}

func (b *TinyBuilder) fetchSources() ([]tinySource, error) {
	var out []tinySource
	for i, s := range b.R.Source {
		src := b.parseSource(s, i)
		switch src.Kind {
		case "file":
			src.Path = filepath.Join(b.R.Dir, src.URL)
			if _, err := os.Stat(src.Path); err != nil {
				return nil, fmt.Errorf("source file %s: %w", src.URL, err)
			}
		case "http":
			src.Path = filepath.Join(b.SrcDest, src.Name)
			if _, err := os.Stat(src.Path); err != nil {
				b.logf("Downloading %s", src.URL)
				if err := download(src.URL, src.Path); err != nil {
					return nil, err
				}
			}
		case "git":
			src.Path = filepath.Join(b.SrcDest, src.Name+".git")
			var cmd *exec.Cmd
			if _, err := os.Stat(src.Path); err == nil {
				cmd = exec.Command("git", "-C", src.Path, "fetch", "--all", "--tags", "--prune", "-q")
			} else {
				b.logf("Cloning %s", src.URL)
				cmd = exec.Command("git", "clone", "-q", "--mirror", src.URL, src.Path)
			}
			cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
			if err := cmd.Run(); err != nil {
				if _, statErr := os.Stat(src.Path); statErr != nil {
					return nil, fmt.Errorf("git %s: %w", src.URL, err)
				}
				b.logf("Could not update %s, using the cached copy", src.URL)
			}
		}
		if src.Kind != "git" {
			for algo, want := range src.Sums {
				sum, err := fileSum(algo, src.Path)
				if err != nil {
					return nil, err
				}
				if !strings.EqualFold(sum, want) {
					return nil, fmt.Errorf("%s: %s is %s, expected %s", src.Name, algo, sum, want)
				}
			}
		}
		out = append(out, src)
	}
	return out, nil
}

func download(url, dest string) error {
	if curl := lookTool("curl"); curl != "" {
		part := dest + ".part"
		cmd := exec.Command(curl, "-qgfL", "--retry", "3", "--retry-delay", "3", "-o", part, url)
		cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
		if err := cmd.Run(); err != nil {
			os.Remove(part)
			return fmt.Errorf("downloading %s: %w", url, err)
		}
		return os.Rename(part, dest)
	}
	client := &http.Client{Timeout: 30 * time.Minute}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "slay-tinypack")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", url, resp.Status)
	}
	part := dest + ".part"
	f, err := os.Create(part)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(part)
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(part, dest)
}

func fileSum(algo, path string) (string, error) {
	if algo == "b2" {
		out, err := exec.Command("b2sum", path).Output()
		if err != nil {
			return "", fmt.Errorf("b2sum %s: %w", path, err)
		}
		return strings.Fields(string(out))[0], nil
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var h hash.Hash = sha256.New()
	if algo == "sha512" {
		h = sha512.New()
	}
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func isArchive(name string) bool {
	for _, suf := range []string{".tar", ".tar.gz", ".tgz", ".tar.bz2", ".tbz2", ".tar.xz", ".txz", ".tar.zst", ".tar.lz", ".tar.lzma", ".zip", ".7z"} {
		if strings.HasSuffix(name, suf) {
			return true
		}
	}
	return false
}

func (b *TinyBuilder) extractSources(srcs []tinySource) error {
	noextract := map[string]bool{}
	for _, n := range b.R.NoExtract {
		noextract[n] = true
	}
	for _, src := range srcs {
		switch {
		case src.Kind == "git":
			dir := filepath.Join(b.SrcDir, strings.TrimSuffix(src.Name, ".git"))
			if out, err := exec.Command("git", "clone", "-q", "--no-checkout", src.Path, dir).CombinedOutput(); err != nil {
				return fmt.Errorf("git clone %s: %v: %s", src.Name, err, out)
			}
			ref := "origin/HEAD"
			if k, v, ok := strings.Cut(src.Ref, "="); ok {
				switch k {
				case "tag":
					ref = "refs/tags/" + v
					if len(v) == 40 && strings.Trim(v, "0123456789abcdef") == "" {
						ref = v
					}
				case "branch":
					ref = "origin/" + v
				default:
					ref = v
				}
			}
			if out, err := exec.Command("git", "-C", dir, "-c", "advice.detachedHead=false", "checkout", "-q", ref).CombinedOutput(); err != nil {
				return fmt.Errorf("git checkout %s in %s: %v: %s", ref, src.Name, err, out)
			}
		case isArchive(src.Name) && !noextract[src.Name]:
			if out, err := exec.Command("bsdtar", "-xf", src.Path, "-C", b.SrcDir).CombinedOutput(); err != nil {
				return fmt.Errorf("extracting %s: %v: %s", src.Name, err, out)
			}
		default:
			if err := tinyCopy(src.Path, filepath.Join(b.SrcDir, src.Name), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

func tinyCopy(src, dest string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chmod(dest, mode)
}

func (b *TinyBuilder) buildDir() string {
	if b.R.BuildDir != "" {
		return filepath.Join(b.SrcDir, b.R.BuildDir)
	}
	entries, _ := os.ReadDir(b.SrcDir)
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	if len(dirs) == 1 {
		return filepath.Join(b.SrcDir, dirs[0])
	}
	for _, guess := range []string{b.R.Name + "-" + b.R.Version, b.R.Name} {
		if info, err := os.Stat(filepath.Join(b.SrcDir, guess)); err == nil && info.IsDir() {
			return filepath.Join(b.SrcDir, guess)
		}
	}
	return b.SrcDir
}

func (b *TinyBuilder) setupSysroot() error {
	switch {
	case b.Sysroot != "":
		abs, err := filepath.Abs(b.Sysroot)
		if err != nil {
			return err
		}
		b.Sysroot = abs
	case len(b.Repos) > 0:
		b.logf("Assembling the %s sysroot", b.R.Carch)
		s, err := BuildTinySysroot(filepath.Join(b.Work, "sysroot"), b.Repos, b.R.Carch, b.R.LibDepends, b.R.Depends)
		if err != nil {
			return err
		}
		b.Sys = s
		b.Sysroot = s.Dir
	default:
		b.Sysroot = "/"
	}
	return nil
}

func (b *TinyBuilder) usesSysroot() bool {
	return b.Sysroot != "/"
}

func firstGlob(pattern string) string {
	m, _ := filepath.Glob(pattern)
	sort.Strings(m)
	if len(m) == 0 {
		return ""
	}
	return m[len(m)-1]
}

func shellQuote(s string) string {
	if s != "" && strings.IndexFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("-_./=:,+@%", r))
	}) < 0 {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func (b *TinyBuilder) writeScript(name, body string) error {
	return os.WriteFile(filepath.Join(b.BinDir, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755)
}

func lookTool(names ...string) string {
	for _, n := range names {
		if p, err := exec.LookPath(n); err == nil {
			return p
		}
	}
	return ""
}

func (b *TinyBuilder) setupToolchain() error {
	triple := b.T.Triple
	var cc, cxx string
	gccStyle := true
	switch {
	case !b.Cross:
		cc, cxx = lookTool("gcc", "cc"), lookTool("g++", "c++")
	case lookTool(triple+"-gcc") != "":
		cc, cxx = lookTool(triple+"-gcc"), lookTool(triple+"-g++")
	case lookTool("clang") != "":
		cc, cxx = lookTool("clang"), lookTool("clang++")
		gccStyle = false
	default:
		return fmt.Errorf("no compiler for %s: install %s-gcc or clang", b.R.Carch, triple)
	}
	if cc == "" {
		return fmt.Errorf("no C compiler found")
	}
	sys := b.Sysroot
	var ccFlags, cxxFlags []string
	if b.usesSysroot() {
		if gccStyle {
			out, err := exec.Command(cc, "-print-file-name=include").Output()
			if err != nil {
				return fmt.Errorf("%s -print-file-name=include: %w", cc, err)
			}
			gccInc := strings.TrimSpace(string(out))
			sysGCC := firstGlob(filepath.Join(sys, "usr/lib/gcc/*/*"))
			sysInc := []string{"-idirafter", gccInc}
			if info, err := os.Stat(gccInc + "-fixed"); err == nil && info.IsDir() {
				sysInc = append(sysInc, "-idirafter", gccInc+"-fixed")
			}
			sysInc = append(sysInc, "-idirafter", filepath.Join(sys, "usr/include"))
			libs := []string{"-B" + filepath.Join(sys, "usr/lib") + "/"}
			if sysGCC != "" {
				libs = append(libs, "-L"+sysGCC)
			}
			libs = append(libs, "-L"+filepath.Join(sys, "usr/lib"))
			ccFlags = append(append([]string{"--sysroot=" + sys, "-nostdinc"}, sysInc...), libs...)
			cxxFlags = []string{"--sysroot=" + sys}
			cxxInc := ""
			if sysGCC != "" {
				cxxInc = filepath.Join(sys, "usr/include/c++", filepath.Base(sysGCC))
			}
			if info, err := os.Stat(cxxInc); err == nil && info.IsDir() {
				cxxFlags = append(cxxFlags, "-nostdinc", "-nostdinc++", "-idirafter", cxxInc)
				if d := firstGlob(filepath.Join(cxxInc, "*-linux-gnu*")); d != "" {
					cxxFlags = append(cxxFlags, "-idirafter", d)
				}
				cxxFlags = append(cxxFlags, "-idirafter", filepath.Join(cxxInc, "backward"))
				cxxFlags = append(cxxFlags, sysInc...)
			}
			cxxFlags = append(cxxFlags, libs...)
		} else {
			ccFlags = []string{"--target=" + triple, "--sysroot=" + sys}
			if lookTool("ld.lld") != "" {
				ccFlags = append(ccFlags, "-fuse-ld=lld")
			}
			cxxFlags = ccFlags
		}
	} else if !gccStyle {
		ccFlags = []string{"--target=" + triple}
		cxxFlags = ccFlags
	}
	quoteAll := func(args []string) string {
		q := make([]string, len(args))
		for i, a := range args {
			q[i] = shellQuote(a)
		}
		return strings.Join(q, " ")
	}
	if err := b.writeScript("cc", "exec "+shellQuote(cc)+" "+quoteAll(ccFlags)+` "$@"`); err != nil {
		return err
	}
	if cxx != "" {
		if err := b.writeScript("c++", "exec "+shellQuote(cxx)+" "+quoteAll(cxxFlags)+` "$@"`); err != nil {
			return err
		}
	}
	links := map[string]string{"gcc": "cc", "g++": "c++", triple + "-gcc": "cc", triple + "-cc": "cc", triple + "-g++": "c++", triple + "-c++": "c++"}
	for name, target := range links {
		if err := os.Symlink(target, filepath.Join(b.BinDir, name)); err != nil {
			return err
		}
	}
	prefix := ""
	if b.Cross {
		prefix = triple + "-"
	}
	for _, tool := range []string{"ar", "nm", "ranlib", "strip", "objcopy", "objdump", "readelf", "ld", "as", "size", "strings"} {
		var candidates []string
		switch tool {
		case "ar", "nm", "ranlib":
			if gccStyle {
				candidates = append(candidates, prefix+"gcc-"+tool)
			}
		}
		candidates = append(candidates, prefix+tool)
		if b.Cross || !gccStyle {
			candidates = append(candidates, "llvm-"+tool)
		}
		p := lookTool(candidates...)
		if p == "" {
			continue
		}
		b.tools[tool] = p
		for _, name := range []string{tool, triple + "-" + tool} {
			if err := os.Symlink(p, filepath.Join(b.BinDir, name)); err != nil {
				return err
			}
		}
	}
	pc := lookTool("pkgconf", "pkg-config")
	if pc != "" {
		body := "exec " + shellQuote(pc) + ` "$@"`
		if b.usesSysroot() {
			libdir := filepath.Join(sys, "usr/lib/pkgconfig") + ":" + filepath.Join(sys, "usr/share/pkgconfig")
			body = "PKG_CONFIG_SYSROOT_DIR=" + shellQuote(sys) + " PKG_CONFIG_LIBDIR=" + shellQuote(libdir) + " PKG_CONFIG_PATH= " + body
		}
		if err := b.writeScript(b.pkgConfigName(), body); err != nil {
			return err
		}
	}
	return b.writeCrossFiles()
}

func (b *TinyBuilder) pkgConfigName() string {
	if b.Cross {
		return b.T.Triple + "-pkg-config"
	}
	return "pkg-config"
}

func (b *TinyBuilder) emulator() []string {
	if !b.Cross || b.T.Qemu == "" || lookTool(b.T.Qemu) == "" {
		return nil
	}
	return []string{lookTool(b.T.Qemu), "-L", b.Sysroot}
}

func (b *TinyBuilder) writeCrossFiles() error {
	bin := func(n string) string { return filepath.Join(b.BinDir, n) }
	q := func(s string) string { return "'" + strings.ReplaceAll(s, "'", `\'`) + "'" }
	var m strings.Builder
	m.WriteString("[binaries]\n")
	m.WriteString("c = " + q(bin("cc")) + "\ncpp = " + q(bin("c++")) + "\npkg-config = " + q(bin(b.pkgConfigName())) + "\n")
	for _, t := range []string{"ar", "strip", "nm", "ranlib", "objcopy"} {
		if _, ok := b.tools[t]; ok {
			m.WriteString(t + " = " + q(bin(t)) + "\n")
		}
	}
	if emu := b.emulator(); emu != nil {
		quoted := make([]string, len(emu))
		for i, e := range emu {
			quoted[i] = q(e)
		}
		m.WriteString("exe_wrapper = [" + strings.Join(quoted, ", ") + "]\n")
	}
	m.WriteString("\n[properties]\n")
	if b.usesSysroot() {
		m.WriteString("sys_root = " + q(b.Sysroot) + "\n")
		m.WriteString("pkg_config_libdir = " + q(filepath.Join(b.Sysroot, "usr/lib/pkgconfig")+":"+filepath.Join(b.Sysroot, "usr/share/pkgconfig")) + "\n")
	}
	m.WriteString("\n[host_machine]\nsystem = 'linux'\ncpu_family = " + q(b.T.MesonCPU) + "\ncpu = " + q(b.R.Carch) + "\nendian = 'little'\n")
	if err := os.WriteFile(filepath.Join(b.Work, "meson-cross.ini"), []byte(m.String()), 0o644); err != nil {
		return err
	}
	var c strings.Builder
	proc := b.R.Carch
	if proc == "i686" {
		proc = "x86"
	}
	c.WriteString("set(CMAKE_SYSTEM_NAME Linux)\nset(CMAKE_SYSTEM_PROCESSOR " + proc + ")\n")
	c.WriteString("set(CMAKE_C_COMPILER \"" + bin("cc") + "\")\nset(CMAKE_CXX_COMPILER \"" + bin("c++") + "\")\n")
	for _, t := range []string{"ar", "ranlib", "strip", "nm", "objcopy"} {
		if _, ok := b.tools[t]; ok {
			c.WriteString("set(CMAKE_" + strings.ToUpper(t) + " \"" + bin(t) + "\")\n")
		}
	}
	if b.usesSysroot() {
		c.WriteString("set(CMAKE_SYSROOT \"" + b.Sysroot + "\")\nset(CMAKE_FIND_ROOT_PATH \"" + b.Sysroot + "\")\n")
		c.WriteString("set(CMAKE_FIND_ROOT_PATH_MODE_PROGRAM NEVER)\nset(CMAKE_FIND_ROOT_PATH_MODE_LIBRARY ONLY)\nset(CMAKE_FIND_ROOT_PATH_MODE_INCLUDE ONLY)\nset(CMAKE_FIND_ROOT_PATH_MODE_PACKAGE ONLY)\n")
	}
	if emu := b.emulator(); emu != nil {
		c.WriteString("set(CMAKE_CROSSCOMPILING_EMULATOR \"" + strings.Join(emu, ";") + "\")\n")
	}
	return os.WriteFile(filepath.Join(b.Work, "toolchain.cmake"), []byte(c.String()), 0o644)
}

func (b *TinyBuilder) buildEnv() []string {
	e := map[string]string{
		"PATH":              b.BinDir + ":/usr/local/bin:/usr/bin:/bin",
		"HOME":              b.HomeDir,
		"LANG":              "C.UTF-8",
		"SOURCE_DATE_EPOCH": strconv.FormatInt(b.Epoch, 10),
		"CARCH":             b.R.Carch,
		"CHOST":             b.T.Triple,
		"srcdir":            b.SrcDir,
		"pkgdir":            b.PkgDir,
		"MAKEFLAGS":         "-j" + strconv.Itoa(b.Jobs),
		"CC":                filepath.Join(b.BinDir, "cc"),
		"CXX":               filepath.Join(b.BinDir, "c++"),
		"PKG_CONFIG":        filepath.Join(b.BinDir, b.pkgConfigName()),
		"CFLAGS":            env.Str("CFLAGS", "-O2 -pipe -fno-plt -fexceptions -Wp,-D_FORTIFY_SOURCE=3 -Wformat -Werror=format-security -fstack-clash-protection"),
		"CXXFLAGS":          env.Str("CXXFLAGS", "-O2 -pipe -fno-plt -fexceptions -Wp,-D_FORTIFY_SOURCE=3 -Wformat -Werror=format-security -fstack-clash-protection -Wp,-D_GLIBCXX_ASSERTIONS"),
		"LDFLAGS":           env.Str("LDFLAGS", "-Wl,-O1 -Wl,--sort-common -Wl,--as-needed -Wl,-z,relro -Wl,-z,now"),
		"GOCACHE":           filepath.Join(b.Work, "go/cache"),
		"GOPATH":            filepath.Join(b.Work, "go/path"),
		"GOMODCACHE":        filepath.Join(b.SrcDest, "go-mod"),
		"GOTOOLCHAIN":       "local",
		"GOOS":              "linux",
		"CARGO_HOME":        filepath.Join(b.SrcDest, "cargo"),
		"CARGO_TARGET_DIR":  filepath.Join(b.Work, "cargo-target"),
	}
	if b.Cross && b.usesSysroot() {
		e["QEMU_LD_PREFIX"] = b.Sysroot
	}
	if b.Cross && b.R.Style == "meson" {
		delete(e, "PKG_CONFIG")
	}
	if b.T.GoArch != "" {
		e["GOARCH"] = b.T.GoArch
	}
	if b.R.Carch == "armv7h" {
		e["GOARM"] = "7"
	}
	if b.R.Carch == "i686" {
		e["GO386"] = "sse2"
	}
	for _, t := range []string{"ar", "nm", "ranlib", "strip", "objcopy", "objdump", "readelf"} {
		if _, ok := b.tools[t]; ok {
			e[strings.ToUpper(t)] = filepath.Join(b.BinDir, t)
		}
	}
	if b.Cross {
		e["CFLAGS"] = stripArchFlags(e["CFLAGS"])
		e["CXXFLAGS"] = stripArchFlags(e["CXXFLAGS"])
	}
	maps.Copy(e, b.R.Env)
	maps.Copy(e, b.extraEnv)
	if b.emulating {
		e = map[string]string{"PATH": "/usr/local/bin:/usr/bin:/bin", "HOME": b.HomeDir, "LANG": "C.UTF-8", "SOURCE_DATE_EPOCH": e["SOURCE_DATE_EPOCH"], "CARCH": b.R.Carch, "srcdir": b.SrcDir, "pkgdir": b.PkgDir, "MAKEFLAGS": e["MAKEFLAGS"]}
		maps.Copy(e, b.extraEnv)
	}
	keys := make([]string, 0, len(e))
	for k := range e {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+"="+e[k])
	}
	return out
}

func stripArchFlags(flags string) string {
	var out []string
	for f := range strings.FieldsSeq(flags) {
		if strings.HasPrefix(f, "-march=") || strings.HasPrefix(f, "-mtune=") {
			continue
		}
		out = append(out, f)
	}
	return strings.Join(out, " ")
}

func (b *TinyBuilder) run(network bool, argv ...string) error {
	if len(argv) == 0 {
		return nil
	}
	envv := b.buildEnv()
	var cmd *exec.Cmd
	if b.Sandbox {
		args := []string{"--die-with-parent", "--new-session", "--unshare-all"}
		if network {
			args = append(args, "--share-net", "--ro-bind-try", "/etc/resolv.conf", "/etc/resolv.conf")
		}
		if b.emulating {
			for _, d := range []string{b.Work, b.SrcDest, "dev", "proc", "tmp", "bin"} {
				os.MkdirAll(filepath.Join(b.Sysroot, d), 0o755)
			}
			args = append(args, "--ro-bind", b.Sysroot, "/")
		} else {
			args = append(args, "--ro-bind", "/", "/")
		}
		args = append(args, "--dev", "/dev", "--proc", "/proc", "--tmpfs", "/tmp", "--bind", b.Work, b.Work)
		if network || !b.emulating {
			args = append(args, "--bind", b.SrcDest, b.SrcDest)
		}
		args = append(args, "--chdir", b.cwd, "--clearenv")
		for _, kv := range envv {
			k, v, _ := strings.Cut(kv, "=")
			args = append(args, "--setenv", k, v)
		}
		args = append(args, "--")
		args = append(args, argv...)
		cmd = exec.Command("bwrap", args...)
	} else {
		if b.emulating {
			return fmt.Errorf("b.emulate needs bubblewrap")
		}
		cmd = exec.Command(argv[0], argv[1:]...)
		cmd.Dir = b.cwd
		cmd.Env = envv
	}
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", strings.Join(argv, " "), err)
	}
	return nil
}

func (b *TinyBuilder) sh(network bool, script string) error {
	return b.run(network, "/bin/sh", "-ec", script)
}

func (b *TinyBuilder) patch(p string) error {
	b.logf("Applying %s", p)
	path := filepath.Join(b.SrcDir, p)
	if _, err := os.Stat(path); err != nil {
		path = filepath.Join(b.R.Dir, p)
	}
	return b.run(false, "patch", "-Np1", "-i", path)
}

func (b *TinyBuilder) fetchDeps() error {
	dir := b.buildDir()
	exists := func(name string) bool {
		_, err := os.Stat(filepath.Join(dir, name))
		return err == nil
	}
	b.cwd = dir
	switch b.R.Style {
	case "go":
		if exists("go.mod") && !exists("vendor") {
			b.logf("Downloading Go modules")
			return b.run(true, "go", "mod", "download")
		}
	case "cargo":
		args := []string{"cargo", "fetch", "--target", b.T.CargoTarget}
		if exists("Cargo.lock") {
			args = append(args, "--locked")
		}
		b.logf("Downloading crates")
		return b.run(true, args...)
	}
	return nil
}

func (b *TinyBuilder) jobsArg() string {
	return strconv.Itoa(b.Jobs)
}

func (b *TinyBuilder) styleBuild() error {
	r := b.R
	switch r.Style {
	case "meson":
		args := []string{"meson", "setup", "build", "--prefix=/usr", "--libexecdir=lib", "--sbindir=bin", "--buildtype=plain", "--auto-features=enabled", "--wrap-mode=nodownload", "-Db_pie=true", "-Dpython.bytecompile=1"}
		if b.Cross {
			args = append(args, "--cross-file", filepath.Join(b.Work, "meson-cross.ini"))
		}
		if err := b.run(false, append(args, r.Configure...)...); err != nil {
			return err
		}
		return b.run(false, "meson", "compile", "-C", "build", "-j", b.jobsArg())
	case "cmake":
		args := []string{"cmake", "-B", "build", "-S", ".", "-DCMAKE_INSTALL_PREFIX=/usr", "-DCMAKE_INSTALL_LIBDIR=lib", "-DCMAKE_BUILD_TYPE=None"}
		if lookTool("ninja") != "" {
			args = append(args, "-G", "Ninja")
		}
		if b.Cross || b.usesSysroot() {
			args = append(args, "-DCMAKE_TOOLCHAIN_FILE="+filepath.Join(b.Work, "toolchain.cmake"))
		}
		if err := b.run(false, append(args, r.Configure...)...); err != nil {
			return err
		}
		return b.run(false, "cmake", "--build", "build", "-j", b.jobsArg())
	case "configure":
		dir := b.buildDir()
		if _, err := os.Stat(filepath.Join(dir, "configure")); err != nil {
			if _, err := os.Stat(filepath.Join(dir, "autogen.sh")); err == nil {
				if err := b.run(false, "sh", "./autogen.sh"); err != nil {
					return err
				}
			} else if err := b.run(false, "autoreconf", "-fi"); err != nil {
				return err
			}
		}
		args := []string{"./configure", "--prefix=/usr", "--sysconfdir=/etc", "--localstatedir=/var", "--libexecdir=/usr/lib", "--sbindir=/usr/bin"}
		if b.Cross {
			args = append(args, "--host="+b.T.Triple, "--build="+b.HostT.Triple)
		}
		if b.usesSysroot() {
			args = append(args, "--with-sysroot="+b.Sysroot)
		}
		if err := b.run(false, append(args, r.Configure...)...); err != nil {
			return err
		}
		return b.run(false, append([]string{"make", "-j" + b.jobsArg()}, r.MakeArgs...)...)
	case "make":
		return b.run(false, append([]string{"make", "-j" + b.jobsArg(), "PREFIX=/usr"}, r.MakeArgs...)...)
	case "go":
		pkgs := r.GoPackages
		if len(pkgs) == 0 {
			pkgs = []string{"."}
		}
		cgo := r.GoCGO
		if cgo == "" {
			cgo = "0"
			if len(r.LibDepends) > 0 {
				cgo = "1"
			}
		}
		b.extraEnv["CGO_ENABLED"] = cgo
		maps.Copy(b.extraEnv, r.GoEnv)
		if b.T.GoArch == "" {
			return fmt.Errorf("Go does not support %s", r.Carch)
		}
		for _, p := range pkgs {
			out := filepath.Join(b.Work, "go/bin", goOutName(p, r.Name))
			args := []string{"go", "build", "-o", out, "-trimpath", "-buildvcs=false"}
			if len(r.GoLDFlags) > 0 {
				args = append(args, "-ldflags", strings.Join(r.GoLDFlags, " "))
			}
			args = append(append(args, r.GoFlags...), p)
			if err := b.run(false, args...); err != nil {
				return err
			}
		}
		return nil
	case "cargo":
		target := b.T.CargoTarget
		key := "CARGO_TARGET_" + strings.ToUpper(strings.NewReplacer("-", "_", ".", "_").Replace(target)) + "_LINKER"
		b.extraEnv[key] = filepath.Join(b.BinDir, "cc")
		b.extraEnv["CC_"+strings.ReplaceAll(target, "-", "_")] = filepath.Join(b.BinDir, "cc")
		b.extraEnv["CXX_"+strings.ReplaceAll(target, "-", "_")] = filepath.Join(b.BinDir, "c++")
		if b.Cross {
			b.extraEnv["PKG_CONFIG_ALLOW_CROSS"] = "1"
			b.extraEnv["HOST_CC"] = lookTool("cc", "gcc")
		}
		args := []string{"cargo", "build", "--release", "--offline", "--target", target}
		if _, err := os.Stat(filepath.Join(b.buildDir(), "Cargo.lock")); err == nil {
			args = append(args, "--locked")
		}
		if len(r.CargoFeatures) > 0 {
			args = append(args, "--features", strings.Join(r.CargoFeatures, ","))
		}
		return b.run(false, args...)
	case "slay":
		self, err := os.Executable()
		if err != nil {
			return err
		}
		if err := b.run(false, append(append([]string{self}, r.SlayArgs...), "build")...); err != nil {
			return err
		}
		if exe := newestELFExecutable(b.cwd); exe != "" && filepath.Base(exe) != r.Name {
			return os.Rename(exe, filepath.Join(b.cwd, r.Name))
		}
		return nil
	case "python":
		return b.run(false, "python", "-m", "build", "--wheel", "--no-isolation")
	}
	return nil
}

func goOutName(pkg, fallback string) string {
	base := filepath.Base(strings.TrimSuffix(pkg, "/..."))
	if base == "." || base == "/" || base == "" {
		return fallback
	}
	return base
}

func (b *TinyBuilder) styleInstall() error {
	r := b.R
	switch r.Style {
	case "meson":
		return b.run(false, "meson", "install", "-C", "build", "--destdir", b.PkgDir, "--no-rebuild")
	case "cmake":
		b.extraEnv["DESTDIR"] = b.PkgDir
		defer delete(b.extraEnv, "DESTDIR")
		return b.run(false, "cmake", "--install", "build")
	case "configure":
		return b.run(false, append([]string{"make", "DESTDIR=" + b.PkgDir, "install"}, r.InstallArgs...)...)
	case "make":
		return b.run(false, append([]string{"make", "DESTDIR=" + b.PkgDir, "PREFIX=/usr", "install"}, r.InstallArgs...)...)
	case "go":
		entries, err := os.ReadDir(filepath.Join(b.Work, "go/bin"))
		if err != nil {
			return err
		}
		for _, e := range entries {
			if err := tinyCopy(filepath.Join(b.Work, "go/bin", e.Name()), filepath.Join(b.PkgDir, "usr/bin", e.Name()), 0o755); err != nil {
				return err
			}
		}
	case "cargo":
		dir := filepath.Join(b.Work, "cargo-target", b.T.CargoTarget, "release")
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, e := range entries {
			info, err := e.Info()
			if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 || strings.Contains(e.Name(), ".") {
				continue
			}
			if err := tinyCopy(filepath.Join(dir, e.Name()), filepath.Join(b.PkgDir, "usr/bin", e.Name()), 0o755); err != nil {
				return err
			}
		}
	case "slay":
		exe := newestELFExecutable(b.buildDir())
		if exe == "" {
			return fmt.Errorf("slay did not produce an executable in %s", b.buildDir())
		}
		return tinyCopy(exe, filepath.Join(b.PkgDir, "usr/bin", r.Name), 0o755)
	case "python":
		return b.sh(false, "python -m installer --destdir="+shellQuote(b.PkgDir)+" dist/*.whl")
	}
	return nil
}

func newestELFExecutable(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var best string
	var bestTime time.Time
	for _, e := range entries {
		info, err := e.Info()
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
			continue
		}
		path := filepath.Join(dir, e.Name())
		f, err := elf.Open(path)
		if err != nil {
			continue
		}
		f.Close()
		if best == "" || info.ModTime().After(bestTime) {
			best, bestTime = path, info.ModTime()
		}
	}
	return best
}

func (b *TinyBuilder) install(src, dest string, mode os.FileMode) error {
	pattern := src
	if !filepath.IsAbs(pattern) {
		pattern = filepath.Join(b.cwd, src)
	}
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		return fmt.Errorf("install: no files match %s", src)
	}
	dest = strings.TrimPrefix(dest, "/")
	toDir := strings.HasSuffix(dest, "/") || len(matches) > 1
	for _, m := range matches {
		target := filepath.Join(b.PkgDir, dest)
		if toDir {
			target = filepath.Join(target, filepath.Base(m))
		}
		if err := tinyCopy(m, target, mode); err != nil {
			return err
		}
	}
	return nil
}
