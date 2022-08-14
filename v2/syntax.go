package main

// TODO: Use a different syntax highlighting package, with support for many different programming languages
import (
	"github.com/xyproto/mode"
	"github.com/xyproto/syntax"
)

var (
	// Assembly
	asmWords = []string{"A0", "A1", "A2", "A3", "A4", "A5", "A6", "A7", "AC", "ADDWATCH", "ALIGN", "AUTO", "BAC0", "BAC1", "BAC2", "BAC3", "BAC4", "BAC5", "BAC6", "BAC7", "BAD0", "BAD1", "BAD2", "BAD3", "BAD4", "BAD5", "BAD6", "BAD7", "BASEREG", "BLK.B", "BLK.D", "BLK.L", "BLK.P", "BLK.S", "BLK.W", "BLK.X", "BUSCR", "CAAR", "CACR", "CAL", "CCR", "CMEXIT", "CNOP", "CRP", "D0", "D1", "D2", "D3", "D4", "D5", "D6", "D7", "DACR0", "DACR1", "DC.B", "DC.D", "DC.L", "DC.P", "DC.S", "DC.W", "DC.X", "DCB.B", "DCB.D", "DCB.L", "DCB.P", "DCB.S", "DCB.W", "DCB.X", "DFC", "DR.B", "DR.L", "DR.W", "DRP", "DS.B", "DS.D", "DS.L", "DS.P", "DS.S", "DS.W", "DS.X", "DTT0", "DTT1", "ELSE", "END", "ENDB", "ENDC", "ENDIF", "ENDM", "ENDOFF", "ENDR", "ENTRY", "EQU", "EQUC", "EQUD", "EQUP", "EQUR", "EQUS", "EQUX", "EREM", "ETEXT", "EVEN", "EXTERN", "EXTRN", "FAIL", "FILESIZE", "FP0", "FP1", "FP2", "FP3", "FP4", "FP5", "FP6", "FP7", "FPCR", "FPIAR", "FPSR", "FileSize", "GLOBAL", "IACR0", "IACR1", "IDNT", "IF1", "IF2", "IFB", "IFC", "IFD", "IFEQ", "IFGE", "IFGT", "IFLE", "IFLT", "IFNB", "IFNC", "IFND", "IFNE", "IMAGE", "INCBIN", "INCDIR", "INCIFF", "INCIFFP", "INCLUDE", "INCSRC", "ISP", "ITT0", "ITT1", "JUMPERR", "JUMPPTR", "LINEA", "LINEF", "LINE_A", "LINE_F", "LIST", "LLEN", "LOAD", "MACRO", "MASK2", "MEXIT", "MMUSR", "MSP", "NOLIST", "NOPAGE", "ODD", "OFFSET", "ORG", "PAGE", "PCR", "PCSR", "PLEN", "PRINTT", "PRINTV", "PSR", "REG", "REGF", "REM", "REPT", "RORG", "RS.B", "RS.L", "RS.W", "RSRESET", "RSSET", "SCC", "SECTION", "SET", "SETCPU", "SETFPU", "SETMMU", "SFC", "SP", "SPC", "SR", "SRP", "TC", "TEXT", "TT0", "TT1", "TTL", "URP", "USP", "VAL", "VBR", "XDEF", "XREF", "ZPC", "_start", "a0", "a1", "a2", "a3", "a4", "a5", "a6", "a7", "abcd", "add", "add", "adda", "addi", "addq", "addx", "and", "andi", "asl", "asr", "bcc", "bchg", "bclr", "bcs", "beq", "bge", "bgt", "bhi", "bhs", "bits", "ble", "blo", "bls", "blt", "bmi", "bne", "bpl", "bra", "bset", "bsr", "btst", "bvc", "bvs", "chk", "clr", "cmp", "cmpa", "cmpi", "cmpm", "d0", "d1", "d2", "d3", "d4", "d5", "d6", "d7", "db", "dbcc", "dbeq", "dbf", "dbra", "dd", "div", "divs", "divu", "dq", "dw", "eor", "eori", "equ", "exg", "ext", "global", "illegal", "inc", "int", "jmp", "jsr", "lea", "lea", "link", "lsl", "lsr", "mov", "move", "movea", "movem", "movep", "moveq", "muls", "mulu", "nbcd", "neg", "negx", "nop", "not", "or", "org", "ori", "out", "pea", "pop", "push", "reset", "rol", "rol", "ror", "ror", "roxl", "roxr", "rte", "rtr", "rts", "sbcd", "scc", "scs", "section", "seq", "sf", "sge", "sgt", "shi", "shl", "shr", "sle", "sls", "slt", "smi", "sne", "sp", "spl", "st", "stop", "sub", "sub", "suba", "subi", "subq", "subx", "svc", "svs", "swap", "syscall", "tas", "trap", "trapv", "tst", "unlk", "xor"}

	// Battlestar
	battlestarWords = []string{"address", "asm", "bootable", "break", "call", "chr", "const", "continue", "counter", "end", "exit", "extern", "fun", "funparam", "halt", "int", "len", "loop", "loopwrite", "mem", "membyte", "memdouble", "memword", "noret", "print", "rawloop", "read", "readbyte", "readdouble", "readword", "ret", "syscall", "sysparam", "use", "value", "var", "write"}

	// Clojure
	clojureWords = []string{"*1", "*2", "*3", "*agent*", "*clojure-version*", "*command-line-args*", "*compile-files*", "*compile-path*", "*e", "*err*", "*file*", "*in*", "*ns*", "*out*", "*print-dup*", "*print-length*", "*print-level*", "*print-meta*", "*print-readably*", "*warn on reflection*", "accessor", "aclone", "add-watch", "agent", "agent-error", "agent-errors", "aget", "alength", "alias", "all-ns", "alter", "alter-meta!", "alter-var-root", "amap", "ancestors", "and", "apply", "areduce", "array-map", "as->", "aset", "aset-boolean", "aset-byte", "aset-char", "aset-double", "aset-float", "aset-int", "aset-long", "aset-short", "assert", "assoc", "assoc", "assoc", "assoc!", "assoc-in", "associative?", "atom", "await", "await-for", "bases", "bean", "bigdec", "bigdec?", "bigint", "binding", "bit-and", "bit-and-not", "bit-clear", "bit-flip", "bit-not", "bit-or", "bit-set", "bit-shift-left", "bit-shift-right", "bit-test", "bit-xor", "boolean", "boolean-array", "booleans", "bound-fn", "bound-fn*", "bound?", "butlast", "byte", "byte-array", "bytes", "case", "cast", "catch", "char", "char-array", "char?", "chars", "class", "class?", "clojure-version", "coll?", "commute", "comp", "comparator", "compare", "compare-and-set!", "compile", "complement", "concat", "cond", "cond->", "cond->>", "condp", "conj", "conj", "conj", "conj", "conj", "conj!", "cons", "constantly", "construct-proxy", "contains?", "count", "count", "counted?", "create-ns", "create-struct", "cycle", "dec", "decimal?", "declare", "dedupe", "def", "definline", "defmacro", "defmoethod", "defmulti", "defn", "defonce", "defprotocol", "defrecord", "defstruct", "deftype", "delay", "delay?", "deliver", "denominator", "deref", "deref", "derive", "descendants", "disj", "disj!", "dissoc", "dissoc!", "distinct", "distinct?", "do", "eval", "doall", "doall", "dorun", "dorun", "doseq", "doseq", "dosync", "dotimes", "doto", "double", "double-array", "double?", "doubles", "drop", "drop-last", "drop-while", "eduction", "empty", "empty?", "ensure", "enumeration-seq", "error-handler", "error-mode", "even?", "every-pred", "every?", "extend", "extend-protocol", "extend-type", "extenders", "extends?", "false?", "ffirst", "file-seq", "filter", "filterv", "finally", "find", "find-ns", "find-var", "first", "first", "flatten", "float", "float-array", "float?", "floats", "flush", "fn", "fn?", "fnext", "fnil", "for", "for", "force", "format", "frequencies", "future", "future-call", "future-cancel", "future-cancelled?", "future-done?", "future?", "gen-class", "gen-interface", "gensym", "gensym", "get", "get", "get", "get", "get", "get-in", "get-method", "get-proxy-class", "get-thread-bindings", "get-validator", "group-by", "hash", "hash-map", "hash-set", "ident?", "identical?", "identity", "if", "if-let", "if-not", "if-some", "ifn?", "import", "in-ns", "inc", "init-proxy", "instance?", "int", "int-array", "int?", "integer?", "interleave", "intern", "intern", "interpose", "into", "into-array", "ints", "io!", "isa?", "isa?", "iterate", "iterate", "iterator-seq", "juxt", "keep", "keep-indexed", "key", "keys", "keyword", "keyword?", "last", "lazy-cat", "lazy-cat", "lazy-seq", "lazy-seq", "let", "letfn", "line-seq", "list", "list?", "load", "load-file", "load-reader", "load-string", "loaded-libs", "locking", "long", "long-array", "longs", "loop", "macroexpand", "macroexpand-1", "make-array", "make-hierarchy", "map", "map-indexed", "map?", "mapcat", "mapv", "max", "max-key", "memfn", "memoize", "merge", "merge-with", "meta", "methods", "min", "min-key", "mod", "name", "namespace", "namespace-munge", "nat-int?", "neg?", "newline", "next", "nfirst", "nil?", "nnext", "non-empty", "not", "not", "not-any?", "not-every?", "ns", "ns-aliases", "ns-imports", "ns-interns", "ns-map", "ns-name", "ns-publics", "ns-refers", "ns-resolve", "ns-resolve", "ns-unalias", "ns-unmap", "nth", "nthnext", "nthrest", "num", "number?", "numerator", "object-array", "odd?", "or", "parents", "partial", "partition", "partition-all", "partition-by", "pcalls", "peek", "peek", "persistent!", "pmap", "pop", "pop", "pop!", "pop-thread-bindings", "pos-int?", "pos?", "pr", "pr-str", "pr-str", "prefer-method", "prefers", "print", "print-str", "print-str", "printf", "println", "println-str", "println-str", "prn", "prn-str", "prn-str", "promise", "proxy", "proxy-mappings", "proxy-super", "push-thread-bindings", "pvalues", "qualified-ident?", "qualified-keyword?", "qualified-symbol?", "quot", "rand", "rand-int", "rand-nth", "random-sample", "range", "ratio?", "rational?", "rationalize", "re-find", "re-groups", "re-matcher", "re-matches", "re-pattern", "re-seq", "read", "read-line", "read-string", "recur", "reduce", "reduce-kv", "reductions", "ref", "ref-history-count", "ref-max-history", "ref-min-history", "ref-set", "refer", "refer-clojure", "reify", "release-pending", "rem", "remove", "remove-all-methods", "remove-method", "remove-ns", "remove-watch", "repeat", "repeatedly", "repeatedly", "replace", "replicate", "require", "reset!", "reset-meta!", "resolve", "rest", "rest", "restart-agent", "resultset-seq", "reverse", "reversible?", "rseq", "rseq", "rsubseq", "satisfies?", "second", "select-keys", "send", "send-off", "seq", "seq?", "seqable?", "seque", "sequence", "sequential?", "set", "set", "set!", "set-error-handler", "set-error-mode", "set-validator", "set?", "short", "short-array", "shorts", "shuffle", "shutdonw-agents", "simple-ident?", "simple-keyword?", "simple-symbol?", "slurp", "some", "some->", "some->>", "some-fn", "sort", "sort-by", "sorted-map", "sorted-map-by", "sorted-set", "sorted-set-by", "sorted?", "special-symbol?", "spit", "split-at", "split-with", "str", "string?", "struct", "struct-map", "subs", "subseq", "subvec", "supers", "swap!", "symbol", "symbol?", "sync", "take", "take-last", "take-nth", "take-while", "test", "the-ns", "thread-bound?", "throw", "time", "to-array", "to-array-2d", "trampoline", "transduce", "transient", "tree-seq", "true?", "try", "type", "underive", "update", "update-in", "update-proxy", "use", "val", "vals", "var", "var-get", "var?", "vec", "vector", "vector-of", "vector?", "very-meta", "volatile!", "vreset!", "vswap!", "when", "when-first", "when-let", "when-not", "when-some", "while", "with-bindings", "with-bindings*", "with-in-str", "with-local-vars", "with-meta", "with-open", "with-out-str", "with-out-str", "with-precision", "xml-seq", "zero?", "zipmap"}

	// CMake, based on /usr/share/nvim/runtime/syntax/cmake.vim
	cmakeWords = []string{"add_compile_options", "add_custom_command", "add_custom_target", "add_definitions", "add_dependencies", "add_executable", "add_library", "add_subdirectory", "add_test", "build_command", "build_name", "cmake_host_system_information", "cmake_minimum_required", "cmake_parse_arguments", "cmake_policy", "configure_file", "create_test_sourcelist", "ctest_build", "ctest_configure", "ctest_coverage", "ctest_memcheck", "ctest_run_script", "ctest_start", "ctest_submit", "ctest_test", "ctest_update", "ctest_upload", "define_property", "enable_language", "endforeach", "endfunction", "endif", "exec_program", "execute_process", "export", "export_library_dependencies", "file", "find_file", "find_library", "find_package", "find_path", "find_program", "fltk_wrap_ui", "foreach", "function", "get_cmake_property", "get_directory_property", "get_filename_component", "get_property", "get_source_file_property", "get_target_property", "get_test_property", "if", "include", "include_directories", "include_external_msproject", "include_guard", "install", "install_files", "install_programs", "install_targets", "list", "load_cache", "load_command", "macro", "make_directory", "mark_as_advanced", "math", "message", "option", "project", "remove", "separate_arguments", "set", "set_directory_properties", "set_package_properties", "set_property", "set_source_files_properties", "set_target_properties", "set_tests_properties", "source_group", "string", "subdirs", "target_compile_definitions", "target_compile_features", "target_compile_options", "target_include_directories", "target_link_libraries", "target_sources", "try_compile", "try_run", "unset", "use_mangled_mesa", "variable_requires", "variable_watch", "while", "write_file"}

	// C#
	csWords = []string{"Boolean", "Byte", "Char", "Decimal", "Double", "Int16", "Int32", "Int64", "IntPtr", "Object", "Short", "Single", "String", "UInt16", "UInt32", "UInt64", "UIntPtr", "abstract", "as", "base", "bool", "break", "byte", "case", "catch", "char", "checked", "class", "const", "continue", "decimal", "default", "delegate", "do", "double", "dynamic", "else", "enum", "event", "explicit", "extern", "false", "finally", "fixed", "float", "for", "foreach", "goto", "if", "implicit", "in", "int", "interface", "internal", "is", "lock", "long", "namespace", "new", "nint", "nuint", "null", "object", "operator", "out", "override", "params", "readonly", "ref", "return", "sbyte", "sealed", "short", "sizeof", "stackalloc", "static", "string", "struct", "switch", "this", "throw", "true", "try", "typeof", "uint", "ulong", "unchecked", "unsafe", "ushort", "using", "virtual", "void", "volatile", "while"} // private, public, protected

	// Elisp
	emacsWords = []string{"add-to-list", "defconst", "defun", "defvar", "if", "lambda", "let", "load", "nil", "require", "setq", "when"} // this should do it

	// Hardware Interface Description Language. Keywords from https://source.android.com/devices/architecture/hidl
	hidlWords = []string{"constexpr", "enum", "extends", "generates", "import", "interface", "oneway", "package", "safe_union", "struct", "typedef", "union"}

	// Kotlin
	kotlinWords = []string{"as", "break", "catch", "class", "continue", "do", "else", "false", "for", "fun", "if", "import", "in", "interface", "is", "null", "object", "override", "package", "return", "super", "this", "throw", "true", "try", "typealias", "typeof", "val", "var", "when", "while"}

	// Lua
	luaWords = []string{"and", "break", "do", "else", "elseif", "end", "false", "for", "function", "goto", "if", "in", "local", "nil", "not", "or", "repeat", "return", "then", "true", "until", "while"}

	// OCaml
	ocamlWords = []string{"and", "as", "assert", "asr", "begin", "class", "constraint", "do", "done", "downto", "else", "end", "exception", "external", "false", "for", "fun", "function", "functor", "if", "in", "include", "inherit", "initializer", "land", "lazy", "let", "lor", "lsl", "lsr", "lxor", "match", "method", "mod", "module", "mutable", "new", "nonrec", "object", "of", "open", "or", "private", "rec", "sig", "struct", "then", "to", "true", "try", "type", "val", "virtual", "when", "while", "with"}

	// Odin
	odinWords = []string{"align_of", "auto_cast", "bit_field", "bit_set", "break", "case", "cast", "const", "context", "continue", "defer", "distinct", "do", "do", "dynamic", "else", "enum", "fallthrough", "for", "foreign", "if", "import", "in", "inline", "macro", "map", "no_inline", "notin", "offset_of", "opaque", "package", "proc", "return", "size_of", "struct", "switch", "transmute", "type_of", "union", "using", "when"}

	// Based on https://selinuxproject.org/page/PolicyLanguage
	policyLanguageWords = []string{"alias", "allow", "and", "attribute", "attribute_role", "auditallow", "auditdeny", "bool", "category", "cfalse", "class", "clone", "common", "constrain", "ctrue", "default_range", "default_role", "default_type", "default_user", "dom", "domby", "dominance", "dontaudit", "else", "equals", "false", "filename", "filesystem", "fscon", "fs_use_task", "fs_use_trans", "fs_use_xattr", "genfscon", "h1", "h2", "high", "identifier", "if", "incomp", "inherits", "iomemcon", "ioportcon", "ipv4_addr", "ipv6_addr", "l1", "l2", "level", "low", "low_high", "mlsconstrain", "mlsvalidatetrans", "module", "netifcon", "neverallow", "nodecon", "not", "notequal", "number", "object_r", "optional", "or", "path", "pcidevicecon", "permissive", "pirqcon", "policycap", "portcon", "r1", "r2", "r3", "range", "range_transition", "require", "role", "roleattribute", "roles", "role_transition", "sameuser", "sensitivity", "sid", "source", "t1", "t2", "t3", "target", "true", "type", "typealias", "typeattribute", "typebounds", "type_change", "type_member", "types", "type_transition", "u1", "u2", "u3", "user", "validatetrans", "version_identifier", "xor"}

	// Scala
	scalaWords = []string{"abstract", "case", "catch", "class", "def", "do", "else", "extends", "false", "final", "finally", "for", "forSome", "if", "implicit", "import", "lazy", "match", "new", "null", "object", "override", "package", "private", "protected", "return", "sealed", "super", "this", "throw", "trait", "try", "true", "type", "val", "var", "while", "with", "yield"}

	// Based on /usr/share/nvim/runtime/syntax/zig.vim
	zigWords = []string{"Frame", "OpaqueType", "TagType", "This", "Type", "TypeOf", "Vector", "addWithOverflow", "align", "alignCast", "alignOf", "allowzero", "and", "anyerror", "anyframe", "as", "asm", "async", "asyncCall", "atomicLoad", "atomicRmw", "atomicStore", "await", "bitCast", "bitOffsetOf", "bitReverse", "bitSizeOf", "bool", "boolToInt", "break", "breakpoint", "byteOffsetOf", "byteSwap", "bytesToSlice", "cDefine", "cImport", "cInclude", "cUndef", "c_int", "c_long", "c_longdouble", "c_longlong", "c_short", "c_uint", "c_ulong", "c_ulonglong", "c_ushort", "c_void", "call", "callconv", "canImplicitCast", "catch", "ceil", "clz", "cmpxchgStrong", "cmpxchgWeak", "compileError", "compileLog", "comptime", "comptime_float", "comptime_int", "const", "continue", "cos", "ctz", "defer", "divExact", "divFloor", "divTrunc", "else", "embedFile", "enum", "enumToInt", "errSetCast", "errdefer", "error", "errorName", "errorReturnTrace", "errorToInt", "exp", "exp2", "export", "export", "extern", "f128", "f16", "f32", "f64", "fabs", "false", "fence", "field", "fieldParentPtr", "floatCast", "floatToInt", "floor", "fn", "for", "frame", "frameAddress", "frameSize", "hasDecl", "hasField", "i0", "if", "import", "inline", "intCast", "intToEnum", "intToError", "intToFloat", "intToPtr", "isize", "linksection", "log", "log10", "log2", "memcpy", "memset", "mod", "mulWithOverflow", "newStackCall", "noalias", "noinline", "noreturn", "nosuspend", "null", "or", "orelse", "packed", "panic", "popCount", "ptrCast", "ptrToInt", "pub", "rem", "resume", "return", "returnAddress", "round", "setAlignStack", "setCold", "setEvalBranchQuota", "setFloatMode", "setGlobalLinkage", "setGlobalSection", "setRuntimeSafety", "shlExact", "shlWithOverflow", "shrExact", "shuffle", "sin", "sizeOf", "sliceToBytes", "splat", "sqrt", "struct", "subWithOverflow", "suspend", "switch", "tagName", "test", "threadlocal", "true", "trunc", "truncate", "try", "type", "typeInfo", "typeName", "u0", "undefined", "union", "unionInit", "unreachable", "usingnamespace", "usize", "var", "void", "volatile", "while"}

	// The D programming language
	dWords = []string{"abstract", "alias", "align", "asm", "assert", "auto", "body", "bool", "break", "byte", "case", "cast", "catch", "cdouble", "cent", "cfloat", "char", "class", "const", "continue", "creal", "dchar", "debug", "default", "delegate", "delete", "deprecated", "do", "double", "else", "enum", "export", "extern", "false", "__FILE__", "__FILE_FULL_PATH__", "final", "finally", "float", "for", "foreach", "foreach_reverse", "__FUNCTION__", "function", "goto", "__gshared", "idouble", "if", "ifloat", "immutable", "import", "in", "inout", "int", "interface", "invariant", "ireal", "is", "lazy", "__LINE__", "long", "macro", "mixin", "__MODULE__", "module", "new", "nothrow", "null", "out", "override", "package", "__parameters", "pragma", "__PRETTY_FUNCTION__", "private", "protected", "public", "pure", "real", "ref", "return", "scope", "shared", "short", "static", "struct", "super", "switch", "synchronized", "template", "this", "throw", "__traits", "true", "try", "typeid", "typeof", "ubyte", "ucent", "uint", "ulong", "union", "unittest", "ushort", "__vector", "version", "void", "wchar", "while", "with"}

	// Standard ML
	smlWords = []string{"abstype", "and", "andalso", "as", "case", "do", "datatype", "else", "end", "eqtype", "exception", "fn", "fun", "functor", "handle", "if", "in", "include", "infix", "infixr", "let", "local", "nonfix", "of", "op", "open", "orelse", "raise", "rec", "sharing", "sig", "signature", "struct", "structure", "then", "type", "val", "where", "with", "withtype", "while"}

	// Erlang
	erlangWords = []string{"after", "and", "andalso", "band", "begin", "bnot", "bor", "bsl", "bsr", "bxor", "case", "catch", "cond", "div", "end", "fun", "if", "let", "not", "of", "or", "orelse", "receive", "rem", "try", "when", "xor"}
)

