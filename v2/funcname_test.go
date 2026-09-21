package main

import (
	"testing"

	"github.com/xyproto/mode"
)

func TestLooksLikeFunctionDef(t *testing.T) {
	for _, tc := range []struct {
		m    mode.Mode
		line string
	}{
		{mode.Go, "func (e *Editor) Foo() {"},
		{mode.Go, "func main() {"},
		{mode.Rust, "pub fn foo() {"},
		{mode.Rust, "pub(crate) async fn foo() -> Result<(), E> {"},
		{mode.Rust, "    fn foo(&self) {"},
		{mode.Zig, "pub fn main() !void {"},
		{mode.Kotlin, "private fun foo() {"},
		{mode.Kotlin, "override fun toString(): String {"},
		{mode.C, "int main(void)"},
		{mode.C, "static void foo(int x) {"},
		{mode.Python, "async def foo():"},
		{mode.Python, "    def foo(self):"},
		{mode.JavaScript, "async function foo() {"},
		{mode.JavaScript, "export function foo() {"},
		{mode.TypeScript, "export async function foo(): Promise<void> {"},
		{mode.Shell, "foo() {"},
		{mode.Lua, "local function foo()"},
	} {
		e := NewSimpleEditor(80)
		e.mode = tc.m
		if !e.LooksLikeFunctionDef(tc.line, e.FuncPrefix()) {
			t.Errorf("%s: %q not recognized as a function definition", tc.m, tc.line)
		}
	}
	for _, tc := range []struct {
		m    mode.Mode
		line string
	}{
		{mode.Go, "f := func() {"},
		{mode.Go, "\treturn func() {"},
		{mode.Rust, "let x = foo(1);"},
		{mode.Rust, "// pub fn foo() {"},
		{mode.Python, "x = foo(1)"},
	} {
		e := NewSimpleEditor(80)
		e.mode = tc.m
		if e.LooksLikeFunctionDef(tc.line, e.FuncPrefix()) {
			t.Errorf("%s: %q wrongly recognized as a function definition", tc.m, tc.line)
		}
	}
}
