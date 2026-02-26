package orchideous

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// systemIncludeDirs returns the system include directories.
func systemIncludeDirs() []string {
	dirs := []string{}
	if fileExists("/usr/include") {
		dirs = append(dirs, "/usr/include")
	}
	cxx := findCompiler(false, false)
	if cxx != "" {
		out, err := exec.Command(cxx, "-dumpmachine").Output()
		if err == nil {
			machine := strings.TrimSpace(string(out))
			machineDir := "/usr/include/" + machine
			if fileExists(machineDir) {
				dirs = append(dirs, machineDir)
			}
		}
	}
	if fileExists("/usr/local/include") {
		dirs = append(dirs, "/usr/local/include")
	}
	if fileExists("/usr/pkg/include") {
		dirs = append(dirs, "/usr/pkg/include")
	}
	return dirs
}

// pkgConfigFlags runs pkg-config for a given package name and returns the flags.
func pkgConfigFlags(pkg string) string {
	out, err := exec.Command("pkg-config", "--cflags", "--libs", pkg).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// hasPkgConfig checks if pkg-config is available.
func hasPkgConfig() bool {
	_, err := exec.LookPath("pkg-config")
	return err == nil
}

// bestGtkPkg returns the best available GTK pkg-config name, preferring the newest version.
func bestGtkPkg() string {
	for _, pkg := range []string{"gtk4", "gtk+-3.0", "gtk+-2.0"} {
		if pkgConfigFlags(pkg) != "" {
			return pkg
		}
	}
	return "gtk4"
}

// bestVtePkg returns the best available VTE pkg-config name, matching the GTK version in use.
func bestVtePkg() string {
	gtk := bestGtkPkg()
	if gtk == "gtk4" {
		if pkgConfigFlags("vte-2.91-gtk4") != "" {
			return "vte-2.91-gtk4"
		}
	}
	for _, pkg := range []string{"vte-2.91-gtk4", "vte-2.91"} {
		if pkgConfigFlags(pkg) != "" {
			return pkg
		}
	}
	return "vte-2.91-gtk4"
}

// sfmlMajorVersion returns the installed SFML major version (2 or 3), or 0 if unavailable.
func sfmlMajorVersion() int {
	out, err := exec.Command("pkg-config", "--modversion", "sfml-system").Output()
	if err != nil {
		return 0
	}
	ver := strings.TrimSpace(string(out))
	if strings.HasPrefix(ver, "3") {
		return 3
	}
	if strings.HasPrefix(ver, "2") {
		return 2
	}
	return 0
}

// pkgNameFromInclude guesses the pkg-config package name from an include path.
func pkgNameFromInclude(inc string) string {
	lower := strings.ToLower(inc)

	// Well-known mappings
	mappings := map[string]string{
		"sdl2/sdl.h":          "sdl2",
		"sdl2/sdl_image.h":    "SDL2_image",
		"sdl2/sdl_mixer.h":    "SDL2_mixer",
		"sdl2/sdl_ttf.h":      "SDL2_ttf",
		"sdl2/sdl_net.h":      "SDL2_net",
		"gtk/gtk.h":           bestGtkPkg(),
		"vte/vte.h":           bestVtePkg(),
		"gl/gl.h":             "gl",
		"gl/glew.h":           "glew",
		"gl/glut.h":           "glu",
		"gl/freeglut.h":       "freeglut",
		"glfw/glfw3.h":        "glfw3",
		"al/al.h":             "openal",
		"al/alc.h":            "openal",
		"vulkan/vulkan.h":     "vulkan",
		"x11/xlib.h":          "x11",
		"x11/xutil.h":         "x11",
		"libconfig.h++":       "libconfig++",
		"libconfig.h":         "libconfig",
		"fcgiapp.h":           "fcgi",
		"pipewire/pipewire.h": "libpipewire-0.3",
		"rtaudio/rtaudio.h":   "rtaudio",
		"raylib.h":            "raylib",
	}

	for pattern, pkg := range mappings {
		if lower == pattern {
			return pkg
		}
	}

	// SFML includes
	if strings.HasPrefix(lower, "sfml/") {
		component := strings.ToLower(strings.TrimSuffix(filepath.Base(inc), filepath.Ext(inc)))
		return "sfml-" + component
	}

	// SDL2/SDL_* -> SDL2_*
	if strings.HasPrefix(lower, "sdl2/sdl_") {
		word := "SDL2_" + inc[9:]
		word = strings.TrimSuffix(word, filepath.Ext(word))
		return word
	}

	// GLM
	if strings.HasPrefix(lower, "glm/") {
		return "glm"
	}

	// Qt includes (start with Q) - handled separately
	if strings.HasPrefix(inc, "Q") {
		return ""
	}

	// boost/ includes don't have pkg-config
	if strings.HasPrefix(lower, "boost/") {
		return ""
	}

	// Use the first path component as pkg-config name
	if strings.Contains(inc, "/") {
		return strings.Split(inc, "/")[0]
	}

	return ""
}

// resolveExtraFlags returns additional link/compile flags for special includes.
func resolveExtraFlags(includes []string, win64 bool) (cflags, ldflags []string) {
	hasPkg := hasPkgConfig()
	isDarwin := runtime.GOOS == "darwin"
	hasFrameworks := fileExists("/Library/Frameworks")
	hasSysFrameworks := fileExists("/System/Library/Frameworks")

	for _, inc := range includes {
		lower := strings.ToLower(inc)

		// Thread support
		if inc == "thread" || inc == "mutex" || inc == "future" ||
			inc == "condition_variable" || inc == "pthread.h" || inc == "new" || inc == "dlfcn.h" {
			ldflags = appendUnique(ldflags, "-ldl")
			ldflags = appendUnique(ldflags, "-pthread")
			ldflags = appendUnique(ldflags, "-lpthread")
		}

		// SFML on macOS: add OpenGL framework + -stdlib=libc++ for clang
		if strings.HasPrefix(lower, "sfml/") {
			if v := sfmlMajorVersion(); v > 0 {
				cflags = appendUnique(cflags, fmt.Sprintf("-DSFML_MAJOR_VERSION=%d", v))
			}
			if hasFrameworks && !win64 {
				cflags = appendUnique(cflags, "-I/usr/local/include")
				ldflags = appendUnique(ldflags, "-F/Library/Frameworks")
				ldflags = appendUnique(ldflags, "-framework")
				ldflags = appendUnique(ldflags, "OpenGL")
			} else if win64 {
				ldflags = appendUnique(ldflags, "-lopengl32")
			}
			if hasPkg {
				if flags := pkgConfigFlags("gl"); flags != "" {
					cflags, ldflags = mergeFlags(cflags, ldflags, flags)
				}
			}
		}

		// OpenGL / GLUT / GLFW
		if strings.HasPrefix(lower, "gl/") || strings.HasPrefix(lower, "opengl/") ||
			strings.HasPrefix(lower, "glut/") || strings.HasPrefix(lower, "glfw/") ||
			strings.Contains(lower, "opengl") {
			if hasFrameworks && !win64 {
				cflags = appendUnique(cflags, "-I/usr/local/include")
				ldflags = appendUnique(ldflags, "-F/Library/Frameworks")
				ldflags = appendUnique(ldflags, "-framework")
				ldflags = appendUnique(ldflags, "OpenGL")
			} else if win64 {
				ldflags = appendUnique(ldflags, "-lopengl32")
			} else if hasPkg {
				if flags := pkgConfigFlags("gl"); flags != "" {
					cflags, ldflags = mergeFlags(cflags, ldflags, flags)
				}
			} else {
				for _, libPath := range []string{"/usr/lib", "/usr/lib/x86_64-linux-gnu", "/usr/local/lib", "/usr/pkg/lib"} {
					if fileExists(filepath.Join(libPath, "libGL.so")) {
						ldflags = appendUnique(ldflags, "-lGL")
						break
					}
				}
			}
		}

		// GLUT specifically
		if strings.HasSuffix(lower, "/glut.h") || strings.HasSuffix(lower, "/freeglut.h") || strings.HasPrefix(lower, "glut/") {
			if hasFrameworks && !win64 {
				ldflags = appendUnique(ldflags, "-framework")
				ldflags = appendUnique(ldflags, "GLUT")
			} else if win64 {
				ldflags = appendUnique(ldflags, "-lglu32")
			} else if hasPkg {
				if flags := pkgConfigFlags("glu"); flags != "" {
					cflags, ldflags = mergeFlags(cflags, ldflags, flags)
				} else if flags := pkgConfigFlags("freeglut"); flags != "" {
					cflags, ldflags = mergeFlags(cflags, ldflags, flags)
				}
			} else {
				for _, libPath := range []string{"/usr/lib", "/usr/lib/x86_64-linux-gnu", "/usr/local/lib", "/usr/pkg/lib"} {
					if fileExists(filepath.Join(libPath, "libglut.so")) {
						ldflags = appendUnique(ldflags, "-lglut")
						break
					}
				}
			}
		}

		// GLEW
		if strings.HasSuffix(lower, "/glew.h") {
			if win64 {
				ldflags = appendUnique(ldflags, "-lglew32")
			}
			if hasPkg {
				if flags := pkgConfigFlags("glew"); flags != "" {
					cflags, ldflags = mergeFlags(cflags, ldflags, flags)
				}
			}
		}

		// OpenAL
		if strings.HasPrefix(lower, "al/") || strings.HasPrefix(lower, "openal") || strings.Contains(lower, "/al.h") {
			if hasSysFrameworks && !win64 {
				cflags = appendUnique(cflags, "-I/System/Library/Frameworks/OpenAL.framework/Headers")
				ldflags = appendUnique(ldflags, "-F/System/Library/Frameworks")
				ldflags = appendUnique(ldflags, "-framework")
				ldflags = appendUnique(ldflags, "OpenAL")
			} else if win64 {
				ldflags = appendUnique(ldflags, "-lopenal32")
			} else if hasPkg {
				if flags := pkgConfigFlags("openal"); flags != "" {
					cflags, ldflags = mergeFlags(cflags, ldflags, flags)
				}
			} else {
				for _, libPath := range []string{"/usr/lib", "/usr/lib/x86_64-linux-gnu", "/usr/local/lib", "/usr/pkg/lib"} {
					if fileExists(filepath.Join(libPath, "libopenal.so")) {
						ldflags = appendUnique(ldflags, "-lopenal")
						break
					}
				}
			}
		}

		// GTK export-dynamic
		if lower == "gtk/gtk.h" {
			ldflags = appendUnique(ldflags, "-Wl,-export-dynamic")
		}

		// SDL2_* sub-libraries
		if strings.HasPrefix(lower, "sdl2/sdl_") && hasPkg {
			word := "SDL2_" + inc[9:]
			word = strings.TrimSuffix(word, filepath.Ext(word))
			if flags := pkgConfigFlags(word); flags != "" {
				cflags, ldflags = mergeFlags(cflags, ldflags, flags)
			}
		}

		// Vulkan
		if strings.HasPrefix(lower, "vulkan/") {
			if hasPkg {
				if flags := pkgConfigFlags("vulkan"); flags != "" {
					cflags, ldflags = mergeFlags(cflags, ldflags, flags)
				}
			} else {
				ldflags = appendUnique(ldflags, "-lvulkan")
			}
		}

		// Qt includes that start with Q - suppress some warnings
		if strings.HasPrefix(inc, "Q") {
			cflags = appendUnique(cflags, "-Wno-class-memaccess")
			cflags = appendUnique(cflags, "-Wno-pedantic")
			// Check for qt include dir
			for _, sysDir := range systemIncludeDirs() {
				qtDir := filepath.Join(sysDir, "qt")
				if fileExists(qtDir) {
					cflags = appendUnique(cflags, "-I"+qtDir)
				}
			}
		}

		// GLM - suppress shadow warnings
		if strings.HasPrefix(lower, "glm/") {
			cflags = appendUnique(cflags, "-Wno-shadow")
		}

		// macOS framework detection for arbitrary includes
		if isDarwin && hasFrameworks && !win64 {
			firstWord := lower
			if strings.Contains(inc, "/") {
				firstWord = strings.Split(lower, "/")[0]
			}
			frameworkPath := "/Library/Frameworks/" + firstWord + ".framework"
			if fileExists(frameworkPath) {
				cflags = appendUnique(cflags, "-I/usr/local/include")
				ldflags = appendUnique(ldflags, "-F/Library/Frameworks")
				ldflags = appendUnique(ldflags, "-framework")
				ldflags = appendUnique(ldflags, firstWord)
			}
		}
	}

	return cflags, ldflags
}

// mergeFlags splits pkg-config output and adds to cflags/ldflags.
func mergeFlags(cflags, ldflags []string, flags string) ([]string, []string) {
	for f := range strings.FieldsSeq(flags) {
		if strings.HasPrefix(f, "-l") || strings.HasPrefix(f, "-L") || strings.HasPrefix(f, "-Wl,") {
			ldflags = appendUnique(ldflags, f)
		} else {
			cflags = appendUnique(cflags, f)
		}
	}
	return cflags, ldflags
}

// findCompiler returns the path to the C++ or C compiler.
func findCompiler(useClang bool, isC bool) string {
	if useClang {
		if isC {
			if p, err := exec.LookPath("clang"); err == nil {
				return p
			}
		} else {
			if p, err := exec.LookPath("clang++"); err == nil {
				return p
			}
		}
	}

	// Check CXX/CC environment variable
	if isC {
		if cc := os.Getenv("CC"); cc != "" {
			if p, err := exec.LookPath(cc); err == nil {
				return p
			}
		}
	} else {
		if cxx := os.Getenv("CXX"); cxx != "" {
			if p, err := exec.LookPath(cxx); err == nil {
				return p
			}
		}
	}

	// Try common compilers
	if isC {
		for _, compiler := range []string{"gcc", "cc", "clang"} {
			if p, err := exec.LookPath(compiler); err == nil {
				return p
			}
		}
	} else {
		for _, compiler := range []string{"g++", "clang++", "c++"} {
			if p, err := exec.LookPath(compiler); err == nil {
				return p
			}
		}
	}
	return ""
}

// findWin64Compiler returns the path to the mingw64 cross-compiler.
func findWin64Compiler(isC bool) string {
	if isC {
		for _, compiler := range []string{"x86_64-w64-mingw32-gcc", "i686-w64-mingw32-gcc"} {
			if p, err := exec.LookPath(compiler); err == nil {
				return p
			}
		}
	} else {
		for _, compiler := range []string{"x86_64-w64-mingw32-g++", "i686-w64-mingw32-g++"} {
			if p, err := exec.LookPath(compiler); err == nil {
				return p
			}
		}
	}
	return ""
}

// dirDefines generates -D flags for data/img/shader directories.
func dirDefines() []string {
	var defs []string
	dirTypes := map[string]string{
		"img":       "IMGDIR",
		"data":      "DATADIR",
		"shaders":   "SHADERDIR",
		"shader":    "SHADERDIR",
		"share":     "SHAREDIR",
		"resources": "RESOURCEDIR",
		"resource":  "RESOURCEDIR",
		"res":       "RESDIR",
		"scripts":   "SCRIPTDIR",
	}

	for dir, define := range dirTypes {
		path := ""
		if fileExists(dir) {
			path = dir + "/"
		} else if fileExists(filepath.Join("..", dir)) {
			path = filepath.Join("..", dir) + "/"
		}
		if path != "" {
			defs = append(defs, `-D`+define+`="`+path+`"`)
		}
	}
	return defs
}

// compilerSupportsStd checks if the compiler supports a given -std= flag.
func compilerSupportsStd(compiler, std string) bool {
	cmd := exec.Command("sh", "-c",
		"echo 'int main(){}' | "+compiler+" -std="+std+" -x c++ -fsyntax-only - 2>/dev/null")
	return cmd.Run() == nil
}

// bestStdFlag returns the best C++ standard flag the compiler supports.
func bestStdFlag(compiler string) string {
	for _, std := range []string{"c++23", "c++2b", "c++20", "c++2a", "c++17", "c++14", "c++11"} {
		if compilerSupportsStd(compiler, std) {
			return std
		}
	}
	return "c++17"
}

func appendUnique(slice []string, val string) []string {
	if slices.Contains(slice, val) {
		return slice
	}
	return append(slice, val)
}

// installDirDefines generates -D flags pointing to installed paths.
func installDirDefines(prefix string) []string {
	var defs []string
	dirTypes := map[string]string{
		"img":       "IMGDIR",
		"data":      "DATADIR",
		"shaders":   "SHADERDIR",
		"shader":    "SHADERDIR",
		"share":     "SHAREDIR",
		"resources": "RESOURCEDIR",
		"resource":  "RESOURCEDIR",
		"res":       "RESDIR",
		"scripts":   "SCRIPTDIR",
	}

	for dir, define := range dirTypes {
		if fileExists(dir) || fileExists(filepath.Join("..", dir)) {
			path := filepath.Join(prefix, dir) + "/"
			defs = append(defs, `-D`+define+`="`+path+`"`)
		}
	}
	return defs
}

// isLinux returns true if running on Linux.
func isLinux() bool {
	return runtime.GOOS == "linux"
}

// isCompilerGCC checks if a compiler path looks like gcc/g++.
func isCompilerGCC(compiler string) bool {
	base := filepath.Base(compiler)
	for _, needle := range []string{"g++", "gcc"} {
		idx := strings.Index(base, needle)
		if idx < 0 {
			continue
		}
		if idx == 0 || !isLetter(base[idx-1]) {
			return true
		}
	}
	return false
}

func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// isCompilerClang checks if a compiler path looks like clang/clang++.
func isCompilerClang(compiler string) bool {
	base := filepath.Base(compiler)
	return strings.Contains(base, "clang")
}

// Qt6 hardcoded flags (from build.py)
const qt6CxxFlags = "-I/usr/include/qt6 -I/usr/include/qt6/Qt3DAnimation -I/usr/include/qt6/Qt3DCore -I/usr/include/qt6/Qt3DExtras -I/usr/include/qt6/Qt3DInput -I/usr/include/qt6/Qt3DLogic -I/usr/include/qt6/Qt3DQuick -I/usr/include/qt6/Qt3DQuickAnimation -I/usr/include/qt6/Qt3DQuickExtras -I/usr/include/qt6/Qt3DQuickInput -I/usr/include/qt6/Qt3DQuickRender -I/usr/include/qt6/Qt3DQuickScene2D -I/usr/include/qt6/Qt3DRender -I/usr/include/qt6/QtConcurrent -I/usr/include/qt6/QtCore -I/usr/include/qt6/QtCore5Compat -I/usr/include/qt6/QtDBus -I/usr/include/qt6/QtDesigner -I/usr/include/qt6/QtDesignerComponents -I/usr/include/qt6/QtDeviceDiscoverySupport -I/usr/include/qt6/QtEglFSDeviceIntegration -I/usr/include/qt6/QtEglFsKmsGbmSupport -I/usr/include/qt6/QtEglFsKmsSupport -I/usr/include/qt6/QtFbSupport -I/usr/include/qt6/QtGui -I/usr/include/qt6/QtHelp -I/usr/include/qt6/QtInputSupport -I/usr/include/qt6/QtKmsSupport -I/usr/include/qt6/QtLabsAnimation -I/usr/include/qt6/QtLabsFolderListModel -I/usr/include/qt6/QtLabsQmlModels -I/usr/include/qt6/QtLabsSettings -I/usr/include/qt6/QtLabsSharedImage -I/usr/include/qt6/QtLabsWavefrontMesh -I/usr/include/qt6/QtNetwork -I/usr/include/qt6/QtNetworkAuth -I/usr/include/qt6/QtOpenGL -I/usr/include/qt6/QtOpenGLWidgets -I/usr/include/qt6/QtPacketProtocol -I/usr/include/qt6/QtPrintSupport -I/usr/include/qt6/QtQml -I/usr/include/qt6/QtQmlCompiler -I/usr/include/qt6/QtQmlDebug -I/usr/include/qt6/QtQmlDom -I/usr/include/qt6/QtQmlLocalStorage -I/usr/include/qt6/QtQmlModels -I/usr/include/qt6/QtQmlWorkerScript -I/usr/include/qt6/QtQuick -I/usr/include/qt6/QtQuick3D -I/usr/include/qt6/QtQuick3DAssetImport -I/usr/include/qt6/QtQuick3DIblBaker -I/usr/include/qt6/QtQuick3DParticles -I/usr/include/qt6/QtQuick3DRuntimeRender -I/usr/include/qt6/QtQuick3DUtils -I/usr/include/qt6/QtQuickControls2 -I/usr/include/qt6/QtQuickControls2Impl -I/usr/include/qt6/QtQuickLayouts -I/usr/include/qt6/QtQuickParticles -I/usr/include/qt6/QtQuickShapes -I/usr/include/qt6/QtQuickTemplates2 -I/usr/include/qt6/QtQuickTest -I/usr/include/qt6/QtQuickWidgets -I/usr/include/qt6/QtShaderTools -I/usr/include/qt6/QtSql -I/usr/include/qt6/QtSvg -I/usr/include/qt6/QtSvgWidgets -I/usr/include/qt6/QtTest -I/usr/include/qt6/QtTools -I/usr/include/qt6/QtUiPlugin -I/usr/include/qt6/QtUiTools -I/usr/include/qt6/QtWaylandClient -I/usr/include/qt6/QtWaylandCompositor -I/usr/include/qt6/QtWidgets -I/usr/include/qt6/QtXml"
const qt6LinkFlags = "-lQt6Concurrent -lQt6Core -lQt6DBus -lQt6EglFSDeviceIntegration -lQt6EglFsKmsGbmSupport -lQt6EglFsKmsSupport -lQt6Gui -lQt6Network -lQt6OpenGL -lQt6OpenGLWidgets -lQt6PrintSupport -lQt6Sql -lQt6Test -lQt6Widgets -lQt6XcbQpa -lQt6Xml"
