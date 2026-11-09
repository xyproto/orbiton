package slay

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

type TinyInstall struct {
	Src  string
	Dest string
	Mode os.FileMode
}

type TinyRecipe struct {
	File           string
	Dir            string
	Carch          string
	Name           string
	Version        string
	Release        string
	Epoch          string
	Desc           string
	URL            string
	License        []string
	Arch           []string
	Groups         []string
	Source         []string
	SHA256         []string
	SHA512         []string
	B2             []string
	NoExtract      []string
	Depends        []string
	OptDepends     []string
	MakeDepends    []string
	LibDepends     []string
	Provides       []string
	Conflicts      []string
	Replaces       []string
	Backup         []string
	Options        []string
	Subpackages    []string
	Patches        []string
	InstallLicense []string
	Install        []TinyInstall
	Style          string
	BuildDir       string
	Configure      []string
	MakeArgs       []string
	InstallArgs    []string
	GoPackages     []string
	GoLDFlags      []string
	GoFlags        []string
	GoCGO          string
	GoEnv          map[string]string
	CargoFeatures  []string
	SlayArgs       []string
	Cross          string
	Env            map[string]string
	L              *lua.LState
	Table          *lua.LTable
}

var tinyStyles = map[string]bool{"meson": true, "cmake": true, "configure": true, "make": true, "go": true, "cargo": true, "slay": true, "python": true, "copy": true, "custom": true}

func HostCarch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64"
	case "386":
		return "i686"
	case "arm64":
		return "aarch64"
	case "arm":
		return "armv7h"
	case "riscv64":
		return "riscv64"
	}
	return runtime.GOARCH
}

func newTinyLuaState() *lua.LState {
	L := lua.NewState(lua.Options{SkipOpenLibs: true})
	for _, lib := range []struct {
		name string
		fn   lua.LGFunction
	}{{lua.BaseLibName, lua.OpenBase}, {lua.TabLibName, lua.OpenTable}, {lua.StringLibName, lua.OpenString}, {lua.MathLibName, lua.OpenMath}} {
		L.Push(L.NewFunction(lib.fn))
		L.Push(lua.LString(lib.name))
		L.Call(1, 0)
	}
	for _, name := range []string{"dofile", "loadfile", "load", "loadstring", "require", "module", "collectgarbage", "getfenv", "setfenv"} {
		L.SetGlobal(name, lua.LNil)
	}
	return L
}

func LoadTinyRecipe(path, carch string) (*TinyRecipe, error) {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		path = filepath.Join(path, "tiny.lua")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if carch == "" {
		carch = HostCarch()
	}
	L := newTinyLuaState()
	fn, err := L.LoadFile(abs)
	if err != nil {
		L.Close()
		return nil, err
	}
	L.Push(fn)
	if err := L.PCall(0, 1, nil); err != nil {
		L.Close()
		return nil, err
	}
	tbl, ok := L.Get(-1).(*lua.LTable)
	L.Pop(1)
	if !ok {
		L.Close()
		return nil, fmt.Errorf("%s: tiny.lua must return a table", abs)
	}
	r := &TinyRecipe{File: abs, Dir: filepath.Dir(abs), Carch: carch, L: L, Table: tbl}
	if err := r.read(); err != nil {
		L.Close()
		return nil, fmt.Errorf("%s: %w", abs, err)
	}
	return r, nil
}

func (r *TinyRecipe) Close() {
	if r.L != nil {
		r.L.Close()
	}
}

func (r *TinyRecipe) field(key string) lua.LValue {
	if overrides, ok := r.Table.RawGetString("arch_overrides").(*lua.LTable); ok {
		if o, ok := overrides.RawGetString(r.Carch).(*lua.LTable); ok {
			if v := o.RawGetString(key); v != lua.LNil {
				return v
			}
		}
	}
	return r.Table.RawGetString(key)
}

