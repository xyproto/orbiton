package main

import "testing"

var pyerror = `
Traceback (most recent call last):
  File "/usr/lib/python3.8/py_compile.py", line 144, in compile
    code = loader.source_to_code(source_bytes, dfile or file,
  File "<frozen importlib._bootstrap_external>", line 846, in source_to_code
  File "<frozen importlib._bootstrap>", line 219, in _call_with_frames_removed
  File "main.py", line 8
    def
      ^
SyntaxError: invalid syntax

During handling of the above exception, another exception occurred:

Traceback (most recent call last):
  File "/usr/lib/python3.8/py_compile.py", line 209, in main
    compile(filename, doraise=True)
  File "/usr/lib/python3.8/py_compile.py", line 150, in compile
    raise py_exc
__main__.PyCompileError:   File "main.py", line 8
    def
      ^
SyntaxError: invalid syntax


During handling of the above exception, another exception occurred:

Traceback (most recent call last):
  File "/usr/lib/python3.8/runpy.py", line 193, in _run_module_as_main
    return _run_code(code, main_globals, None,
  File "/usr/lib/python3.8/runpy.py", line 86, in _run_code
    exec(code, run_globals)
  File "/usr/lib/python3.8/py_compile.py", line 218, in <module>
    sys.exit(main())
  File "/usr/lib/python3.8/py_compile.py", line 213, in main
    if quiet < 2:
NameError: name 'quiet' is not defined
`

func TestParsePythonError(t *testing.T) {
	lineNumber, columnNumber, errorMessage := ParsePythonError(pyerror, "main.py")
	if lineNumber != 8 {
		t.Fatalf("line number should be 8, but is %d\n", lineNumber)
	}
	if columnNumber != 3 {
		t.Fatalf("column number should be 3, but is %d\n", columnNumber)
	}
	if errorMessage != "invalid syntax" {
		t.Fail()
	}
}
