//go:build !tinygo && js && wasm
// +build !tinygo,js,wasm

package gfx

import "syscall/js"

// JS gives access to js.Global and js.TypedArrayOf
var JS = JavaScript{
	Global: js.Global,
}

// JavaScript is a type that contains fields with Global and TypedArrayOf funcs.
type JavaScript struct {
	Global func() js.Value
}

func (j JavaScript) Document() js.Value {
	return j.Global().Get("document")
}

// Body returns the js.Value for document.body
// The (first) optional innerHTML argument can be used to set body.innerHTML.
func (j JavaScript) Body(innerHTML ...string) js.Value {
	body := j.Document().Get("body")

	if len(innerHTML) > 0 {
		body.Set("innerHTML", innerHTML[0])
	}

	return body
}
