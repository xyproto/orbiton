package main

// TODO: Use a different syntax highlighting package, with support for many different programming languages
import "github.com/xyproto/syntax"

var (
	// Based on /usr/share/nvim/runtime/syntax/cmake.vim
	cmakeWords = []string{"add_compile_options", "add_custom_command", "add_custom_target", "add_definitions", "add_dependencies", "add_executable", "add_library", "add_subdirectory", "add_test", "build_command", "build_name", "cmake_host_system_information", "cmake_minimum_required", "cmake_parse_arguments", "cmake_policy", "configure_file", "create_test_sourcelist", "ctest_build", "ctest_configure", "ctest_coverage", "ctest_memcheck", "ctest_run_script", "ctest_start", "ctest_submit", "ctest_test", "ctest_update", "ctest_upload", "define_property", "enable_language", "exec_program", "execute_process", "export", "export_library_dependencies", "file", "find_file", "find_library", "find_package", "find_path", "find_program", "fltk_wrap_ui", "foreach", "function", "get_cmake_property", "get_directory_property", "get_filename_component", "get_property", "get_source_file_property", "get_target_property", "get_test_property", "if", "include", "include_directories", "include_external_msproject", "include_guard", "install", "install_files", "install_programs", "install_targets", "list", "load_cache", "load_command", "macro", "make_directory", "mark_as_advanced", "math", "message", "option", "project", "remove", "separate_arguments", "set", "set_directory_properties", "set_package_properties", "set_property", "set_source_files_properties", "set_target_properties", "set_tests_properties", "source_group", "string", "subdirs", "target_compile_definitions", "target_compile_features", "target_compile_options", "target_include_directories", "target_link_libraries", "target_sources", "try_compile", "try_run", "unset", "use_mangled_mesa", "variable_requires", "variable_watch", "while", "write_file"}

	emacsWords = []string{"defun", "require", "if", "when", "setq", "add-to-list", "lambda", "defvar", "defconst", "let", "nil", "load"} // this should do it

	// Based on /usr/share/nvim/runtime/syntax/zig.vim
	zigWords = []string{"const", "var", "extern", "packed", "export", "pub", "noalias", "inline", "noinline", "comptime", "callconv", "volatile", "allowzero", "align", "linksection", "threadlocal", "struct", "enum", "union", "error", "break", "return", "continue", "asm", "defer", "errdefer", "unreachable", "try", "catch", "async", "nosuspend", "await", "suspend", "resume", "if", "else", "switch", "and", "or", "orelse", "while", "for", "null", "undefined", "fn", "usingnamespace", "test", "bool", "f16", "f32", "f64", "f128", "void", "noreturn", "type", "anyerror", "anyframe", "i0", "u0", "isize", "usize", "comptime_int", "comptime_float", "c_short", "c_ushort", "c_int", "c_uint", "c_long", "c_ulong", "c_longlong", "c_ulonglong", "c_longdouble", "c_void", "true", "false", "addWithOverflow", "as", "atomicLoad", "atomicStore", "bitCast", "breakpoint", "alignCast", "alignOf", "cDefine", "cImport", "cInclude", "cUndef", "canImplicitCast", "clz", "cmpxchgWeak", "cmpxchgStrong", "compileError", "compileLog", "ctz", "popCount", "divExact", "divFloor", "divTrunc", "embedFile", "export", "tagName", "TagType", "errorName", "call", "errorReturnTrace", "fence", "fieldParentPtr", "field", "unionInit", "frameAddress", "import", "newStackCall", "asyncCall", "intToPtr", "memcpy", "memset", "mod", "mulWithOverflow", "splat", "bitOffsetOf", "byteOffsetOf", "OpaqueType", "panic", "ptrCast", "ptrToInt", "rem", "returnAddress", "setCold", "Type", "shuffle", "setRuntimeSafety", "setEvalBranchQuota", "setFloatMode", "setGlobalLinkage", "setGlobalSection", "shlExact", "This", "hasDecl", "hasField", "shlWithOverflow", "shrExact", "sizeOf", "bitSizeOf", "sqrt", "byteSwap", "subWithOverflow", "intCast", "floatCast", "intToFloat", "floatToInt", "boolToInt", "errSetCast", "truncate", "typeInfo", "typeName", "TypeOf", "atomicRmw", "bytesToSlice", "sliceToBytes", "intToError", "errorToInt", "intToEnum", "enumToInt", "setAlignStack", "frame", "Frame", "frameSize", "bitReverse", "Vector", "sin", "cos", "exp", "exp2", "log", "log2", "log10", "fabs", "floor", "ceil", "trunc", "round"}

	kotlinWords = []string{"as", "break", "class", "continue", "do", "else", "false", "for", "fun", "if", "in", "interface", "is", "null", "object", "package", "return", "super", "this", "throw", "true", "try", "typealias", "typeof", "val", "var", "when", "while"}

	// From https://source.android.com/devices/architecture/hidl
	hidlWords = []string{"safe_union", "struct", "union", "enum", "typedef", "generates", "package", "interface", "extends", "import", "constexpr", "oneway"}
)

// adjustSyntaxHighlightingKeywords contains per-language adjustments to highlighting of keywords
func adjustSyntaxHighlightingKeywords(mode Mode) {
	var addKeywords, delKeywords []string
	switch mode {
	case modeGo:
		addKeywords = []string{"go", "fallthrough", "string", "print", "println", "range", "defer"}
		delKeywords = []string{"mut", "pass", "build", "None", "char", "get", "set"}
	case modeLisp:
		syntax.Keywords = make(map[string]struct{})
		addKeywords = emacsWords
	case modeCMake:
		addKeywords = cmakeWords
		delKeywords = append(delKeywords, []string{"build", "package"}...)
	case modeZig:
		syntax.Keywords = make(map[string]struct{})
		addKeywords = zigWords
	case modeKotlin:
		syntax.Keywords = make(map[string]struct{})
		addKeywords = kotlinWords
	case modeJava:
		addKeywords = []string{"package"}
	case modeHIDL:
		syntax.Keywords = make(map[string]struct{})
		addKeywords = hidlWords
	case modeShell:
		delKeywords = []string{"float", "with", "exec", "long", "double", "no", "pass", "#else", "#endif", "ret", "super"}
		addKeywords = []string{"-f", "--force"}
		fallthrough // to the default case
	default:
		delKeywords = append(delKeywords, []string{"require", "build", "package", "super"}...)
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
	case modeShell, modePython, modeCMake, modeConfig:
		return "#"
	case modeAssembly:
		return ";"
	case modeHaskell:
		return "--"
	case modeVim:
		return "\""
	case modeLisp:
		return ";;"
	default:
		return "//"
	}
}
