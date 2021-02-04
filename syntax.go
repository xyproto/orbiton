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

	// From: https://selinuxproject.org/page/PolicyLanguage
	policyLanguageWords = []string{"alias", "allow", "and", "attribute", "attribute_role", "auditallow", "auditdeny", "bool", "category", "cfalse", "class", "clone", "common", "constrain", "ctrue", "default_range", "default_role", "default_type", "default_user", "dom", "domby", "dominance", "dontaudit", "else", "equals", "false", "filename", "filesystem", "fscon", "fs_use_task", "fs_use_trans", "fs_use_xattr", "genfscon", "h1", "h2", "high", "identifier", "if", "incomp", "inherits", "iomemcon", "ioportcon", "ipv4_addr", "ipv6_addr", "l1", "l2", "level", "low", "low_high", "mlsconstrain", "mlsvalidatetrans", "module", "netifcon", "neverallow", "nodecon", "not", "notequal", "number", "object_r", "optional", "or", "path", "pcidevicecon", "permissive", "pirqcon", "policycap", "portcon", "r1", "r2", "r3", "range", "range_transition", "require", "role", "roleattribute", "roles", "role_transition", "sameuser", "sensitivity", "sid", "source", "t1", "t2", "t3", "target", "true", "type", "typealias", "typeattribute", "typebounds", "type_change", "type_member", "types", "type_transition", "u1", "u2", "u3", "user", "validatetrans", "version_identifier", "xor"}

	luaWords = []string{"and", "break", "do", "else", "elseif", "end", "false", "for", "function", "goto", "if", "in", "local", "nil", "not", "or", "repeat", "return", "then", "true", "until", "while"}
)

// adjustSyntaxHighlightingKeywords contains per-language adjustments to highlighting of keywords
func adjustSyntaxHighlightingKeywords(mode Mode) {
	var addKeywords, delKeywords []string
	switch mode {
	case modeGo, modeOdin:
		addKeywords = []string{"defer", "fallthrough", "go", "print", "println", "range", "string"}
		delKeywords = []string{"None", "build", "char", "get", "include", "mut", "pass", "set", "template", "then", "when", "where", "fi"}
	case modeLisp:
		syntax.Keywords = make(map[string]struct{})
		addKeywords = emacsWords
	case modeCMake:
		delKeywords = append(delKeywords, []string{"build", "package"}...)
		addKeywords = cmakeWords
	case modeZig:
		syntax.Keywords = make(map[string]struct{})
		addKeywords = zigWords
	case modeVim:
		addKeywords = []string{"call", "echo", "elseif", "endfunction", "map", "nmap", "redraw"}
	case modeKotlin:
		syntax.Keywords = make(map[string]struct{})
		addKeywords = kotlinWords
		delKeywords = []string{"it"}
	case modeJava:
		addKeywords = []string{"package"}
	case modeHIDL:
		syntax.Keywords = make(map[string]struct{})
		addKeywords = hidlWords
	case modePolicyLanguage: // SE Linux
		syntax.Keywords = make(map[string]struct{})
		addKeywords = policyLanguageWords
	case modeSQL:
		addKeywords = []string{"NOT"}
	case modeConfig:
		delKeywords = []string{"install"}
	case modeOak:
		addKeywords = []string{"fn"}
		delKeywords = []string{"from", "new", "print"}
	case modeRust:
		addKeywords = []string{"assert_eq", "fn", "impl", "loop", "mod", "out", "panic", "usize", "i64", "i32", "i16", "u64", "u32", "u16", "String", "char"}
		delKeywords = []string{"build", "done", "end", "next", "int64", "uint64", "int32", "uint32", "int16", "uint16", "int", "get", "print", "last"}
	case modeLua:
		syntax.Keywords = make(map[string]struct{})
		addKeywords = luaWords
	case modeNroff:
		syntax.Keywords = make(map[string]struct{})
		addKeywords = []string{"B", "BR", "PP", "SH", "TP", "fB", "fP"}
	case modeShell:
		addKeywords = []string{"--force", "-f", "cmake", "configure", "do", "fdisk", "for", "gdisk", "in", "make", "mv", "ninja", "rm", "rmdir", "while"}
		delKeywords = []string{"#else", "#endif", "default", "double", "exec", "float", "install", "long", "no", "pass", "ret", "super", "var", "with"}
		fallthrough // to the default case
	default:
		delKeywords = append(delKeywords, []string{"require", "build", "package", "super", "type", "set"}...)
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
	case modeShell, modePython, modeCMake, modeConfig, modeCrystal, modeNim, modePolicyLanguage:
		return "#"
	case modeAssembly:
		return ";"
	case modeHaskell, modeSQL, modeLua, modeAda:
		return "--"
	case modeVim:
		return "\""
	case modeLisp:
		return ";;"
	case modeBat:
		return "@rem" // or rem or just ":" ...
	case modeNroff:
		return `.\"`
	default:
		return "//"
	}
}
