package main

// TODO: Use a different syntax highlighting package, with support for many different programming languages
import (
	"strings"

	"github.com/xyproto/mode"
)

var Keywords = map[string]struct{}{
	"#define":          {},
	"#elif":            {},
	"#else":            {},
	"#endif":           {},
	"#ifdef":           {},
	"#ifndef":          {},
	"#include":         {},
	"#pragma":          {},
	"BEGIN":            {},
	"END":              {},
	"False":            {},
	"Infinity":         {},
	"NULL":             {},
	"NaN":              {},
	"None":             {},
	"True":             {},
	"abstract":         {},
	"alias":            {},
	"align_union":      {},
	"alignof":          {},
	"and":              {},
	"append":           {},
	"as":               {},
	"asm":              {},
	"assert":           {},
	"auto":             {},
	"axiom":            {},
	"begin":            {},
	"bool":             {},
	"boolean":          {},
	"break":            {},
	"build":            {},
	"byte":             {},
	"caller":           {},
	"case":             {},
	"catch":            {},
	"char":             {},
	"concept":          {},
	"concept_map":      {},
	"const":            {},
	"const_cast":       {},
	"constexpr":        {},
	"continue":         {},
	"debugger":         {},
	"decltype":         {},
	"def":              {},
	"default":          {},
	"defined":          {},
	"del":              {},
	"delegate":         {},
	"delete":           {},
	"die":              {},
	"do":               {},
	"done":             {},
	"double":           {},
	"dump":             {},
	"dynamic_cast":     {},
	"elif":             {},
	"else":             {},
	"elsif":            {},
	"end":              {},
	"ensure":           {},
	"enum":             {},
	"esac":             {},
	"eval":             {},
	"except":           {},
	"exec":             {},
	"exit":             {},
	"explicit":         {},
	"export":           {},
	"extends":          {},
	"extern":           {},
	"false":            {},
	"fi":               {},
	"final":            {},
	"finally":          {},
	"float":            {},
	"float32":          {},
	"float64":          {},
	"fn":               {},
	"for":              {},
	"foreach":          {},
	"friend":           {},
	"from":             {},
	"func":             {},
	"function":         {},
	"generic":          {},
	"get":              {},
	"global":           {},
	"goto":             {},
	"if":               {},
	"implements":       {},
	"import":           {},
	"in":               {},
	"inline":           {},
	"install":          {},
	"instanceof":       {},
	"int":              {},
	"int16":            {},
	"int32":            {},
	"int64":            {},
	"int8":             {},
	"interface":        {},
	"is":               {},
	"lambda":           {},
	"last":             {},
	"late_check":       {},
	"let":              {},
	"local":            {},
	"long":             {},
	"make":             {},
	"map":              {},
	"match":            {},
	"module":           {},
	"mut":              {},
	"mutable":          {},
	"namespace":        {},
	"native":           {},
	"new":              {},
	"next":             {},
	"nil":              {},
	"no":               {},
	"nonlocal":         {},
	"not":              {},
	"null":             {},
	"nullptr":          {},
	"operator":         {},
	"or":               {},
	"our":              {},
	"package":          {},
	"pass":             {},
	"print":            {},
	"property":         {},
	"raise":            {},
	"redo":             {},
	"register":         {},
	"reinterpret_cast": {},
	"require":          {},
	"rescue":           {},
	"ret":              {},
	"retry":            {},
	"return":           {},
	"self":             {},
	"set":              {},
	"short":            {},
	"signed":           {},
	"sizeof":           {},
	"static":           {},
	"static_assert":    {},
	"static_cast":      {},
	"strictfp":         {},
	"struct":           {},
	"sub":              {},
	"super":            {},
	"switch":           {},
	"synchronized":     {},
	"template":         {},
	"then":             {},
	"this":             {},
	"throw":            {},
	"throws":           {},
	"transient":        {},
	"true":             {},
	"try":              {},
	"type":             {},
	"typedef":          {},
	"typeid":           {},
	"typename":         {},
	"typeof":           {},
	"undef":            {},
	"undefined":        {},
	"union":            {},
	"unless":           {},
	"unsigned":         {},
	"until":            {},
	"use":              {},
	"using":            {},
	"var":              {},
	"virtual":          {},
	"void":             {},
	"volatile":         {},
	"wantarray":        {},
	"when":             {},
	"where":            {},
	"while":            {},
	"with":             {},
	"yield":            {},
}

