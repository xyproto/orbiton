package slay

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

func (b *TinyBuilder) callLua(f *lua.LFunction, bt *lua.LTable, args ...lua.LValue) error {
	return b.R.L.CallByParam(lua.P{Fn: f, NRet: 0, Protect: true}, append([]lua.LValue{bt}, args...)...)
}

func luaArgs(L *lua.LState, start int) []string {
	var out []string
	var add func(v lua.LValue)
	add = func(v lua.LValue) {
		switch x := v.(type) {
		case *lua.LTable:
			x.ForEach(func(k, item lua.LValue) {
				if _, ok := k.(lua.LNumber); ok {
					add(item)
				}
			})
		default:
			if s, ok := luaScalar(v); ok {
				out = append(out, s)
			}
		}
	}
	for i := start; i <= L.GetTop(); i++ {
		add(L.Get(i))
	}
	return out
}

func (b *TinyBuilder) luaBuilder() *lua.LTable {
	L := b.R.L
	t := L.NewTable()
	set := func(name string, v lua.LValue) { t.RawSetString(name, v) }
	fail := func(L *lua.LState, err error) int {
		if err != nil {
			L.RaiseError("%s", err.Error())
		}
		return 0
	}
	fn := func(name string, f func(L *lua.LState) int) { set(name, L.NewFunction(f)) }
	strs := func(list []string) *lua.LTable {
		tb := L.NewTable()
		for _, s := range list {
			tb.Append(lua.LString(s))
		}
		return tb
	}
	set("carch", lua.LString(b.R.Carch))
	set("cross", lua.LBool(b.Cross))
	set("sysroot", lua.LString(b.Sysroot))
	set("srcdir", lua.LString(b.SrcDir))
	set("pkgdir", lua.LString(b.PkgDir))
	set("name", lua.LString(b.R.Name))
	set("version", lua.LString(b.R.Version))
	set("release", lua.LString(b.R.Release))
	set("triple", lua.LString(b.T.Triple))
	set("host_triple", lua.LString(b.HostT.Triple))
	set("jobs", lua.LNumber(b.Jobs))
	set("configure", strs(b.R.Configure))
	fn("sh", func(L *lua.LState) int {
		args := luaArgs(L, 1)
		if len(args) == 0 {
			return 0
		}
		var script strings.Builder
		script.WriteString(args[0])
		for _, a := range args[1:] {
			script.WriteString(" " + shellQuote(a))
		}
		return fail(L, b.sh(false, script.String()))
	})
	fn("run", func(L *lua.LState) int {
		return fail(L, b.run(false, luaArgs(L, 1)...))
	})
	fn("cd", func(L *lua.LState) int {
		dir := L.CheckString(1)
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(b.cwd, dir)
		}
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			return fail(L, fmt.Errorf("cd: %s is not a directory", dir))
		}
		b.cwd = dir
		return 0
	})
	fn("env", func(L *lua.LState) int {
		b.extraEnv[L.CheckString(1)] = L.CheckString(2)
		return 0
	})
	fn("exists", func(L *lua.LState) int {
		p := L.CheckString(1)
		if !filepath.IsAbs(p) {
			p = filepath.Join(b.cwd, p)
		}
		_, err := os.Lstat(p)
		L.Push(lua.LBool(err == nil))
		return 1
	})
	fn("glob", func(L *lua.LState) int {
		p := L.CheckString(1)
		if !filepath.IsAbs(p) {
			p = filepath.Join(b.cwd, p)
		}
		m, _ := filepath.Glob(p)
		L.Push(strs(m))
		return 1
	})
	fn("patch", func(L *lua.LState) int {
		return fail(L, b.patch(L.CheckString(1)))
	})
	fn("install", func(L *lua.LState) int {
		mode := os.FileMode(0o644)
		if L.GetTop() >= 3 {
			s, _ := luaScalar(L.Get(3))
			m, err := strconv.ParseUint(s, 8, 32)
			if err != nil {
				return fail(L, fmt.Errorf("install: bad mode %q", s))
			}
			mode = os.FileMode(m)
		}
		return fail(L, b.install(L.CheckString(1), L.CheckString(2), mode))
	})
	fn("emulate", func(L *lua.LState) int {
		f := L.CheckFunction(1)
		if !b.Cross {
			return fail(L, L.CallByParam(lua.P{Fn: f, NRet: 0, Protect: true}))
		}
		if b.emulator() == nil {
			return fail(L, fmt.Errorf("b.emulate needs %s and binfmt support to run %s programs", b.T.Qemu, b.R.Carch))
		}
		b.emulating = true
		err := L.CallByParam(lua.P{Fn: f, NRet: 0, Protect: true})
		b.emulating = false
		return fail(L, err)
	})
	style := func(name string) func(L *lua.LState) int {
		return func(L *lua.LState) int {
			saved := b.R.Style
			b.R.Style = name
			if extra := luaArgs(L, 1); len(extra) > 0 {
				switch name {
				case "meson", "cmake", "configure":
					saved := b.R.Configure
					b.R.Configure = append(append([]string{}, b.R.Configure...), extra...)
					defer func() { b.R.Configure = saved }()
				case "make":
					saved := b.R.MakeArgs
					b.R.MakeArgs = append(append([]string{}, b.R.MakeArgs...), extra...)
					defer func() { b.R.MakeArgs = saved }()
				case "slay":
					saved := b.R.SlayArgs
					b.R.SlayArgs = append(append([]string{}, b.R.SlayArgs...), extra...)
					defer func() { b.R.SlayArgs = saved }()
				case "go":
					saved := b.R.GoPackages
					b.R.GoPackages = extra
					defer func() { b.R.GoPackages = saved }()
				}
			}
			err := b.styleBuild()
			b.R.Style = saved
			return fail(L, err)
		}
	}
	for _, s := range []string{"meson", "cmake", "configure", "make", "go", "cargo", "slay", "python"} {
		fn(s, style(s))
	}
	fn("style_install", func(L *lua.LState) int {
		saved := b.R.Style
		if L.GetTop() >= 1 {
			b.R.Style = L.CheckString(1)
		}
		err := b.styleInstall()
		b.R.Style = saved
		return fail(L, err)
	})
	fn("configure_args", func(L *lua.LState) int {
		L.Push(strs(b.R.Configure))
		return 1
	})
	fn("log", func(L *lua.LState) int {
		b.logf("%s", strings.Join(luaArgs(L, 1), " "))
		return 0
	})
	return t
}
