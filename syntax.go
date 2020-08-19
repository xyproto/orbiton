package main

// TODO: Use a different syntax highlighting package, with support for many different programming languages
import "github.com/xyproto/syntax"

var (
	// Based on /usr/share/nvim/runtime/syntax/cmake.vim
	cmakeWords = []string{"add_compile_options", "add_custom_command", "add_custom_target", "add_definitions", "add_dependencies", "add_executable", "add_library", "add_subdirectory", "add_test", "build_command", "build_name", "cmake_host_system_information", "cmake_minimum_required", "cmake_parse_arguments", "cmake_policy", "configure_file", "create_test_sourcelist", "ctest_build", "ctest_configure", "ctest_coverage", "ctest_memcheck", "ctest_run_script", "ctest_start", "ctest_submit", "ctest_test", "ctest_update", "ctest_upload", "define_property", "enable_language", "exec_program", "execute_process", "export", "export_library_dependencies", "file", "find_file", "find_library", "find_package", "find_path", "find_program", "fltk_wrap_ui", "foreach", "function", "get_cmake_property", "get_directory_property", "get_filename_component", "get_property", "get_source_file_property", "get_target_property", "get_test_property", "if", "include", "include_directories", "include_external_msproject", "include_guard", "install", "install_files", "install_programs", "install_targets", "list", "load_cache", "load_command", "macro", "make_directory", "mark_as_advanced", "math", "message", "option", "project", "remove", "separate_arguments", "set", "set_directory_properties", "set_package_properties", "set_property", "set_source_files_properties", "set_target_properties", "set_tests_properties", "source_group", "string", "subdirs", "target_compile_definitions", "target_compile_features", "target_compile_options", "target_include_directories", "target_link_libraries", "target_sources", "try_compile", "try_run", "unset", "use_mangled_mesa", "variable_requires", "variable_watch", "while", "write_file"}

	emacsWords = []string{"add-to-list", "defconst", "defun", "defvar", "if", "lambda", "let", "load", "nil", "require", "setq", "when"} // this should do it

	// Based on /usr/share/nvim/runtime/syntax/zig.vim
	zigWords = []string{"Frame", "OpaqueType", "TagType", "This", "Type", "TypeOf", "Vector", "addWithOverflow", "align", "alignCast", "alignOf", "allowzero", "and", "anyerror", "anyframe", "as", "asm", "async", "asyncCall", "atomicLoad", "atomicRmw", "atomicStore", "await", "bitCast", "bitOffsetOf", "bitReverse", "bitSizeOf", "bool", "boolToInt", "break", "breakpoint", "byteOffsetOf", "byteSwap", "bytesToSlice", "cDefine", "cImport", "cInclude", "cUndef", "c_int", "c_long", "c_longdouble", "c_longlong", "c_short", "c_uint", "c_ulong", "c_ulonglong", "c_ushort", "c_void", "call", "callconv", "canImplicitCast", "catch", "ceil", "clz", "cmpxchgStrong", "cmpxchgWeak", "compileError", "compileLog", "comptime", "comptime_float", "comptime_int", "const", "continue", "cos", "ctz", "defer", "divExact", "divFloor", "divTrunc", "else", "embedFile", "enum", "enumToInt", "errSetCast", "errdefer", "error", "errorName", "errorReturnTrace", "errorToInt", "exp", "exp2", "export", "export", "extern", "f128", "f16", "f32", "f64", "fabs", "false", "fence", "field", "fieldParentPtr", "floatCast", "floatToInt", "floor", "fn", "for", "frame", "frameAddress", "frameSize", "hasDecl", "hasField", "i0", "if", "import", "inline", "intCast", "intToEnum", "intToError", "intToFloat", "intToPtr", "isize", "linksection", "log", "log10", "log2", "memcpy", "memset", "mod", "mulWithOverflow", "newStackCall", "noalias", "noinline", "noreturn", "nosuspend", "null", "or", "orelse", "packed", "panic", "popCount", "ptrCast", "ptrToInt", "pub", "rem", "resume", "return", "returnAddress", "round", "setAlignStack", "setCold", "setEvalBranchQuota", "setFloatMode", "setGlobalLinkage", "setGlobalSection", "setRuntimeSafety", "shlExact", "shlWithOverflow", "shrExact", "shuffle", "sin", "sizeOf", "sliceToBytes", "splat", "sqrt", "struct", "subWithOverflow", "suspend", "switch", "tagName", "test", "threadlocal", "true", "trunc", "truncate", "try", "type", "typeInfo", "typeName", "u0", "undefined", "union", "unionInit", "unreachable", "usingnamespace", "usize", "var", "void", "volatile", "while"}

	kotlinWords = []string{"as", "break", "catch", "class", "continue", "do", "else", "false", "for", "fun", "if", "import", "in", "interface", "is", "it", "null", "object", "override", "package", "return", "super", "this", "throw", "true", "try", "typealias", "typeof", "val", "var", "when", "while"}

	// From https://source.android.com/devices/architecture/hidl
	hidlWords = []string{"constexpr", "enum", "extends", "generates", "import", "interface", "oneway", "package", "safe_union", "struct", "typedef", "union"}

	luaWords = []string{"and", "break", "do", "else", "elseif", "end", "false", "for", "function", "goto", "if", "in", "local", "nil", "not", "or", "repeat", "return", "then", "true", "until", "while"}
)