var (
	// Assembly
	asmWords = []string{"A0", "A1", "A2", "A3", "A4", "A5", "A6", "A7", "AC", "ADDWATCH", "ALIGN", "AUTO", "BAC0", "BAC1", "BAC2", "BAC3", "BAC4", "BAC5", "BAC6", "BAC7", "BAD0", "BAD1", "BAD2", "BAD3", "BAD4", "BAD5", "BAD6", "BAD7", "BASEREG", "BLK.B", "BLK.D", "BLK.L", "BLK.P", "BLK.S", "BLK.W", "BLK.X", "BUSCR", "CAAR", "CACR", "CAL", "CCR", "CMEXIT", "CNOP", "CRP", "D0", "D1", "D2", "D3", "D4", "D5", "D6", "D7", "DACR0", "DACR1", "DC.B", "DC.D", "DC.L", "DC.P", "DC.S", "DC.W", "DC.X", "DCB.B", "DCB.D", "DCB.L", "DCB.P", "DCB.S", "DCB.W", "DCB.X", "DFC", "DR.B", "DR.L", "DR.W", "DRP", "DS.B", "DS.D", "DS.L", "DS.P", "DS.S", "DS.W", "DS.X", "DTT0", "DTT1", "ELSE", "END", "ENDB", "ENDC", "ENDIF", "ENDM", "ENDOFF", "ENDR", "ENTRY", "EQU", "EQUC", "EQUD", "EQUP", "EQUR", "EQUS", "EQUX", "EREM", "ETEXT", "EVEN", "EXTERN", "EXTRN", "FAIL", "FILESIZE", "FP0", "FP1", "FP2", "FP3", "FP4", "FP5", "FP6", "FP7", "FPCR", "FPIAR", "FPSR", "FileSize", "GLOBAL", "IACR0", "IACR1", "IDNT", "IF1", "IF2", "IFB", "IFC", "IFD", "IFEQ", "IFGE", "IFGT", "IFLE", "IFLT", "IFNB", "IFNC", "IFND", "IFNE", "IMAGE", "INCBIN", "INCDIR", "INCIFF", "INCIFFP", "INCLUDE", "INCSRC", "ISP", "ITT0", "ITT1", "JUMPERR", "JUMPPTR", "LINEA", "LINEF", "LINE_A", "LINE_F", "LIST", "LLEN", "LOAD", "MACRO", "MASK2", "MEXIT", "MMUSR", "MSP", "NOLIST", "NOPAGE", "ODD", "OFFSET", "ORG", "PAGE", "PCR", "PCSR", "PLEN", "PRINTT", "PRINTV", "PSR", "REG", "REGF", "REM", "REPT", "RORG", "RS.B", "RS.L", "RS.W", "RSRESET", "RSSET", "SCC", "SECTION", "SET", "SETCPU", "SETFPU", "SETMMU", "SFC", "SP", "SPC", "SR", "SRP", "TC", "TEXT", "TT0", "TT1", "TTL", "URP", "USP", "VAL", "VBR", "XDEF", "XREF", "ZPC", "_start", "a0", "a1", "a2", "a3", "a4", "a5", "a6", "a7", "abcd", "add", "add", "adda", "addi", "addq", "addx", "and", "andi", "asl", "asr", "bcc", "bchg", "bclr", "bcs", "beq", "bge", "bgt", "bhi", "bhs", "bits", "ble", "blo", "bls", "blt", "bmi", "bne", "bpl", "bra", "bset", "bsr", "btst", "bvc", "bvs", "chk", "clr", "cmp", "cmpa", "cmpi", "cmpm", "d0", "d1", "d2", "d3", "d4", "d5", "d6", "d7", "db", "dbcc", "dbeq", "dbf", "dbra", "dd", "div", "divs", "divu", "dq", "dw", "eor", "eori", "equ", "exg", "ext", "global", "illegal", "inc", "int", "jmp", "jsr", "lea", "lea", "link", "lsl", "lsr", "mov", "move", "movea", "movem", "movep", "moveq", "muls", "mulu", "nbcd", "neg", "negx", "nop", "not", "or", "org", "ori", "out", "pea", "pop", "push", "reset", "rol", "rol", "ror", "ror", "roxl", "roxr", "rte", "rtr", "rts", "sbcd", "scc", "scs", "section", "seq", "sf", "sge", "sgt", "shi", "shl", "shr", "sle", "sls", "slt", "smi", "sne", "sp", "spl", "st", "stop", "sub", "sub", "suba", "subi", "subq", "subx", "svc", "svs", "swap", "syscall", "tas", "trap", "trapv", "tst", "unlk", "xor"}

	// Battlestar
	battlestarWords = []string{"address", "asm", "bootable", "break", "call", "chr", "const", "continue", "counter", "end", "exit", "extern", "fun", "funparam", "halt", "int", "len", "loop", "loopwrite", "mem", "membyte", "memdouble", "memword", "noret", "print", "rawloop", "read", "readbyte", "readdouble", "readword", "ret", "syscall", "sysparam", "use", "value", "var", "write"}

	// C3
	c3Words = []string{"$$BENCHMARK_FNS ", "$$BENCHMARK_NAMES", "$$DATE", "$$FILE", "$$FILEPATH", "$$FUNC", "$$FUNCTION", "$$LINE", "$$LINE_RAW", "$$MODULE", "$$TEST_FNS", "$$TEST_NAMES", "$$TIME", "$alignof", "$assert", "$case", "$default", "$defined", "$echo", "$else", "$embed", "$endfor", "$endforeach", "$endif", "$endswitch", "$error", "$eval", "$evaltype", "$exec", "$extnameof", "$for", "$foreach", "$if", "$include", "$nameof", "$offsetof", "$qnameof", "$sizeof", "$stringify", "$switch", "$typefrom", "$typeof", "$vaarg", "$vaconst", "$vacount", "$vaexpr", "$varef", "$vasplat", "$vatype", "@align", "@benchmark", "@bigendian", "@builtin", "@cdecl", "@deprecated", "@dynamic", "@export", "@extern", "@extname", "@inline", "@interface", "@littleendian", "@local", "@maydiscard", "@naked", "@nodiscard", "@noinit", "@noinline", "@noreturn", "@nostrip", "@obfuscate", "@operator", "@overlap", "@packed", "@priority", "@private", "@public", "@pure", "@reflect", "@section", "@stdcall", "@test", "@unused", "@used", "@veccall", "@wasm", "@weak", "@winmain", "any", "anyfault", "asm", "assert", "bitstruct", "bool", "break", "case", "catch", "char", "const", "continue", "def", "default", "defer", "distinct", "do", "double", "else", "enum", "extern", "false", "fault", "float", "float128", "float16", "fn", "for", "foreach", "foreach_r", "ichar", "if", "import", "inline", "int", "int128", "iptr", "isz", "long", "macro", "module", "nextcase", "null", "return", "short", "static", "struct", "switch", "tlocal", "true", "try", "typeid", "uint", "uint128", "ulong", "union", "uptr", "ushort", "usz", "var", "void", "while"}

	// Clojure
	clojureWords = []string{"*1", "*2", "*3", "*agent*", "*clojure-version*", "*command-line-args*", "*compile-files*", "*compile-path*", "*e", "*err*", "*file*", "*in*", "*ns*", "*out*", "*print-dup*", "*print-length*", "*print-level*", "*print-meta*", "*print-readably*", "*warn on reflection*", "accessor", "aclone", "add-watch", "agent", "agent-error", "agent-errors", "aget", "alength", "alias", "all-ns", "alter", "alter-meta!", "alter-var-root", "amap", "ancestors", "and", "apply", "areduce", "array-map", "as->", "aset", "aset-boolean", "aset-byte", "aset-char", "aset-double", "aset-float", "aset-int", "aset-long", "aset-short", "assert", "assoc", "assoc", "assoc", "assoc!", "assoc-in", "associative?", "atom", "await", "await-for", "bases", "bean", "bigdec", "bigdec?", "bigint", "binding", "bit-and", "bit-and-not", "bit-clear", "bit-flip", "bit-not", "bit-or", "bit-set", "bit-shift-left", "bit-shift-right", "bit-test", "bit-xor", "boolean", "boolean-array", "booleans", "bound-fn", "bound-fn*", "bound?", "butlast", "byte", "byte-array", "bytes", "case", "cast", "catch", "char", "char-array", "char?", "chars", "class", "class?", "clojure-version", "coll?", "commute", "comp", "comparator", "compare", "compare-and-set!", "compile", "complement", "concat", "cond", "cond->", "cond->>", "condp", "conj", "conj", "conj", "conj", "conj", "conj!", "cons", "constantly", "construct-proxy", "contains?", "count", "count", "counted?", "create-ns", "create-struct", "cycle", "dec", "decimal?", "declare", "dedupe", "def", "definline", "defmacro", "defmoethod", "defmulti", "defn", "defonce", "defprotocol", "defrecord", "defstruct", "deftype", "delay", "delay?", "deliver", "denominator", "deref", "deref", "derive", "descendants", "disj", "disj!", "dissoc", "dissoc!", "distinct", "distinct?", "do", "eval", "doall", "doall", "dorun", "dorun", "doseq", "doseq", "dosync", "dotimes", "doto", "double", "double-array", "double?", "doubles", "drop", "drop-last", "drop-while", "eduction", "empty", "empty?", "ensure", "enumeration-seq", "error-handler", "error-mode", "even?", "every-pred", "every?", "extend", "extend-protocol", "extend-type", "extenders", "extends?", "false?", "ffirst", "file-seq", "filter", "filterv", "finally", "find", "find-ns", "find-var", "first", "first", "flatten", "float", "float-array", "float?", "floats", "flush", "fn", "fn?", "fnext", "fnil", "for", "for", "force", "format", "frequencies", "future", "future-call", "future-cancel", "future-cancelled?", "future-done?", "future?", "gen-class", "gen-interface", "gensym", "gensym", "get", "get", "get", "get", "get", "get-in", "get-method", "get-proxy-class", "get-thread-bindings", "get-validator", "group-by", "hash", "hash-map", "hash-set", "ident?", "identical?", "identity", "if", "if-let", "if-not", "if-some", "ifn?", "import", "in-ns", "inc", "init-proxy", "instance?", "int", "int-array", "int?", "integer?", "interleave", "intern", "intern", "interpose", "into", "into-array", "ints", "io!", "isa?", "isa?", "iterate", "iterate", "iterator-seq", "juxt", "keep", "keep-indexed", "key", "keys", "keyword", "keyword?", "last", "lazy-cat", "lazy-cat", "lazy-seq", "lazy-seq", "let", "letfn", "line-seq", "list", "list?", "load", "load-file", "load-reader", "load-string", "loaded-libs", "locking", "long", "long-array", "longs", "loop", "macroexpand", "macroexpand-1", "make-array", "make-hierarchy", "map", "map-indexed", "map?", "mapcat", "mapv", "max", "max-key", "memfn", "memoize", "merge", "merge-with", "meta", "methods", "min", "min-key", "mod", "name", "namespace", "namespace-munge", "nat-int?", "neg?", "newline", "next", "nfirst", "nil?", "nnext", "non-empty", "not", "not", "not-any?", "not-every?", "ns", "ns-aliases", "ns-imports", "ns-interns", "ns-map", "ns-name", "ns-publics", "ns-refers", "ns-resolve", "ns-resolve", "ns-unalias", "ns-unmap", "nth", "nthnext", "nthrest", "num", "number?", "numerator", "object-array", "odd?", "or", "parents", "partial", "partition", "partition-all", "partition-by", "pcalls", "peek", "peek", "persistent!", "pmap", "pop", "pop", "pop!", "pop-thread-bindings", "pos-int?", "pos?", "pr", "pr-str", "pr-str", "prefer-method", "prefers", "print", "print-str", "print-str", "printf", "println", "println-str", "println-str", "prn", "prn-str", "prn-str", "promise", "proxy", "proxy-mappings", "proxy-super", "push-thread-bindings", "pvalues", "qualified-ident?", "qualified-keyword?", "qualified-symbol?", "quot", "rand", "rand-int", "rand-nth", "random-sample", "range", "ratio?", "rational?", "rationalize", "re-find", "re-groups", "re-matcher", "re-matches", "re-pattern", "re-seq", "read", "read-line", "read-string", "recur", "reduce", "reduce-kv", "reductions", "ref", "ref-history-count", "ref-max-history", "ref-min-history", "ref-set", "refer", "refer-clojure", "reify", "release-pending", "rem", "remove", "remove-all-methods", "remove-method", "remove-ns", "remove-watch", "repeat", "repeatedly", "repeatedly", "replace", "replicate", "require", "reset!", "reset-meta!", "resolve", "rest", "rest", "restart-agent", "resultset-seq", "reverse", "reversible?", "rseq", "rseq", "rsubseq", "satisfies?", "second", "select-keys", "send", "send-off", "seq", "seq?", "seqable?", "seque", "sequence", "sequential?", "set", "set", "set!", "set-error-handler", "set-error-mode", "set-validator", "set?", "short", "short-array", "shorts", "shuffle", "shutdonw-agents", "simple-ident?", "simple-keyword?", "simple-symbol?", "slurp", "some", "some->", "some->>", "some-fn", "sort", "sort-by", "sorted-map", "sorted-map-by", "sorted-set", "sorted-set-by", "sorted?", "special-symbol?", "spit", "split-at", "split-with", "str", "string?", "struct", "struct-map", "subs", "subseq", "subvec", "supers", "swap!", "symbol", "symbol?", "sync", "take", "take-last", "take-nth", "take-while", "test", "the-ns", "thread-bound?", "throw", "time", "to-array", "to-array-2d", "trampoline", "transduce", "transient", "tree-seq", "true?", "try", "type", "underive", "update", "update-in", "update-proxy", "use", "val", "vals", "var", "var-get", "var?", "vec", "vector", "vector-of", "vector?", "very-meta", "volatile!", "vreset!", "vswap!", "when", "when-first", "when-let", "when-not", "when-some", "while", "with-bindings", "with-bindings*", "with-in-str", "with-local-vars", "with-meta", "with-open", "with-out-str", "with-out-str", "with-precision", "xml-seq", "zero?", "zipmap"}

	// CMake, based on /usr/share/nvim/runtime/syntax/cmake.vim
	cmakeWords = []string{"add_compile_options", "add_custom_command", "add_custom_target", "add_definitions", "add_dependencies", "add_executable", "add_library", "add_subdirectory", "add_test", "build_command", "build_name", "cmake_host_system_information", "cmake_minimum_required", "cmake_parse_arguments", "cmake_policy", "configure_file", "create_test_sourcelist", "ctest_build", "ctest_configure", "ctest_coverage", "ctest_memcheck", "ctest_run_script", "ctest_start", "ctest_submit", "ctest_test", "ctest_update", "ctest_upload", "define_property", "enable_language", "enable_testing", "endforeach", "endfunction", "endif", "exec_program", "execute_process", "export", "export_library_dependencies", "file", "find_file", "find_library", "find_package", "find_path", "find_program", "fltk_wrap_ui", "foreach", "function", "get_cmake_property", "get_directory_property", "get_filename_component", "get_property", "get_source_file_property", "get_target_property", "get_test_property", "if", "include", "include_directories", "include_external_msproject", "include_guard", "install", "install_files", "install_programs", "install_targets", "list", "load_cache", "load_command", "macro", "make_directory", "mark_as_advanced", "math", "message", "option", "project", "remove", "separate_arguments", "set", "set_directory_properties", "set_package_properties", "set_property", "set_source_files_properties", "set_target_properties", "set_tests_properties", "source_group", "string", "subdirs", "target_compile_definitions", "target_compile_features", "target_compile_options", "target_include_directories", "target_link_libraries", "target_sources", "try_compile", "try_run", "unset", "use_mangled_mesa", "variable_requires", "variable_watch", "while", "write_file"}

	// C#
	csWords = []string{"Boolean", "Byte", "Char", "Decimal", "Double", "Int16", "Int32", "Int64", "IntPtr", "Object", "Short", "Single", "String", "UInt16", "UInt32", "UInt64", "UIntPtr", "abstract", "as", "base", "bool", "break", "byte", "case", "catch", "char", "checked", "class", "const", "continue", "decimal", "default", "delegate", "do", "double", "dynamic", "else", "enum", "event", "explicit", "extern", "false", "finally", "fixed", "float", "for", "foreach", "goto", "if", "implicit", "in", "int", "interface", "internal", "is", "lock", "long", "namespace", "new", "nint", "nuint", "null", "object", "operator", "out", "override", "params", "readonly", "ref", "return", "sbyte", "sealed", "short", "sizeof", "stackalloc", "static", "string", "struct", "switch", "this", "throw", "true", "try", "typeof", "uint", "ulong", "unchecked", "unsafe", "ushort", "using", "virtual", "void", "volatile", "while"} // private, public, protected

	// CSS
	cssWords = []string{"align-content", "align-items", "align-self", "background-color", "background-image", "background-position", "background-repeat", "background-size", "border", "border-color", "border-radius", "border-style", "border-width", "bottom", "color", "display", "flex", "flex-direction", "flex-wrap", "font-family", "font-size", "font-style", "font-weight", "height", "justify-content", "left", "letter-spacing", "line-height", "margin", "margin-bottom", "margin-left", "margin-right", "margin-top", "max-height", "max-width", "min-height", "min-width", "padding", "padding-bottom", "padding-left", "padding-right", "padding-top", "position", "right", "text-align", "text-decoration", "text-transform", "top", "width", "word-spacing", "z-index"}

	// Most common types in C and C++
	cTypes = []string{"bool", "char", "const", "constexpr", "double", "float", "inline", "int", "int16_t", "int32_t", "int64_t", "int8_t", "long", "short", "signed", "size_t", "static", "uint", "uint16_t", "uint32_t", "uint64_t", "uint8_t", "unsigned", "void", "volatile"}

	// Dart + some FFI classes
	dartWords = []string{"ArrayType", "BigInt", "DateTime", "Deprecated", "Double", "Duration", "Float", "Function", "Future", "Int16", "Int32", "Int64", "Int8", "Iterable", "List", "Map", "Null", "Object", "Pointer", "Queue", "Set", "Stream", "String", "Struct", "Uint16", "Uint32", "Uint64", "Uint8", "Uri", "Void", "abstract", "as", "assert", "async", "await", "bool", "break", "case", "catch", "class", "const", "continue", "covariant", "default", "deferred", "do", "double", "dynamic", "else", "enum", "export", "extends", "extension", "external", "factory", "false", "final", "finally", "for", "get", "hide", "if", "implements", "import", "in", "int", "interface", "is", "late", "library", "mixin", "new", "null", "num", "on", "operator", "override", "part", "required", "rethrow", "return", "set", "show", "static", "super", "switch", "sync", "this", "throw", "true", "try", "typedef", "var", "void", "while", "with", "yield"}

	// Elisp
	emacsWords = []string{"add-to-list", "defconst", "defun", "defvar", "if", "lambda", "let", "load", "nil", "require", "setq", "when"} // this should do it

	// Fortran77
	fortran77Words = []string{"assign", "backspace", "block data", "call", "close", "common", "continue", "data", "dimension", "do", "else", "else if", "end", "endfile", "endif", "entry", "equivalence", "external", "format", "function", "goto", "if", "implicit", "inquire", "intrinsic", "open", "parameter", "pause", "print", "program", "read", "return", "rewind", "rewrite", "save", "stop", "subroutine", "then", "write"}

	// Fortran90
	fortran90Words = []string{"allocatable", "allocate", "assign", "backspace", "block data", "call", "case", "close", "common", "contains", "continue", "cycle", "data", "deallocate", "dimension", "do", "else", "else if", "elsewhere", "end", "endfile", "endif", "entry", "equivalence", "exit", "external", "format", "function", "goto", "if", "implicit", "include", "inquire", "intent", "interface", "intrinsic", "module", "namelist", "nullify", "only", "open", "operator", "optional", "parameter", "pause", "pointer", "print", "private", "procedure", "program", "public", "read", "recursive", "result", "return", "rewind", "rewrite", "save", "select", "sequence", "stop", "subroutine", "target", "then", "use", "where", "while", "write"}

	// F#
	fsharpWords = []string{"abstract", "and", "as", "asr", "assert", "base", "begin", "break", "checked", "class", "component", "const", "const", "constraint", "continue", "default", "delegate", "do", "done", "downcast", "downto", "elif", "else", "end", "event", "exception", "extern", "external", "false", "finally", "fixed", "for", "fun", "function", "global", "if", "in", "include", "inherit", "inline", "interface", "internal", "land", "lazy", "let!", "let", "lor", "lsl", "lsr", "lxor", "match!", "match", "member", "mixin", "mod", "module", "mutable", "namespace", "new", "not", "null", "of", "open", "or", "override", "parallel", "private", "process", "protected", "public", "pure", "rec", "return!", "return", "sealed", "select", "sig", "static", "struct", "tailcall", "then", "to", "trait", "true", "try", "type", "upcast", "use!", "use", "val", "virtual", "void", "when", "while", "with", "yield!", "yield"}

	// GDScript
	gdscriptWords = []string{"as", "assert", "await", "break", "breakpoint", "class", "class_name", "const", "continue", "elif", "else", "enum", "export", "extends", "for", "func", "if", "INF", "is", "master", "mastersync", "match", "NAN", "onready", "pass", "PI", "preload", "puppet", "puppetsync", "remote", "remotesync", "return", "self", "setget", "signal", "static", "TAU", "tool", "var", "while", "yield"}

	// Haxe
	haxeWords = []string{"abstract", "break", "case", "cast", "catch", "class", "continue", "default", "do", "dynamic", "else", "enum", "extends", "extern", "false", "final", "for", "function", "if", "implements", "import", "in", "inline", "interface", "macro", "new", "null", "operator", "overload", "override", "package", "private", "public", "return", "static", "switch", "this", "throw", "true", "try", "typedef", "untyped", "using", "var", "while"}

	// Hardware Interface Description Language. Keywords from https://source.android.com/devices/architecture/hidl
	hidlWords = []string{"constexpr", "enum", "extends", "generates", "import", "interface", "oneway", "package", "safe_union", "struct", "typedef", "union"}

	// Inko
	inkoWords = []string{"and", "as", "asnyc", "break", "builtin", "case", "class", "else", "enum", "false", "for", "if", "impl", "import", "let", "loop", "match", "move", "mut", "next", "nil", "or", "pub", "recover", "ref", "return", "self", "static", "throw", "trait", "true", "try", "uni", "while"}

	// Just
	justWords = []string{"absolute_path", "arch", "capitalize", "clean", "env_var", "env_var_or_default", "error", "extension", "file_name", "file_stem", "include", "invocation_directory", "invocation_directory_native", "join", "just_executable", "justfile", "justfile_directory", "kebabcase", "lowercamelcase", "lowercase", "os", "os_family", "parent_directory", "path_exists", "quote", "replace", "replace_regex", "sha256", "sha256_file", "shoutykebabcase", "shoutysnakecase", "snakecase", "titlecase", "trim", "trim_end", "trim_end_match", "trim_end_matches", "trim_start", "trim_start_match", "trim_start_matches", "uppercamelcase", "uppercase", "uuid", "without_extension"}

	// Koka
	kokaWords = []string{"abstract", "alias", "as", "behind", "break", "c", "co", "con", "continue", "cs", "ctl", "effect", "elif", "else", "exists", "extend", "extern", "file", "final", "finally", "fn", "forall", "fun", "handle", "handler", "if", "import", "in", "infix", "infixl", "infixr", "initially", "inline", "interface", "js", "linear", "mask", "match", "module", "named", "noinline", "open", "override", "pub", "raw", "rec", "reference", "return", "some", "struct", "then", "type", "unsafe", "val", "value", "var", "with"}

	// Kotlin
	kotlinWords = []string{"as", "break", "by", "catch", "class", "continue", "do", "downTo", "else", "false", "for", "fun", "if", "import", "in", "interface", "is", "null", "object", "override", "package", "return", "step", "super", "suspend", "this", "throw", "true", "try", "typealias", "typeof", "val", "var", "when", "while"}

	// Lilypond
	lilypondWords = []string{"AccidentalSuggestion", "AmbitusLine", "Balloon_engraver", "BarNumber", "ChordGrid", "ChordNames", "Completion_heads_engraver", "Completion_rest_engraver", "CueVoice", "DrumStaff", "DynamicLineSpanner", "EnableGregorianDivisiones", "Engraver_group", "Ez_numbers_engraver", "Forbid_line_break_engraver", "FretBoards", "GregorianTranscriptionStaff", "Grid_line_span_engraver", "Grid_point_engraver", "HorizontalBracketText", "Horizontal_bracket_engraver", "IIJ", "IJ", "KievanStaff", "KievanVoice", "Mark_engraver", "Measure_grouping_engraver", "MensuralStaff", "MensuralVoice", "MultiMeasureRestScript", "MultiMeasureRestText", "NoteNames", "Note_heads_engraver", "Note_name_engraver", "NullVoice", "OneStaff", "Performer_group", "PianoStaff", "Pitch_squash_engraver", "R", "RemoveAllEmptyStaves", "RemoveEmptyStaves", "RhythmicStaff", "Score_engraver", "Score_performer", "Span_stem_engraver", "Staff.midiInstrument", "Staff_collecting_engraver", "Staff_symbol_engraver", "TabStaff", "TabVoice", "Text_mark_engraver", "TieColumn", "Timing", "TupletNumber", "VaticanaLyrics", "VaticanaScore", "VaticanaStaff", "VaticanaVoice", "VerticalAxisGroup", "Voice", "Volta_engraver", "X-offset", "abs-fontsize", "absolute", "accent", "accentus", "accepts", "acciaccatura", "accidental", "accidentalStyle", "add-grace-property", "add-stem-support", "add-toc-item!", "addChordShape", "addInstrumentDefinition", "addQuote", "additionalPitchPrefix", "addlyrics", "aeolian", "after", "afterGrace", "afterGraceFraction", "aikenHeads", "aikenHeadsMinor", "aikenThinHeads", "aikenThinHeadsMinor", "alias", "align-on-other", "alignAboveContext", "alignBelowContext", "allowPageTurn", "allowVoltaHook", "alterBroken", "alternative", "ambitusAfter", "ambitusAfter", "annotate-spacing", "appendToTag", "applyContext", "applyMusic", "applyOutput", "applySwing", "applySwingWithOffset", "appoggiatura", "arabicStringNumbers", "arpeggio", "arpeggio-direction", "arpeggioArrowDown", "arpeggioArrowUp", "arpeggioBracket", "arpeggioNormal", "arpeggioParenthesis", "arpeggioParenthesisDashed", "arrow-head", "articulate", "articulation-event", "ascendens", "assertBeamQuant", "assertBeamSlope", "associatedVoice", "auctum", "aug", "augmentum", "auto-first-page-number", "auto-footnote", "autoBeamOff", "autoBeamOn", "autoBeaming", "autoBreaksOff", "autoBreaksOn", "autoChange", "autoLineBreaksOff", "autoLineBreaksOn", "autoPageBreaksOff", "autoPageBreaksOn", "backslashed-digit", "balloonGrobText", "balloonLengthOff", "balloonLengthOn", "balloonText", "banjo-c-tuning", "banjo-double-c-tuning", "banjo-double-d-tuning", "banjo-modal-tuning", "banjo-open-d-tuning", "banjo-open-dm-tuning", "banjo-open-g-tuning", "bar", "barNumberCheck", "barNumberVisibility", "bartype", "base-shortest-duration", "baseMoment", "bassFigureExtendersOff", "bassFigureExtendersOn", "bassFigureStaffAlignmentDown", "bassFigureStaffAlignmentNeutral", "bassFigureStaffAlignmentUp", "beam", "beamExceptions", "beatStructure", "bendAfter", "bendHold", "bendStartLevel", "binding-offset", "blackTriangleMarkup", "blank-after-score-page-penalty", "blank-last-page-penalty", "blank-page-penalty", "bold", "book", "bookOutputName", "bookOutputSuffix", "bookTitleMarkup", "bookpart", "bookpart-level-page-numbering", "bottom-margin", "box", "bp", "bracket", "bracket", "break", "break-align-symbols", "break-visibility", "breakDynamicSpan", "breakable", "breakbefore", "breathe", "breve", "cadenzaOff", "cadenzaOn", "caesura", "caps", "cavum", "center-align", "center-column", "change", "char", "check-consistency", "choral", "choral-cautionary", "chordChanges", "chordNameExceptions", "chordNameLowercaseMinor", "chordNameSeparator", "chordNoteNamer", "chordPrefixSpacer", "chordRepeats", "chordRootNamer", "chordmode", "chords", "circle", "circulus", "clef", "clip-regions", "cm", "coda", "codaMark", "color", "column", "column-lines", "combine", "common-shortest-duration", "compound-meter", "compoundMeter", "compressEmptyMeasures", "compressMMRests", "concat", "consists", "context", "context-spec-music", "controlpitch", "countPercentRepeats", "cr", "cresc", "crescHairpin", "crescTextCresc", "crescendo-event", "crescendoSpanner", "crescendoText", "cross", "crossStaff", "cueClef", "cueClefUnset", "cueDuring", "cueDuringWithClef", "currentBarNumber", "customTabClef", "dashBang", "dashDash", "dashDot", "dashHat", "dashLarger", "dashPlus", "dashUnderscore", "deadNote", "deadNotesOff", "deadNotesOn", "debug-beam-scoring", "debug-slur-scoring", "debug-tie-scoring", "decr", "decresc", "decrescendoSpanner", "decrescendoText", "default", "default", "default-staff-staff-spacing", "defaultTimeSignature", "defaultchild", "defineBarLine", "deminutum", "denies", "descendens", "dim", "dim", "dimHairpin", "dimTextDecr", "dimTextDecresc", "dimTextDim", "dir-column", "discant", "displayLilyMusic", "displayMusic", "displayScheme", "divisioMaior", "divisioMaxima", "divisioMinima", "dodecaphonic", "dodecaphonic-first", "dodecaphonic-no-repeat", "dorian", "dotsDown", "dotsNeutral", "dotsUp", "doubleflat", "doublesharp", "downbow", "downmordent", "downprall", "draw-circle", "draw-dashed-line", "draw-dotted-line", "draw-hline", "draw-line", "draw-squiggle-line", "dropNote", "drumPitchNames", "drumPitchTable", "drumStyleTable", "drummode", "drums", "dwn", "dynamic", "dynamic-event", "dynamicDown", "dynamicNeutral", "dynamicUp", "easyHeadsOff", "easyHeadsOn", "ellipse", "enablePolymeter", "endSpanners", "endcr", "enddecr", "episemFinis", "episemInitium", "epsfile", "espressivo", "etc", "eventChords", "expandEmptyMeasures", "explicitClefVisibility", "explicitKeySignatureVisibility", "extra-offset", "extra-spacing-height", "extra-spacing-width", "eyeglasses", "featherDurations", "fermata", "ff", "fff", "ffff", "fffff", "figured-bass", "figuredBassAlterationDirection", "figuredBassPlusDirection", "figuredBassPlusStrokedAlist", "figuremode", "figures", "fill-line", "fill-with-pattern", "filled-box", "finalis", "fine", "finger", "fingeringOrientations", "first-page-number", "first-visible", "fixed", "flageolet", "flat", "flexa", "followVoice", "font-interface", "font-size", "fontCaps", "fontSize", "fonts", "fontsize", "footnote", "footnote-separator-markup", "forget", "four-string-banjo", "fp", "fraction", "freeBass", "frenchChords", "fret-diagram", "fret-diagram-interface", "fret-diagram-terse", "fret-diagram-verbose", "fromproperty", "funkHeads", "funkHeadsMinor", "general-align", "germanChords", "glide", "glide", "glissando", "glissandoMap", "grace", "gridInterval", "grob-interface", "grobdescriptions", "grow-direction", "halfopen", "halign", "harmonic", "harmonicByFret", "harmonicByRatio", "harmonicNote", "harmonicsOff", "harmonicsOn", "harp-pedal", "haydnturn", "hbracket", "hcenter-in", "header", "henzelongfermata", "henzeshortfermata", "hide", "hideKeySignature", "hideNotes", "hideSplitTiedTabNotes", "hideStaffSwitch", "horizontal-shift", "hspace", "huge", "ictus", "if", "iij", "ij", "image", "improvisationOff", "improvisationOn", "in", "inStaffSegno", "incipit", "inclinatum", "include", "indent", "inherit-acceptability", "initialContextFrom", "inner-margin", "instrumentSwitch", "inversion", "invertChords", "ionian", "italianChords", "italic", "jump", "justified-lines", "justify", "justify-field", "justify-line", "justify-string", "keepAliveInterfaces", "keepWithTag", "key", "kievanOff", "kievanOn", "killCues", "label", "laissezVibrer", "language", "languageRestore", "languageSaveAndChange", "large", "larger", "last-bottom-spacing", "layout", "layout-set-staff-size", "left-align", "left-brace", "left-column", "left-margin", "lheel", "ligature", "line", "line-width", "linea", "lineprall", "locrian", "longa", "longfermata", "lookup", "lower", "ltoe", "ly:minimal-breaking", "ly:one-line-auto-height-breaking", "ly:one-line-breaking", "ly:one-page-breaking", "ly:optimal-breaking", "ly:page-turn-breaking", "lydian", "lyricmode", "lyrics", "lyricsto", "m", "magnification->font-size", "magnify", "magnifyMusic", "magnifyStaff", "magstep", "maj", "major", "majorSevenSymbol", "make-dynamic-script", "make-relative", "makeClusters", "makeDefaultStringTuning", "marcato", "mark", "markLengthOff", "markLengthOn", "markalphabet", "markletter", "markup", "markup-markup-spacing", "markup-system-spacing", "markupMap", "markuplist", "max-systems-per-page", "maxima", "measureBarType", "measureLength", "measurePosition", "melisma", "melismaEnd", "mergeDifferentlyDottedOff", "mergeDifferentlyDottedOn", "mergeDifferentlyHeadedOff", "mergeDifferentlyHeadedOn", "mf", "midi", "midiBalance", "midiChannelMapping", "midiChorusLevel", "midiDrumPitches", "midiExpression", "midiPanPosition", "midiReverbLevel", "min-systems-per-page", "minimum-Y-extent", "minimumFret", "minimumPageTurnLength", "minimumRepeatLengthForPageTurn", "minor", "minorChordModifier", "mixed", "mixolydian", "mm", "modalInversion", "modalTranspose", "mode", "modern", "modern-cautionary", "modern-voice", "modern-voice-cautionary", "mordent", "mp", "multi-measure-rest-by-number", "musicLength", "musicMap", "musicQuotes", "musicglyph", "n", "name", "natural", "neo-modern", "neo-modern-cautionary", "neo-modern-voice", "neo-modern-voice-cautionary", "new", "newSpacingSection", "no-reset", "noBeam", "noBreak", "noChordSymbol", "noPageBreak", "noPageTurn", "nonstaff-nonstaff-spacing", "nonstaff-relatedstaff-spacing", "nonstaff-unrelatedstaff-spacing", "normal-size-sub", "normal-size-super", "normal-text", "normal-weight", "normalsize", "note", "note-by-number", "note-event", "noteNameFunction", "noteNameSeparator", "notemode", "null", "number", "numericTimeSignature", "octaveCheck", "offset", "omit", "on-the-fly", "once", "oneVoice", "open", "oriscus", "ottava", "ottavation", "ottavation-numbers", "ottavation-ordinals", "ottavation-simple-ordinals", "ottavationMarkups", "outer-margin", "output-count", "output-def", "output-suffix", "outside-staff-horizontal-padding", "outside-staff-padding", "outside-staff-priority", "oval", "overlay", "override", "override-lines", "overrideProperty", "overrideTimeSignatureSettings", "overtie", "p", "pad-around", "pad-markup", "pad-to-box", "pad-x", "page-breaking", "page-breaking-system-system-spacing", "page-count", "page-link", "page-number-type", "page-ref", "page-spacing-weight", "pageBreak", "pageTurn", "palmMute", "palmMuteOn", "paper", "paper-height", "paper-width", "parallelMusic", "parenthesize", "partCombine", "partCombineApart", "partCombineAutomatic", "partCombineChords", "partCombineDown", "partCombineForce", "partCombineListener", "partCombineSoloI", "partCombineSoloII", "partCombineUnisono", "partCombineUp", "partial", "path", "pattern", "pedalSustainStyle", "percent", "pes", "phrasingSlurDashPattern", "phrasingSlurDashed", "phrasingSlurDotted", "phrasingSlurDown", "phrasingSlurHalfDashed", "phrasingSlurHalfSolid", "phrasingSlurNeutral", "phrasingSlurSolid", "phrasingSlurUp", "phrygian", "piano", "piano-cautionary", "pitchedTrill", "pitchnames", "pointAndClickOff", "pointAndClickOn", "pointAndClickTypes", "polygon", "portato", "postscript", "pp", "ppp", "pppp", "ppppp", "prall", "pralldown", "prallmordent", "prallprall", "prallup", "preBend", "preBendHold", "predefinedDiagramTable", "predefinedFretboardsOff", "predefinedFretboardsOn", "print-all-headers", "print-first-page-number", "print-page-number", "printAccidentalNames", "printNotesLanguage", "printOctaveNames", "property-recursive", "propertyOverride", "propertyRevert", "propertySet", "propertyTweak", "propertyUnset", "pt", "pushToTag", "put-adjacent", "qr-code", "quilisma", "quoteDuring", "quotedCueEventTypes", "quotedEventTypes", "ragged-bottom", "ragged-last", "ragged-last-bottom", "ragged-right", "raise", "raiseNote", "reduceChords", "relative", "remove", "remove-empty", "remove-first", "remove-grace-property", "remove-layer", "removeWithTag", "repeat", "repeatCommands", "repeatCountVisibility", "repeatTie", "replace", "reset-footnotes-on-new-page", "resetRelativeOctave", "responsum", "rest", "rest-by-number", "rest-event", "restNumberThreshold", "restrainOpenStrings", "retrograde", "reverseturn", "revert", "revertTimeSignatureSettings", "rfz", "rgb-color", "rheel", "rhythm", "right-align", "right-brace", "right-column", "right-margin", "rightHandFinger", "romanStringNumbers", "rotate", "rounded-box", "rtoe", "sacredHarpHeads", "sacredHarpHeadsMinor", "sans", "scale", "scaleDurations", "score", "score-lines", "score-markup-spacing", "score-system-spacing", "scoreTitleMarkup", "section", "sectionLabel", "segno", "segnoMark", "self-alignment-X", "semiGermanChords", "semicirculus", "semiflat", "semisharp", "serif", "sesquiflat", "sesquisharp", "set", "set-global-staff-size", "settingsFrom", "sf", "sff", "sfz", "shape", "sharp", "shiftDurations", "shiftOff", "shiftOn", "shiftOnn", "shiftOnnn", "short-indent", "shortfermata", "showFirstLength", "showKeySignature", "showLastLength", "showStaffSwitch", "signumcongruentiae", "simple", "single", "skip", "skipBars", "skipTypesetting", "slashChordSeparator", "slashSeparator", "slashed-digit", "slashedGrace", "slashturn", "slur-event", "slurDashPattern", "slurDashed", "slurDotted", "slurDown", "slurHalfDashed", "slurHalfSolid", "slurNeutral", "slurSolid", "slurUp", "small", "smallCaps", "smaller", "snappizzicato", "sostenutoOff", "sostenutoOn", "sourcefileline", "sourcefilename", "southernHarmonyHeads", "southernHarmonyHeadsMinor", "sp", "space-alist", "spacing", "spp", "staccatissimo", "staccato", "staff-affinity", "staff-padding", "staff-space", "staff-staff-spacing", "staffHighlight", "staffgroup-staff-spacing", "start-repeat", "startAcciaccaturaMusic", "startAppoggiaturaMusic", "startGraceMusic", "startGroup", "startStaff", "startTrillSpan", "stdBass", "stdBassIV", "stdBassV", "stdBassVI", "stem-spacing-correction", "stemDown", "stemLeftBeamCount", "stemNeutral", "stemRightBeamCount", "stemUp", "stencil", "stopAcciaccaturaMusic", "stopAppoggiaturaMusic", "stopGraceMusic", "stopGroup", "stopStaff", "stopStaffHighlight", "stopTrillSpan", "stopped", "storePredefinedDiagram", "strictBeatBeaming", "string-lines", "stringNumberOrientations", "stringTuning", "stringTunings", "strokeFingerOrientations", "stropha", "strut", "styledNoteHeads", "sub", "subdivideBeams", "suggestAccidentals", "super", "sus", "sustainOff", "sustainOn", "system-count", "system-separator-markup", "system-system-spacing", "systems-per-page", "tabChordRepeats", "tabChordRepetition", "tabFullNotation", "table", "table-of-contents", "tag", "tagGroup", "taor", "teaching", "teeny", "tempo", "temporary", "tenuto", "text", "textEndMark", "textLengthOff", "textLengthOn", "textMark", "textSpannerDown", "textSpannerNeutral", "textSpannerUp", "thumb", "tie", "tieDashPattern", "tieDashed", "tieDotted", "tieDown", "tieHalfDashed", "tieHalfSolid", "tieNeutral", "tieSolid", "tieUp", "tieWaitForNote", "tied-lyric", "time", "timeSignatureFraction", "times", "tiny", "tocFormatMarkup", "tocIndentMarkup", "tocItem", "tocItemMarkup", "tocItemWithDotsMarkup", "tocTitleMarkup", "top-margin", "top-markup-spacing", "top-system-spacing", "toplevel-bookparts", "toplevel-scores", "translate", "translate-scaled", "transparent", "transpose", "transposedCueDuring", "transposition", "treCorde", "tremolo", "triangle", "trill", "tripletFeel", "tuplet", "tuplet-slur", "tupletDown", "tupletNeutral", "tupletSpan", "tupletSpannerDuration", "tupletUp", "turn", "tweak", "two-sided", "type", "typewriter", "unHideNotes", "unaCorda", "underline", "undertie", "undo", "unfold", "unfoldRepeats", "unfolded", "universal-color", "unless", "unset", "upbow", "upmordent", "upprall", "upright", "varcoda", "vcenter", "verbatim-file", "version", "versus", "verylongfermata", "veryshortfermata", "virga", "virgula", "voice", "voiceFour", "voiceFourStyle", "voiceNeutralStyle", "voiceOne", "voiceOneStyle", "voiceThree", "voiceThreeStyle", "voiceTwo", "voiceTwoStyle", "voices", "void", "volta", "volta", "volta-number", "vshape", "vspace", "walkerHeads", "walkerHeadsMinor", "whiteTriangleMarkup", "whiteout", "with", "with-color", "with-dimension", "with-dimension-from", "with-dimensions", "with-dimensions-from", "with-link", "with-outline", "with-string-transformer", "with-true-dimension", "with-true-dimensions", "with-url", "withMusicProperty", "woodwind-diagram", "wordwrap", "wordwrap-field", "wordwrap-lines", "wordwrap-string", "xNote", "xNotesOff", "xNotesOn"}

	// Lua
	luaWords = []string{"and", "break", "do", "else", "elseif", "end", "false", "for", "function", "goto", "if", "in", "local", "nil", "not", "or", "repeat", "return", "then", "true", "until", "while"}

	// Object Pascal
	objPasWords = []string{"AND", "Array", "Boolean", "Byte", "CASE", "CONST", "Char", "DO", "ELSE", "FOR", "FUNCTION", "IF", "Integer", "LABEL", "NOT", "OF", "PROCEDURE", "PROGRAM", "Pointer", "RECORD", "REPEAT", "Repeat", "String", "THEN", "TO", "TYPE", "Text", "UNTIL", "USES", "VAR", "Word", "do", "downto", "function", "nil", "of", "procedure", "program", "then", "to", "uses"}

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

	// Docker
	dockerWords = []string{"arg", "attach", "build", "cmd", "commit", "container", "copy", "cp", "create", "diff", "entrypoint", "env", "events", "exec", "export", "expose", "from", "history", "image", "images", "import", "info", "inspect", "kill", "load", "login", "logout", "logs", "network", "pause", "port", "ps", "pull", "push", "rename", "repository", "restart", "rm", "rmi", "run", "save", "search", "start", "stats", "stop", "tag", "top", "unpause", "update", "version", "volume", "wait", "workdir"}

	// Ollama
	ollamaWords = []string{"from", "parameter", "template", "system", "adapter", "license"}

	// Swift
	swiftWords = []string{"associatedtype", "class", "deinit", "enum", "extension", "fileprivate", "func", "import", "init", "inout", "internal", "let", "open", "operator", "private", "precedencegroup", "protocol", "public", "rethrows", "static", "struct", "subscript", "typealias", "var", "break", "case", "catch", "continue", "default", "defer", "do", "else", "fallthrough", "for", "guard", "if", "in", "repeat", "return", "throw", "switch", "where", "while", "Any", "as", "await", "catch", "false", "is", "nil", "rethrows", "self", "Self", "super", "throw", "throws", "true", "try", "#available", "#colorLiteral", "#elseif", "#else", "#endif", "#fileLiteral", "#if", "#imageLiteral", "#keyPath", "#selector", "#sourceLocation", "associativity", "convenience", "didSet", "dynamic", "final", "get", "indirect", "infix", "lazy", "left", "mutating", "none", "nonmutating", "optional", "override", "postfix", "precedence", "prefix", "Protocol", "required", "right", "set", "some", "Type", "unowned", "weak", "willSet"}

	// For Shell, Make and Just
	shellWords = []string{"--force", "-f", "checkout", "configure", "dd", "do", "doas", "done", "endif", "exec", "fdisk", "for", "gdisk", "ifeq", "ifneq", "in", "make", "mv", "ninja", "rm", "rmdir", "setopt", "su", "sudo", "while"}
)