func (r *TinyRecipe) expand(s string) string {
	return os.Expand(s, func(name string) string {
		switch name {
		case "name", "pkgname":
			return r.Name
		case "version", "pkgver":
			return r.Version
		case "release", "pkgrel":
			return r.Release
		case "carch", "CARCH":
			return r.Carch
		}
		if v, ok := r.Table.RawGetString(name).(lua.LString); ok {
			return string(v)
		}
		return "${" + name + "}"
	})
}

func luaScalar(v lua.LValue) (string, bool) {
	switch x := v.(type) {
	case lua.LString:
		return string(x), true
	case lua.LNumber:
		return strconv.FormatFloat(float64(x), 'f', -1, 64), true
	case lua.LBool:
		if x {
			return "true", true
		}
		return "false", true
	}
	return "", false
}

func (r *TinyRecipe) str(key string) string {
	v := r.field(key)
	if s, ok := luaScalar(v); ok {
		return r.expand(s)
	}
	return ""
}

func (r *TinyRecipe) listOf(v lua.LValue) []string {
	var out []string
	switch x := v.(type) {
	case *lua.LTable:
		x.ForEach(func(k, item lua.LValue) {
			if _, isIndex := k.(lua.LNumber); !isIndex {
				return
			}
			if s, ok := luaScalar(item); ok {
				out = append(out, r.expand(s))
			}
		})
	default:
		if s, ok := luaScalar(v); ok && s != "" {
			out = append(out, r.expand(s))
		}
	}
	return out
}

func (r *TinyRecipe) list(key string) []string {
	return r.listOf(r.field(key))
}

func (r *TinyRecipe) sub(key string) *lua.LTable {
	t, _ := r.field(key).(*lua.LTable)
	return t
}