func clearKeywords() {
	syntax.Keywords = make(map[string]struct{})
}

// adjustSyntaxHighlightingKeywords contains per-language adjustments to highlighting of keywords
func adjustSyntaxHighlightingKeywords(m mode.Mode) {
	var addKeywords, delKeywords []string
	switch m {
	case mode.Ada:
		addKeywords = []string{"constant", "loop", "procedure", "project"}
	case mode.Assembly:
		clearKeywords()
		addKeywords = asmWords
	case mode.Battlestar:
		clearKeywords()
		addKeywords = battlestarWords
	case mode.Clojure:
		clearKeywords()
		addKeywords = clojureWords
	case mode.CMake:
		delKeywords = append(delKeywords, "build", "package")
		addKeywords = cmakeWords
	case mode.Config:
		delKeywords = []string{"auto", "install"}
	case mode.CS:
		clearKeywords()
		addKeywords = csWords
	case mode.D:
		clearKeywords()
		addKeywords = dWords
	case mode.Erlang:
		clearKeywords()
		addKeywords = erlangWords
	case mode.Go:
		addKeywords = []string{"defer", "error", "fallthrough", "go", "print", "println", "range", "rune", "string", "uint", "uint8", "uint16", "uint32", "uint64"}
		delKeywords = []string{"False", "None", "True", "assert", "auto", "build", "char", "def", "def", "del", "die", "done", "end", "fi", "final", "finally", "fn", "from", "get", "in", "include", "is", "last", "let", "match", "mut", "next", "no", "pass", "redo", "rescue", "ret", "retry", "set", "template", "then", "this", "when", "where", "while", "yes"}
	case mode.HIDL:
		clearKeywords()
		addKeywords = hidlWords
	case mode.AIDL:
		addKeywords = append(addKeywords, "interface")
		addKeywords = append(addKeywords, hidlWords...)
		fallthrough // continue to mode.Java
	case mode.Java:
		addKeywords = append(addKeywords, "package")
		delKeywords = append(delKeywords, "add", "in", "local", "sub")
	case mode.JSON:
		delKeywords = []string{"install"}
	case mode.Kotlin:
		clearKeywords()
		addKeywords = kotlinWords
	case mode.Lisp:
		clearKeywords()
		addKeywords = emacsWords
	case mode.Teal, mode.Terra:
		fallthrough // continue to mode.Lua, for now
	case mode.Lua:
		clearKeywords()
		addKeywords = luaWords
	case mode.Nroff:
		clearKeywords()
		delKeywords = []string{"class"}
		addKeywords = []string{"B", "BR", "PP", "SH", "TP", "fB", "fP", "RB", "TH", "IR", "IP", "fI", "fR"}
	case mode.ManPage:
		clearKeywords()
	case mode.ObjectPascal:
		addKeywords = []string{"do", "downto", "function", "nil", "procedure", "program", "then", "to", "uses"}
	case mode.Oak:
		addKeywords = []string{"fn"}
		delKeywords = []string{"from", "new", "print"}
	case mode.Python:
		delKeywords = []string{"append", "exit", "fn", "get", "package", "print"}
	case mode.Odin:
		clearKeywords()
		addKeywords = odinWords
	case mode.PolicyLanguage: // SE Linux
		clearKeywords()
		addKeywords = policyLanguageWords
	case mode.Garnet, mode.Hare, mode.Jakt, mode.Rust: // Originally only for Rust, split up as needed
		addKeywords = []string{"String", "assert_eq", "char", "fn", "i16", "i32", "i64", "i8", "impl", "loop", "mod", "out", "panic", "u16", "u32", "u64", "u8", "usize"}
		// "as" and "mut" are treated as special cases in the syntax package
		delKeywords = []string{"as", "build", "byte", "done", "get", "int", "int16", "int32", "int64", "last", "map", "mut", "next", "pass", "print", "uint16", "uint32", "uint64", "var"}
		if m != mode.Garnet {
			delKeywords = append(delKeywords, "end")
		}
	case mode.Scala:
		clearKeywords()
		addKeywords = scalaWords
	case mode.OCaml:
		clearKeywords()
		addKeywords = ocamlWords
	case mode.Elm, mode.StandardML:
		clearKeywords()
		addKeywords = smlWords
	case mode.SQL:
		addKeywords = []string{"NOT"}
	case mode.Vim:
		addKeywords = []string{"call", "echo", "elseif", "endfunction", "map", "nmap", "redraw"}
	case mode.Zig:
		clearKeywords()
		addKeywords = zigWords
	case mode.GoAssembly:
		// Only highlight some words, to make them stand out
		clearKeywords()
		addKeywords = []string{"db", "dd", "dw", "int", "resb", "resd", "resw", "section", "syscall"}
	case mode.Make, mode.Shell:
		addKeywords = []string{"--force", "-f", "checkout", "clean", "cmake", "configure", "dd", "do", "doas", "done", "endif", "exec", "fdisk", "for", "gdisk", "ifeq", "ifneq", "in", "make", "mv", "ninja", "rm", "rmdir", "setopt", "su", "sudo", "while"}
		delKeywords = []string{"#else", "#endif", "as", "build", "default", "double", "exec", "finally", "float", "fn", "generic", "get", "long", "new", "no", "package", "pass", "print", "property", "require", "ret", "set", "super", "super", "template", "type", "var", "with"}
		if m == mode.Shell { // Only for shell scripts, not for Makefiles
			delKeywords = append(delKeywords, "install")
		}
	case mode.Shader:
		addKeywords = []string{"buffer", "bvec2", "bvec3", "bvec4", "coherent", "dvec2", "dvec3", "dvec4", "flat", "in", "inout", "invariant", "ivec2", "ivec3", "ivec4", "layout", "mat", "mat2", "mat3", "mat4", "noperspective", "out", "precision", "readonly", "restrict", "smooth", "uniform", "uvec2", "uvec3", "uvec4", "vec2", "vec3", "vec4", "volatile", "writeonly"}
		fallthrough // Continue to C/C++ and then to the default
	case mode.C, mode.Cpp:
		addKeywords = append(addKeywords, "int8_t", "uint8_t", "int16_t", "uint16_t", "int32_t", "uint32_t", "int64_t", "uint64_t", "size_t")
		fallthrough // Continue to the default
	default:
		addKeywords = append(addKeywords, "endif", "ifeq", "ifneq")
		delKeywords = append(delKeywords, "build", "done", "package", "require", "set", "super", "type")
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
	case mode.Bazel, mode.CMake, mode.Config, mode.Crystal, mode.Make, mode.Nim, mode.PolicyLanguage, mode.Python, mode.Shell:
		return "#"
	case mode.Assembly:
		return ";"
	case mode.Elm, mode.OCaml, mode.StandardML:
		// not applicable, just return something that is unlikely to appear at the beginning of a line
		fallthrough
	case mode.Ada, mode.Agda, mode.Garnet, mode.Haskell, mode.Lua, mode.SQL, mode.Teal, mode.Terra:
		return "--"
	case mode.Vim:
		return "\""
	case mode.Clojure, mode.Lisp:
		return ";;"
	case mode.Bat:
		return "@rem" // or rem or just ":" ...
	case mode.Nroff:
		return `.\"`
	case mode.Amber:
		return "!!"
	case mode.Perl, mode.Prolog:
		return "%"
	case mode.M4:
		return "dnl"
	case mode.Basic:
		return "'"
	case mode.Email:
		return "GIT:"
	default:
		return "//"
	}
}
