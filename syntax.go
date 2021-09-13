package main

// TODO: Use a different syntax highlighting package, with support for many different programming languages
import "github.com/xyproto/syntax"

var (
	battlestarWords = []string{"address", "asm", "bootable", "break", "call", "chr", "const", "continue", "counter", "end", "exit", "extern", "fun", "funparam", "halt", "int", "len", "loop", "loopwrite", "mem", "membyte", "memdouble", "memword", "noret", "print", "rawloop", "read", "readbyte", "readdouble", "readword", "ret", "syscall", "sysparam", "use", "value", "var", "write"}

	clojureWords = []string{"*1", "*2", "*3", "*agent*", "*clojure-version*", "*command-line-args*", "*compile-files*", "*compile-path*", "*e", "*err*", "*file*", "*in*", "*ns*", "*out*", "*print-dup*", "*print-length*", "*print-level*", "*print-meta*", "*print-readably*", "*warn on reflection*", "accessor", "aclone", "add-watch", "agent", "agent-error", "agent-errors", "aget", "alength", "alias", "all-ns", "alter", "alter-meta!", "alter-var-root", "amap", "ancestors", "and", "apply", "areduce", "array-map", "as->", "aset", "aset-boolean", "aset-byte", "aset-char", "aset-double", "aset-float", "aset-int", "aset-long", "aset-short", "assert", "assoc", "assoc", "assoc", "assoc!", "assoc-in", "associative?", "atom", "await", "await-for", "bases", "bean", "bigdec", "bigdec?", "bigint", "binding", "bit-and", "bit-and-not", "bit-clear", "bit-flip", "bit-not", "bit-or", "bit-set", "bit-shift-left", "bit-shift-right", "bit-test", "bit-xor", "boolean", "boolean-array", "booleans", "bound-fn", "bound-fn*", "bound?", "butlast", "byte", "byte-array", "bytes", "case", "cast", "catch", "char", "char-array", "char?", "chars", "class", "class?", "clojure-version", "coll?", "commute", "comp", "comparator", "compare", "compare-and-set!", "compile", "complement", "concat", "cond", "cond->", "cond->>", "condp", "conj", "conj", "conj", "conj", "conj", "conj!", "cons", "constantly", "construct-proxy", "contains?", "count", "count", "counted?", "create-ns", "create-struct", "cycle", "dec", "decimal?", "declare", "dedupe", "def", "definline", "defmacro", "defmoethod", "defmulti", "defn", "defonce", "defprotocol", "defrecord", "defstruct", "deftype", "delay", "delay?", "deliver", "denominator", "deref", "deref", "derive", "descendants", "disj", "disj!", "dissoc", "dissoc!", "distinct", "distinct?", "do", "eval", "doall", "doall", "dorun", "dorun", "doseq", "doseq", "dosync", "dotimes", "doto", "double", "double-array", "double?", "doubles", "drop", "drop-last", "drop-while", "eduction", "empty", "empty?", "ensure", "enumeration-seq", "error-handler", "error-mode", "even?", "every-pred", "every?", "extend", "extend-protocol", "extend-type", "extenders", "extends?", "false?", "ffirst", "file-seq", "filter", "filterv", "finally", "find", "find-ns", "find-var", "first", "first", "flatten", "float", "float-array", "float?", "floats", "flush", "fn", "fn?", "fnext", "fnil", "for", "for", "force", "format", "frequencies", "future", "future-call", "future-cancel", "future-cancelled?", "future-done?", "future?", "gen-class", "gen-interface", "gensym", "gensym", "get", "get", "get", "get", "get", "get-in", "get-method", "get-proxy-class", "get-thread-bindings", "get-validator", "group-by", "hash", "hash-map", "hash-set", "ident?", "identical?", "identity", "if", "if-let", "if-not", "if-some", "ifn?", "import", "in-ns", "inc", "init-proxy", "instance?", "int", "int-array", "int?", "integer?", "interleave", "intern", "intern", "interpose", "into", "into-array", "ints", "io!", "isa?", "isa?", "iterate", "iterate", "iterator-seq", "juxt", "keep", "keep-indexed", "key", "keys", "keyword", "keyword?", "last", "lazy-cat", "lazy-cat", "lazy-seq", "lazy-seq", "let", "letfn", "line-seq", "list", "list?", "load", "load-file", "load-reader", "load-string", "loaded-libs", "locking", "long", "long-array", "longs", "loop", "macroexpand", "macroexpand-1", "make-array", "make-hierarchy", "map", "map-indexed", "map?", "mapcat", "mapv", "max", "max-key", "memfn", "memoize", "merge", "merge-with", "meta", "methods", "min", "min-key", "mod", "name", "namespace", "namespace-munge", "nat-int?", "neg?", "newline", "next", "nfirst", "nil?", "nnext", "non-empty", "not", "not", "not-any?", "not-every?", "ns", "ns-aliases", "ns-imports", "ns-interns", "ns-map", "ns-name", "ns-publics", "ns-refers", "ns-resolve", "ns-resolve", "ns-unalias", "ns-unmap", "nth", "nthnext", "nthrest", "num", "number?", "numerator", "object-array", "odd?", "or", "parents", "partial", "partition", "partition-all", "partition-by", "pcalls", "peek", "peek", "persistent!", "pmap", "pop", "pop", "pop!", "pop-thread-bindings", "pos-int?", "pos?", "pr", "pr-str", "pr-str", "prefer-method", "prefers", "print", "print-str", "print-str", "printf", "println", "println-str", "println-str", "prn", "prn-str", "prn-str", "promise", "proxy", "proxy-mappings", "proxy-super", "push-thread-bindings", "pvalues", "qualified-ident?", "qualified-keyword?", "qualified-symbol?", "quot", "rand", "rand-int", "rand-nth", "random-sample", "range", "ratio?", "rational?", "rationalize", "re-find", "re-groups", "re-matcher", "re-matches", "re-pattern", "re-seq", "read", "read-line", "read-string", "recur", "reduce", "reduce-kv", "reductions", "ref", "ref-history-count", "ref-max-history", "ref-min-history", "ref-set", "refer", "refer-clojure", "reify", "release-pending", "rem", "remove", "remove-all-methods", "remove-method", "remove-ns", "remove-watch", "repeat", "repeatedly", "repeatedly", "replace", "replicate", "require", "reset!", "reset-meta!", "resolve", "rest", "rest", "restart-agent", "resultset-seq", "reverse", "reversible?", "rseq", "rseq", "rsubseq", "satisfies?", "second", "select-keys", "send", "send-off", "seq", "seq?", "seqable?", "seque", "sequence", "sequential?", "set", "set", "set!", "set-error-handler", "set-error-mode", "set-validator", "set?", "short", "short-array", "shorts", "shuffle", "shutdonw-agents", "simple-ident?", "simple-keyword?", "simple-symbol?", "slurp", "some", "some->", "some->>", "some-fn", "sort", "sort-by", "sorted-map", "sorted-map-by", "sorted-set", "sorted-set-by", "sorted?", "special-symbol?", "spit", "split-at", "split-with", "str", "string?", "struct", "struct-map", "subs", "subseq", "subvec", "supers", "swap!", "symbol", "symbol?", "sync", "take", "take-last", "take-nth", "take-while", "test", "the-ns", "thread-bound?", "throw", "time", "to-array", "to-array-2d", "trampoline", "transduce", "transient", "tree-seq", "true?", "try", "type", "underive", "update", "update-in", "update-proxy", "use", "val", "vals", "var", "var-get", "var?", "vec", "vector", "vector-of", "vector?", "very-meta", "volatile!", "vreset!", "vswap!", "when", "when-first", "when-let", "when-not", "when-some", "while", "with-bindings", "with-bindings*", "with-in-str", "with-local-vars", "with-meta", "with-open", "with-out-str", "with-out-str", "with-precision", "xml-seq", "zero?", "zipmap"}

	// Based on /usr/share/nvim/runtime/syntax/cmake.vim
	cmakeWords = []string{"add_compile_options", "add_custom_command", "add_custom_target", "add_definitions", "add_dependencies", "add_executable", "add_library", "add_subdirectory", "add_test", "build_command", "build_name", "cmake_host_system_information", "cmake_minimum_required", "cmake_parse_arguments", "cmake_policy", "configure_file", "create_test_sourcelist", "ctest_build", "ctest_configure", "ctest_coverage", "ctest_memcheck", "ctest_run_script", "ctest_start", "ctest_submit", "ctest_test", "ctest_update", "ctest_upload", "define_property", "enable_language", "exec_program", "execute_process", "export", "export_library_dependencies", "file", "find_file", "find_library", "find_package", "find_path", "find_program", "fltk_wrap_ui", "foreach", "function", "get_cmake_property", "get_directory_property", "get_filename_component", "get_property", "get_source_file_property", "get_target_property", "get_test_property", "if", "include", "include_directories", "include_external_msproject", "include_guard", "install", "install_files", "install_programs", "install_targets", "list", "load_cache", "load_command", "macro", "make_directory", "mark_as_advanced", "math", "message", "option", "project", "remove", "separate_arguments", "set", "set_directory_properties", "set_package_properties", "set_property", "set_source_files_properties", "set_target_properties", "set_tests_properties", "source_group", "string", "subdirs", "target_compile_definitions", "target_compile_features", "target_compile_options", "target_include_directories", "target_link_libraries", "target_sources", "try_compile", "try_run", "unset", "use_mangled_mesa", "variable_requires", "variable_watch", "while", "write_file"}

	csWords = []string{"Boolean", "Byte", "Char", "Decimal", "Double", "Int16", "Int32", "Int64", "IntPtr", "Object", "Short", "Single", "String", "UInt16", "UInt32", "UInt64", "UIntPtr", "abstract", "as", "base", "bool", "break", "byte", "case", "catch", "char", "checked", "class", "const", "continue", "decimal", "default", "delegate", "do", "double", "dynamic", "else", "enum", "event", "explicit", "extern", "false", "finally", "fixed", "float", "for", "foreach", "goto", "if", "implicit", "in", "int", "interface", "internal", "is", "lock", "long", "namespace", "new", "nint", "nuint", "null", "object", "operator", "out", "override", "params", "readonly", "ref", "return", "sbyte", "sealed", "short", "sizeof", "stackalloc", "static", "string", "struct", "switch", "this", "throw", "true", "try", "typeof", "uint", "ulong", "unchecked", "unsafe", "ushort", "using", "virtual", "void", "volatile", "while"} // private, public, protected

	emacsWords = []string{"add-to-list", "defconst", "defun", "defvar", "if", "lambda", "let", "load", "nil", "require", "setq", "when"} // this should do it

	// From https://source.android.com/devices/architecture/hidl
	hidlWords = []string{"constexpr", "enum", "extends", "generates", "import", "interface", "oneway", "package", "safe_union", "struct", "typedef", "union"}

	kotlinWords = []string{"as", "break", "catch", "class", "continue", "do", "else", "false", "for", "fun", "if", "import", "in", "interface", "is", "null", "object", "override", "package", "return", "super", "this", "throw", "true", "try", "typealias", "typeof", "val", "var", "when", "while"}

	luaWords = []string{"and", "break", "do", "else", "elseif", "end", "false", "for", "function", "goto", "if", "in", "local", "nil", "not", "or", "repeat", "return", "then", "true", "until", "while"}

	odinWords = []string{"align_of", "auto_cast", "bit_field", "bit_set", "break", "case", "cast", "const", "context", "continue", "defer", "distinct", "do", "do", "dynamic", "else", "enum", "fallthrough", "for", "foreign", "if", "import", "in", "inline", "macro", "map", "no_inline", "notin", "offset_of", "opaque", "package", "proc", "return", "size_of", "struct", "switch", "transmute", "type_of", "union", "using", "when"}

	// From: https://selinuxproject.org/page/PolicyLanguage
	policyLanguageWords = []string{"alias", "allow", "and", "attribute", "attribute_role", "auditallow", "auditdeny", "bool", "category", "cfalse", "class", "clone", "common", "constrain", "ctrue", "default_range", "default_role", "default_type", "default_user", "dom", "domby", "dominance", "dontaudit", "else", "equals", "false", "filename", "filesystem", "fscon", "fs_use_task", "fs_use_trans", "fs_use_xattr", "genfscon", "h1", "h2", "high", "identifier", "if", "incomp", "inherits", "iomemcon", "ioportcon", "ipv4_addr", "ipv6_addr", "l1", "l2", "level", "low", "low_high", "mlsconstrain", "mlsvalidatetrans", "module", "netifcon", "neverallow", "nodecon", "not", "notequal", "number", "object_r", "optional", "or", "path", "pcidevicecon", "permissive", "pirqcon", "policycap", "portcon", "r1", "r2", "r3", "range", "range_transition", "require", "role", "roleattribute", "roles", "role_transition", "sameuser", "sensitivity", "sid", "source", "t1", "t2", "t3", "target", "true", "type", "typealias", "typeattribute", "typebounds", "type_change", "type_member", "types", "type_transition", "u1", "u2", "u3", "user", "validatetrans", "version_identifier", "xor"}

	scalaWords = []string{"abstract", "case", "catch", "class", "def", "do", "else", "extends", "false", "final", "finally", "for", "forSome", "if", "implicit", "import", "lazy", "match", "new", "null", "object", "override", "package", "private", "protected", "return", "sealed", "super", "this", "throw", "trait", "try", "true", "type", "val", "var", "while", "with", "yield"}

	// Based on /usr/share/nvim/runtime/syntax/zig.vim
	zigWords = []string{"Frame", "OpaqueType", "TagType", "This", "Type", "TypeOf", "Vector", "addWithOverflow", "align", "alignCast", "alignOf", "allowzero", "and", "anyerror", "anyframe", "as", "asm", "async", "asyncCall", "atomicLoad", "atomicRmw", "atomicStore", "await", "bitCast", "bitOffsetOf", "bitReverse", "bitSizeOf", "bool", "boolToInt", "break", "breakpoint", "byteOffsetOf", "byteSwap", "bytesToSlice", "cDefine", "cImport", "cInclude", "cUndef", "c_int", "c_long", "c_longdouble", "c_longlong", "c_short", "c_uint", "c_ulong", "c_ulonglong", "c_ushort", "c_void", "call", "callconv", "canImplicitCast", "catch", "ceil", "clz", "cmpxchgStrong", "cmpxchgWeak", "compileError", "compileLog", "comptime", "comptime_float", "comptime_int", "const", "continue", "cos", "ctz", "defer", "divExact", "divFloor", "divTrunc", "else", "embedFile", "enum", "enumToInt", "errSetCast", "errdefer", "error", "errorName", "errorReturnTrace", "errorToInt", "exp", "exp2", "export", "export", "extern", "f128", "f16", "f32", "f64", "fabs", "false", "fence", "field", "fieldParentPtr", "floatCast", "floatToInt", "floor", "fn", "for", "frame", "frameAddress", "frameSize", "hasDecl", "hasField", "i0", "if", "import", "inline", "intCast", "intToEnum", "intToError", "intToFloat", "intToPtr", "isize", "linksection", "log", "log10", "log2", "memcpy", "memset", "mod", "mulWithOverflow", "newStackCall", "noalias", "noinline", "noreturn", "nosuspend", "null", "or", "orelse", "packed", "panic", "popCount", "ptrCast", "ptrToInt", "pub", "rem", "resume", "return", "returnAddress", "round", "setAlignStack", "setCold", "setEvalBranchQuota", "setFloatMode", "setGlobalLinkage", "setGlobalSection", "setRuntimeSafety", "shlExact", "shlWithOverflow", "shrExact", "shuffle", "sin", "sizeOf", "sliceToBytes", "splat", "sqrt", "struct", "subWithOverflow", "suspend", "switch", "tagName", "test", "threadlocal", "true", "trunc", "truncate", "try", "type", "typeInfo", "typeName", "u0", "undefined", "union", "unionInit", "unreachable", "usingnamespace", "usize", "var", "void", "volatile", "while"}

	// The D programming language
	dWords = []string{"abstract", "alias", "align", "asm", "assert", "auto", "body", "bool", "break", "byte", "case", "cast", "catch", "cdouble", "cent", "cfloat", "char", "class", "const", "continue", "creal", "dchar", "debug", "default", "delegate", "delete", "deprecated", "do", "double", "else", "enum", "export", "extern", "false", "__FILE__", "__FILE_FULL_PATH__", "final", "finally", "float", "for", "foreach", "foreach_reverse", "__FUNCTION__", "function", "goto", "__gshared", "idouble", "if", "ifloat", "immutable", "import", "in", "inout", "int", "interface", "invariant", "ireal", "is", "lazy", "__LINE__", "long", "macro", "mixin", "__MODULE__", "module", "new", "nothrow", "null", "out", "override", "package", "__parameters", "pragma", "__PRETTY_FUNCTION__", "private", "protected", "public", "pure", "real", "ref", "return", "scope", "shared", "short", "static", "struct", "super", "switch", "synchronized", "template", "this", "throw", "__traits", "true", "try", "typeid", "typeof", "ubyte", "ucent", "uint", "ulong", "union", "unittest", "ushort", "__vector", "version", "void", "wchar", "while", "with"}
)