func (r *TinyRecipe) read() error {
	r.Name = r.str("name")
	r.Version = r.str("version")
	r.Release = r.str("release")
	if r.Release == "" {
		r.Release = "1"
	}
	if r.Name == "" || r.Version == "" {
		return fmt.Errorf("name and version are required")
	}
	if strings.ContainsAny(r.Version, "-/: ") {
		return fmt.Errorf("version %q may not contain '-', '/', ':' or spaces", r.Version)
	}
	r.Epoch = r.str("epoch")
	r.Desc = r.str("desc")
	r.URL = r.str("url")
	r.License = r.list("license")
	r.Arch = r.list("arch")
	r.Groups = r.list("groups")
	r.Source = r.list("source")
	r.SHA256 = r.list("sha256")
	r.SHA512 = r.list("sha512")
	r.B2 = r.list("b2")
	r.NoExtract = r.list("noextract")
	r.Depends = r.list("depends")
	r.OptDepends = r.list("optdepends")
	r.MakeDepends = r.list("makedepends")
	r.LibDepends = r.list("libdepends")
	r.Provides = r.list("provides")
	r.Conflicts = r.list("conflicts")
	r.Replaces = r.list("replaces")
	r.Backup = r.list("backup")
	r.Options = r.list("options")
	r.Subpackages = r.list("subpackages")
	r.Patches = r.list("patches")
	r.InstallLicense = r.list("install_license")
	r.Style = r.str("style")
	r.BuildDir = r.str("build_dir")
	r.Configure = r.list("configure")
	r.MakeArgs = r.list("make_args")
	r.InstallArgs = r.list("install_args")
	r.Cross = r.str("cross")
	if r.Cross == "" {
		r.Cross = "yes"
	}
	if r.Style == "" {
		if r.Table.RawGetString("build") != lua.LNil {
			r.Style = "custom"
		} else {
			r.Style = "copy"
		}
	}
	if !tinyStyles[r.Style] {
		return fmt.Errorf("unknown style %q", r.Style)
	}
	for name, sums := range map[string][]string{"sha256": r.SHA256, "sha512": r.SHA512, "b2": r.B2} {
		if len(sums) != 0 && len(sums) != len(r.Source) {
			return fmt.Errorf("%s has %d entries, but source has %d", name, len(sums), len(r.Source))
		}
	}
	if len(r.Arch) > 0 && !r.SupportsArch(r.Carch) && os.Getenv("IGNOREARCH") != "1" {
		return fmt.Errorf("%s does not support %s (arch = %s)", r.Name, r.Carch, strings.Join(r.Arch, " "))
	}
	if g := r.sub("go"); g != nil {
		r.GoPackages = r.listOf(g.RawGetString("packages"))
		r.GoLDFlags = r.listOf(g.RawGetString("ldflags"))
		r.GoFlags = r.listOf(g.RawGetString("flags"))
		if s, ok := luaScalar(g.RawGetString("cgo")); ok {
			r.GoCGO = map[string]string{"true": "1", "false": "0"}[s]
			if r.GoCGO == "" {
				r.GoCGO = s
			}
		}
		r.GoEnv = r.mapOf(g.RawGetString("env"))
	}
	if c := r.sub("cargo"); c != nil {
		r.CargoFeatures = r.listOf(c.RawGetString("features"))
	}
	if s := r.sub("slay"); s != nil {
		s.ForEach(func(k, v lua.LValue) {
			name, ok := k.(lua.LString)
			if !ok {
				return
			}
			if b, isBool := v.(lua.LBool); isBool {
				if b {
					r.SlayArgs = append(r.SlayArgs, string(name))
				}
				return
			}
			if val, ok := luaScalar(v); ok {
				r.SlayArgs = append(r.SlayArgs, string(name)+"="+r.expand(val))
			}
		})
	}
	r.Env = r.mapOf(r.field("env"))
	if t := r.sub("install"); t != nil {
		var err error
		t.ForEach(func(_, item lua.LValue) {
			if err != nil {
				return
			}
			spec, ok := item.(*lua.LTable)
			if !ok {
				err = fmt.Errorf("install entries must be tables like { \"src\", \"dest\", mode = 644 }")
				return
			}
			src, _ := luaScalar(spec.RawGetInt(1))
			dest, _ := luaScalar(spec.RawGetInt(2))
			mode := os.FileMode(0o644)
			if m, ok := luaScalar(spec.RawGetString("mode")); ok {
				parsed, perr := strconv.ParseUint(m, 8, 32)
				if perr != nil {
					err = fmt.Errorf("bad install mode %q", m)
					return
				}
				mode = os.FileMode(parsed)
			}
			r.Install = append(r.Install, TinyInstall{Src: r.expand(src), Dest: r.expand(dest), Mode: mode})
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *TinyRecipe) mapOf(v lua.LValue) map[string]string {
	t, ok := v.(*lua.LTable)
	if !ok {
		return nil
	}
	m := map[string]string{}
	t.ForEach(func(k, val lua.LValue) {
		key, ok := k.(lua.LString)
		if !ok {
			return
		}
		if s, ok := luaScalar(val); ok {
			m[string(key)] = r.expand(s)
		}
	})
	return m
}

func (r *TinyRecipe) SupportsArch(carch string) bool {
	for _, a := range r.Arch {
		if a == carch || a == "any" {
			return true
		}
	}
	return false
}

func (r *TinyRecipe) FullVersion() string {
	v := r.Version + "-" + r.Release
	if r.Epoch != "" && r.Epoch != "0" {
		v = r.Epoch + ":" + v
	}
	return v
}

func (r *TinyRecipe) PkgArch() string {
	if len(r.Arch) == 1 && r.Arch[0] == "any" {
		return "any"
	}
	return r.Carch
}

func (r *TinyRecipe) Function(name string) *lua.LFunction {
	if f, ok := r.field(name).(*lua.LFunction); ok {
		return f
	}
	return nil
}

func (r *TinyRecipe) HasOption(name string, def bool) bool {
	for _, o := range r.Options {
		if o == name {
			return true
		}
		if o == "!"+name {
			return false
		}
	}
	return def
}