func clearKeywords() {
	Keywords = make(map[string]struct{})
}

func addKeywords(addKeywords []string) {
	// Add the keywords that are to be syntax highlighted
	for _, kw := range addKeywords {
		Keywords[kw] = struct{}{}
	}
}

func addKeywordsAsUppercase(xs []string) {
	uppercase := []string{}
	for _, word := range xs {
		uppercase = append(uppercase, strings.ToUpper(word))
	}
	addKeywords(uppercase)
}

func removeKeywords(delKeywords []string) {
	// Remove keywords that should not be syntax highlighted
	for _, kw := range delKeywords {
		delete(Keywords, kw)
	}
}

func addAndRemoveKeywords(addAndDelKeywords ...[]string) {
	l := len(addAndDelKeywords)
	if l > 0 {
		addKeywords(addAndDelKeywords[0])
	}
	if l > 1 {
		removeKeywords(addAndDelKeywords[1])
	}
}

func setKeywords(addAndDelKeywords ...[]string) {
	clearKeywords()
	addAndRemoveKeywords(addAndDelKeywords...)
}

// adjustSyntaxHighlightingKeywords contains per-language adjustments to highlighting of keywords
func adjustSyntaxHighlightingKeywords(m mode.Mode) {
	switch m {
	case mode.ABC:
		addKeywords([]string{"MIDI"})
	case mode.Ada:
		addKeywords([]string{"constant", "loop", "procedure", "project"})
	case mode.Assembly:
		setKeywords(asmWords)
	case mode.Battlestar:
		setKeywords(battlestarWords)
	case mode.C3:
		setKeywords(c3Words)
	case mode.Clojure:
		setKeywords(clojureWords)
	case mode.CMake:
		addAndRemoveKeywords(cmakeWords, []string{"build", "package"})
	case mode.Config, mode.Ini, mode.FSTAB, mode.Nix:
		removeKeywords([]string{"auto", "build", "default", "for", "from", "get", "install", "int", "local", "no", "not", "package", "super", "type", "var", "with"})
		addKeywords([]string{"DB_PASSWORD", "PASSWORD", "POSTGRES_PASSWORD", "PWD", "Password", "Pwd", "SECRET", "SECRETS", "Secret", "Secrets", "bind", "password", "pwd", "secret", "secrets", "set-option", "set-window-option", "unbind", "uses"})
	case mode.CS:
		setKeywords(csWords)
	case mode.CSS:
		var massagedWords []string
		for _, word := range cssWords {
			if strings.Contains(word, "-") {
				fields := strings.Split(word, "-")
				massagedWords = append(massagedWords, fields...)
			} else {
				massagedWords = append(massagedWords, word)
			}
		}
		setKeywords(massagedWords)
		//removeKeywords([]string{"flex"}) // flex can be part of the property name and also the value
		addKeywords([]string{"animation", "events", "pointer"})
	case mode.D:
		setKeywords(dWords)
	case mode.Dart:
		setKeywords(dartWords)
	case mode.Docker:
		removeKeywords([]string{"auto", "default", "from", "install", "int", "local", "no", "not", "pull", "type", "var"})
		addKeywords(dockerWords)
		addKeywordsAsUppercase(dockerWords)
		removeKeywords([]string{"copy", "entrypoint", "env", "from", "pull", "run"}) // remove the lowercase variety of these
	case mode.Erlang:
		setKeywords(erlangWords)
	case mode.Fortran77:
		setKeywords(fortran77Words)
		addKeywordsAsUppercase(fortran77Words)
	case mode.Fortran90:
		setKeywords(fortran90Words)
	case mode.FSharp:
		setKeywords(fsharpWords)
	case mode.GDScript:
		setKeywords(gdscriptWords)
	case mode.Go:
		// TODO: Define goWords and use setKeywords instead
		addKeywords := []string{"defer", "error", "fallthrough", "func", "go", "import", "package", "print", "println", "range", "rune", "string", "uint", "uint16", "uint32", "uint64", "uint8"}
		delKeywords := []string{"False", "None", "True", "assert", "auto", "build", "char", "class", "def", "def", "del", "die", "done", "end", "fi", "final", "finally", "fn", "foreach", "from", "get", "in", "include", "is", "last", "let", "match", "mut", "next", "no", "pass", "redo", "rescue", "ret", "retry", "set", "static", "template", "then", "this", "until", "when", "where", "while", "yes"}
		addAndRemoveKeywords(addKeywords, delKeywords)
	case mode.Haxe:
		setKeywords(haxeWords)
	case mode.HIDL:
		setKeywords(hidlWords)
	case mode.Inko:
		setKeywords(inkoWords)
	case mode.AIDL:
		addKeywords(append([]string{"interface"}, hidlWords...))
		fallthrough // continue to mode.Java
	case mode.Java:
		addKeywords := []string{"package"}
		delKeywords := []string{"add", "bool", "get", "in", "local", "sub", "until"}
		addAndRemoveKeywords(addKeywords, delKeywords)
	case mode.JavaScript:
		kws := []string{"super"}
		addKeywords(kws)
	case mode.JSON:
		removeKeywords([]string{"install", "until"})
	case mode.Koka:
		setKeywords(kokaWords)
	case mode.Kotlin:
		setKeywords(kotlinWords)
	case mode.Lilypond:
		setKeywords(lilypondWords)
	case mode.Lisp:
		setKeywords(emacsWords)
	case mode.Lua, mode.Teal, mode.Terra: // use the Lua mode for Teal and Terra, for now
		setKeywords(luaWords)
	case mode.Nroff:
		addKeywords := []string{"B", "BR", "PP", "SH", "TP", "fB", "fI", "fP", "RB", "TH", "IR", "IP", "fI", "fR"}
		delKeywords := []string{"class"}
		setKeywords(addKeywords, delKeywords)
	case mode.ManPage:
		clearKeywords()
	case mode.ObjectPascal:
		addKeywords(objPasWords)
	case mode.Oak:
		addAndRemoveKeywords([]string{"fn"}, []string{"from", "new", "print"})
	case mode.Python, mode.Nim, mode.Mojo, mode.Starlark:
		addAndRemoveKeywords([]string{"type"}, []string{"append", "exit", "fn", "get", "package", "print", "until"})
	case mode.Odin:
		setKeywords(odinWords)
	case mode.Ollama:
		removeKeywords([]string{"auto", "default", "from", "install", "int", "local", "no", "not", "type", "var"})
		addKeywords(ollamaWords)
		addKeywordsAsUppercase(ollamaWords)
	case mode.PolicyLanguage: // SE Linux
		setKeywords(policyLanguageWords)
	case mode.Hare:
		addKeywords := []string{"String", "assert_eq", "char", "done", "fn", "i16", "i32", "i64", "i8", "impl", "loop", "mod", "out", "panic", "u16", "u32", "u64", "u8", "usize"}
		// "as" and "mut" are treated as special cases in the syntax package
		delKeywords := []string{"as", "build", "byte", "end", "foreach", "get", "int", "int16", "int32", "int64", "last", "map", "mut", "next", "pass", "print", "uint16", "uint32", "uint64", "until", "var"}
		addAndRemoveKeywords(addKeywords, delKeywords)
	case mode.Garnet, mode.Jakt, mode.Rust: // Originally only for Rust, split up as needed
		addKeywords := []string{"String", "assert_eq", "char", "fn", "i16", "i32", "i64", "i8", "impl", "loop", "mod", "out", "panic", "u16", "u32", "u64", "u8", "usize"}
		// "as" and "mut" are treated as special cases in the syntax package
		delKeywords := []string{"as", "build", "byte", "done", "foreach", "get", "int", "int16", "int32", "int64", "last", "map", "mut", "next", "pass", "print", "uint16", "uint32", "uint64", "until", "var"}
		if m != mode.Garnet {
			delKeywords = append(delKeywords, "end")
		}
		addAndRemoveKeywords(addKeywords, delKeywords)
	case mode.Scala:
		setKeywords(scalaWords)
	case mode.OCaml:
		setKeywords(ocamlWords)
	case mode.Elm, mode.StandardML:
		setKeywords(smlWords)
	case mode.SQL:
		addKeywords([]string{"NOT"})
	case mode.Swift:
		setKeywords(swiftWords)
	case mode.Vim:
		addKeywords([]string{"call", "echo", "elseif", "endfunction", "map", "nmap", "redraw"})
	case mode.Zig:
		setKeywords(zigWords, []string{"log"})
	case mode.GoAssembly:
		// Only highlight some words, to make them stand out
		addKeywords := []string{"cap", "close", "complex", "complex128", "complex64", "copy", "db", "dd", "dw", "imag", "int", "len", "panic", "real", "recover", "resb", "resd", "resw", "section", "syscall", "uintptr"}
		setKeywords(addKeywords)
	case mode.Just:
		addKeywords(justWords)
		fallthrough // Continue to Make
	case mode.Make:
		delKeywords := []string{"#else", "#endif", "and", "as", "build", "default", "del", "double", "exec", "export", "finally", "float", "fn", "generic", "get", "install", "local", "long", "new", "no", "package", "pass", "print", "property", "require", "ret", "set", "stop", "super", "super", "template", "type", "var", "with"}
		addAndRemoveKeywords(shellWords, delKeywords)
	case mode.Shell:
		delKeywords := []string{"#else", "#endif", "as", "build", "default", "del", "double", "exec", "false", "finally", "float", "fn", "generic", "get", "install", "long", "new", "no", "package", "pass", "print", "property", "require", "ret", "set", "super", "super", "template", "true", "type", "var", "with"}
		addAndRemoveKeywords(shellWords, delKeywords)
	case mode.Shader:
		addKeywords([]string{"buffer", "bvec2", "bvec3", "bvec4", "coherent", "dvec2", "dvec3", "dvec4", "flat", "in", "inout", "invariant", "ivec2", "ivec3", "ivec4", "layout", "mat", "mat2", "mat3", "mat4", "noperspective", "out", "precision", "readonly", "restrict", "smooth", "uniform", "uvec2", "uvec3", "uvec4", "vec2", "vec3", "vec4", "volatile", "writeonly"})
		fallthrough // Continue to C/C++ and then to the default
	case mode.Arduino, mode.C, mode.Cpp, mode.ObjC:
		addKeywords := []string{"int8_t", "uint8_t", "int16_t", "uint16_t", "int32_t", "uint32_t", "int64_t", "uint64_t", "size_t"}
		delKeywords := []string{"ret", "static"} // static is treated separately, as a special keyword
		addAndRemoveKeywords(addKeywords, delKeywords)
		fallthrough // Continue to the default
	case mode.Text:
		clearKeywords()
	default:
		addKeywords := []string{"elif", "endif", "ifeq", "ifneq"}
		delKeywords := []string{"build", "done", "package", "require", "set", "super", "type", "when"}
		addAndRemoveKeywords(addKeywords, delKeywords)
	}
}

