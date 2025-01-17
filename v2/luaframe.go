package main

import (
	"strings"

	"github.com/xyproto/mode"
)

// LuaLove tries to figure out if the current source code looks like it is Lua code that uses LÖVE or not
func (e *Editor) LuaLove() bool {
	return e.mode == mode.Lua && strings.Contains(e.String(), "function love.draw(")
}

// LuaLovr tries to figure out if the current source code looks like it is Lua code that uses LÖVR or not
func (e *Editor) LuaLovr() bool {
	return e.mode == mode.Lua && strings.Contains(e.String(), "function lovr.draw(")
}

// LuaLoveOrLovr tries to figure out if the current source code looks like it is Lua code that uses LÖVE, LÖVR or neither
func (e *Editor) LuaLoveOrLovr() bool {
	sourceCode := e.String()
	return e.mode == mode.Lua && (strings.Contains(sourceCode, "function love.draw(") || strings.Contains(sourceCode, "function lovr.draw("))
}
