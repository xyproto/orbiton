package main

// TODO: Use a different syntax highlighting package, with support for many different programming languages
import "github.com/xyproto/syntax"

var (
	// Based on /usr/share/nvim/runtime/syntax/cmake.vim
	cmakeKeywords = []string{"add_compile_options", "add_custom_command", "add_custom_target", "add_definitions", "add_dependencies", "add_executable", "add_library", "add_subdirectory", "add_test", "build_command", "build_name", "cmake_host_system_information", "cmake_minimum_required", "cmake_parse_arguments", "cmake_policy", "configure_file", "create_test_sourcelist", "ctest_build", "ctest_configure", "ctest_coverage", "ctest_memcheck", "ctest_run_script", "ctest_start", "ctest_submit", "ctest_test", "ctest_update", "ctest_upload", "define_property", "enable_language", "exec_program", "execute_process", "export", "export_library_dependencies", "file", "find_file", "find_library", "find_package", "find_path", "find_program", "fltk_wrap_ui", "foreach", "function", "get_cmake_property", "get_directory_property", "get_filename_component", "get_property", "get_source_file_property", "get_target_property", "get_test_property", "if", "include", "include_directories", "include_external_msproject", "include_guard", "install", "install_files", "install_programs", "install_targets", "list", "load_cache", "load_command", "macro", "make_directory", "mark_as_advanced", "math", "message", "option", "project", "remove", "separate_arguments", "set", "set_directory_properties", "set_package_properties", "set_property", "set_source_files_properties", "set_target_properties", "set_tests_properties", "source_group", "string", "subdirs", "target_compile_definitions", "target_compile_features", "target_compile_options", "target_include_directories", "target_link_libraries", "target_sources", "try_compile", "try_run", "unset", "use_mangled_mesa", "variable_requires", "variable_watch", "while", "write_file"}

	emacsKeywords = []string{"defun", "require", "if", "when", "setq", "add-to-list", "lambda", "defvar", "defconst"} // this should do it
)

// SingleLineCommentMarker will return the string that starts a single-line
// comment for the current language mode the editor is in.
func (e *Editor) SingleLineCommentMarker() string {
	switch e.mode {
	case modeShell, modePython, modeCMake:
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

// adjustSyntaxHighlightingKeywords contains per-language adjustments to highlighting of keywords
func adjustSyntaxHighlightingKeywords(mode Mode) {
	var addKeywords, delKeywords []string
	switch mode {
	case modeGo:
		addKeywords = []string{"fallthrough", "string", "print", "println", "range", "defer"}
		delKeywords = []string{"mut", "pass", "build", "None", "char"}
	case modeLisp:
		syntax.Keywords = make(map[string]struct{})
		addKeywords = emacsKeywords
	case modeCMake:
		addKeywords = cmakeKeywords
		delKeywords = append(delKeywords, []string{"build", "package"}...)
	case modeShell:
		delKeywords = []string{"float", "with", "exec", "long", "double", "no", "pass"}
		fallthrough
	default:
		delKeywords = append(delKeywords, []string{"build", "package"}...)
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