// adjustSyntaxHighlightingKeywords contains per-language adjustments to highlighting of keywords
func adjustSyntaxHighlightingKeywords(mode Mode) {
	var addKeywords, delKeywords []string
	switch mode {
	case modeGo:
		addKeywords = []string{"defer", "fallthrough", "go", "print", "println", "range", "string"}
		delKeywords = []string{"None", "build", "char", "get", "mut", "pass", "set"}
	case modeLisp:
		syntax.Keywords = make(map[string]struct{})
		addKeywords = emacsWords
	case modeCMake:
		addKeywords = cmakeWords
		delKeywords = append(delKeywords, []string{"build", "package"}...)
	case modeZig:
		syntax.Keywords = make(map[string]struct{})
		addKeywords = zigWords
	case modeVim:
		addKeywords = []string{"call", "echo", "elseif", "endfunction", "map", "nmap", "redraw"}
	case modeKotlin:
		syntax.Keywords = make(map[string]struct{})
		addKeywords = kotlinWords
	case modeJava:
		addKeywords = []string{"package"}
	case modeHIDL:
		syntax.Keywords = make(map[string]struct{})
		addKeywords = hidlWords
	case modeSQL:
		addKeywords = []string{"NOT"}
	case modeOak:
		delKeywords = []string{"from", "new", "print"}
		addKeywords = []string{"fn"}
	case modeRust:
		addKeywords = []string{"fn"}
	case modeLua:
		syntax.Keywords = make(map[string]struct{})
		addKeywords = luaWords
	case modeShell:
		delKeywords = []string{"#else", "#endif", "default", "double", "exec", "float", "install", "long", "no", "pass", "ret", "super", "var", "with"}
		addKeywords = []string{"--force", "-f", "cmake", "configure", "fdisk", "gdisk", "make", "mv", "ninja", "rm", "rmdir"}
		fallthrough // to the default case
	default:
		delKeywords = append(delKeywords, []string{"require", "build", "package", "super", "type"}...)
	}
	// Add extra keywords that are to be syntax highlighted
	for _, kw := range addKeywords {
		syntax.Keywords[kw] = struct{}{}
	}
	// Remove keywords that should not be syntax highlighted
	for _, kw := range delKeywords {
		delete(syntax.Keywords, kw)
	}
}

// SingleLineCommentMarker will return the string that starts a single-line
// comment for the current language mode the editor is in.
func (e *Editor) SingleLineCommentMarker() string {
	switch e.mode {
	case modeShell, modePython, modeCMake, modeConfig, modeCrystal, modeNim:
		return "#"
	case modeAssembly:
		return ";"
	case modeHaskell, modeSQL, modeLua:
		return "--"
	case modeVim:
		return "\""
	case modeLisp:
		return ";;"
	case modeBat:
		return "@rem" // or rem or just ":" ...
	default:
		return "//"
	}
}