// SingleLineCommentMarker will return the string that starts a single-line
// comment for the current language mode the editor is in.
func (e *Editor) SingleLineCommentMarker() string {
	switch e.mode {
	case mode.ABC, mode.Lilypond, mode.Perl, mode.Prolog:
		return "%"
	case mode.Amber:
		return "!!"
	case mode.Assembly, mode.Ini:
		return ";"
	case mode.Basic:
		return "'"
	case mode.Bat:
		return "@rem" // or rem or just ":" ...
	case mode.Algol68, mode.Bazel, mode.CMake, mode.Config, mode.Crystal, mode.Docker, mode.FSTAB, mode.GDScript, mode.Ignore, mode.Just, mode.Make, mode.Nim, mode.Nix, mode.Mojo, mode.PolicyLanguage, mode.Python, mode.R, mode.Ruby, mode.Shell, mode.Starlark:
		return "#"
	case mode.Clojure, mode.Lisp:
		return ";;"
	case mode.Email:
		return "GIT:"
	case mode.Fortran77:
		return "*" // TODO: Also add "C", "c" and all the others
	case mode.Fortran90:
		return "!" // TODO: Only at the start of lines
	case mode.OCaml, mode.StandardML:
		// Not applicable, just return the multiline comment start marker
		return "(*"
	case mode.Ada, mode.Agda, mode.Elm, mode.Garnet, mode.Haskell, mode.Lua, mode.Nmap, mode.SQL, mode.Teal, mode.Terra:
		return "--"
	case mode.M4:
		return "dnl"
	case mode.Nroff:
		return `.\"`
	case mode.ObjectPascal:
		return "{"
	case mode.ReStructured:
		return "["
	case mode.Vim:
		return "\""
	default:
		return "//"
	}
}