func clearKeywords() {
	syntax.Keywords = make(map[string]struct{})
}

// adjustSyntaxHighlightingKeywords contains per-language adjustments to highlighting of keywords
func adjustSyntaxHighlightingKeywords(mode Mode) {
	var addKeywords, delKeywords []string
	switch mode {
	case modeBattlestar:
		clearKeywords()
		addKeywords = battlestarWords
	case modeClojure:
		clearKeywords()
		addKeywords = clojureWords
	case modeCMake:
		delKeywords = append(delKeywords, []string{"build", "package"}...)
		addKeywords = cmakeWords
	case modeConfig:
		delKeywords = []string{"install"}
	case modeD:
		clearKeywords()
		addKeywords = dWords
	case modeCS:
		clearKeywords()
		addKeywords = csWords
	case modeGo:
		addKeywords = []string{"defer", "fallthrough", "go", "print", "println", "range", "string"}
		delKeywords = []string{"None", "build", "char", "def", "def", "die", "fi", "get", "in", "include", "let", "mut", "next", "pass", "redo", "rescue", "ret", "retry", "set", "template", "then", "when", "where", "while"}
	case modeHIDL:
		clearKeywords()
		addKeywords = hidlWords
	case modeJava:
		addKeywords = []string{"package"}
	case modeJSON:
		delKeywords = []string{"install"}
	case modeKotlin:
		clearKeywords()
		addKeywords = kotlinWords
	case modeLisp:
		clearKeywords()
		addKeywords = emacsWords
	case modeLua:
		clearKeywords()
		addKeywords = luaWords
	case modeNroff:
		clearKeywords()
		delKeywords = []string{"class"}
		addKeywords = []string{"B", "BR", "PP", "SH", "TP", "fB", "fP", "RB", "TH", "IR", "IP", "fI", "fR"}
	case modeManPage:
		clearKeywords()
	case modeOak:
		addKeywords = []string{"fn"}
		delKeywords = []string{"from", "new", "print"}
	case modePython:
		delKeywords = []string{"fn"}
	case modeOdin:
		clearKeywords()
		addKeywords = odinWords
	case modePolicyLanguage: // SE Linux
		clearKeywords()
		addKeywords = policyLanguageWords
	case modeRust:
		addKeywords = []string{"assert_eq", "fn", "impl", "loop", "mod", "out", "panic", "usize", "i64", "i32", "i16", "u64", "u32", "u16", "String", "char"}
		delKeywords = []string{"build", "done", "end", "next", "int64", "uint64", "int32", "uint32", "int16", "uint16", "int", "get", "print", "last"}
	case modeScala:
		clearKeywords()
		addKeywords = scalaWords
	case modeSQL:
		addKeywords = []string{"NOT"}
	case modeVim:
		addKeywords = []string{"call", "echo", "elseif", "endfunction", "map", "nmap", "redraw"}
	case modeZig:
		clearKeywords()
		addKeywords = zigWords
	case modeShell:
		addKeywords = []string{"--force", "-f", "cmake", "configure", "do", "fdisk", "for", "gdisk", "in", "make", "mv", "ninja", "rm", "rmdir", "setopt", "while"}
		delKeywords = []string{"#else", "#endif", "default", "double", "exec", "float", "install", "long", "no", "pass", "ret", "super", "var", "with"}
		fallthrough // to the default case
	default:
		addKeywords = append(addKeywords, []string{"ifeq", "ifneq"}...)
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
	case modeShell, modePython, modeCMake, modeConfig, modeCrystal, modeNim, modePolicyLanguage, modeBazel:
		return "#"
	case modeAssembly:
		return ";"
	case modeHaskell, modeSQL, modeLua, modeAda:
		return "--"
	case modeVim:
		return "\""
	case modeClojure, modeLisp:
		return ";;"
	case modeBat:
		return "@rem" // or rem or just ":" ...
	case modeNroff:
		return `.\"`
	case modeAmber:
		return "!!"
	default:
		return "//"
	}
}
