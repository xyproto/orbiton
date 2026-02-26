## Orchideous <img src="img/orchideous.svg" width="128" align="right">

![Standard](https://img.shields.io/badge/C%2B%2B-23-blue.svg)
[![CI](https://github.com/xyproto/orchideous/actions/workflows/ci.yml/badge.svg)](https://github.com/xyproto/orchideous/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-BSD3-blue.svg)](https://opensource.org/licenses/BSD-3-Clause)

Zero-configuration build tool for C and C++ projects.

Have you ever had a single `main.cpp` file that you just want to compile, without having to make sure the order of flags are correct and ideally without having to provide any flags at all?

**Orchideous** (or `oh` for short) handles compiler detection, flag ordering, library discovery via `pkg-config`, incremental rebuilds, testing, formatting, cross-compilation, and more — all without a single configuration file.

It should be possible to compile all of the examples in the `examples` directory, simply by running `oh` in each directory, as long as the right packages and libraries have been installed.

This is a Go port of [xyproto/cxx](https://github.com/xyproto/cxx) (which does approximately the same, but uses Python + Scons instead).

## Quick Start

```sh
oh              # build the project
oh run          # build and run
oh clean        # remove built files
```

No configuration files are needed, but the project needs to either be very simple (a single `main.cpp`) or have an `oh`-friendly directory structure.

The auto-detection of external libraries and headers relies on them being included in the main source file.

## Installation

### Arch Linux

Install `orchideous` from AUR, or build from source:

```sh
git clone https://github.com/xyproto/orchideous
cd orchideous
make
sudo make install
```

### Other Linux distros, FreeBSD, NetBSD, macOS

```sh
git clone https://github.com/xyproto/orchideous
cd orchideous
make
sudo make install    # use gmake on BSD
```

Or with `go install`:

```sh
go install github.com/xyproto/orchideous/cmd/oh@latest
sudo ln -sf ~/go/bin/oh /usr/local/bin/oh
```

## All Commands

```
oh                  build the project
oh run              build and run
oh debug            debug build and launch debugger (gdb/cgdb)
oh debugbuild       debug build (without launching debugger)
oh debugnosan       debug build (without sanitizers)
oh opt              optimized build
oh strict           build with strict warning flags
oh sloppy           build with sloppy flags
oh small            build a smaller executable
oh tiny             build a tiny executable (+ sstrip/upx)
oh clang            build using clang++
oh clangdebug       debug build using clang++ (launches lldb)
oh clangstrict      use clang++ and strict flags
oh clangsloppy      use clang++ and sloppy flags
oh clangrebuild     clean and build with clang++
oh clangtest        build and run tests with clang++
oh clean            remove built files
oh fastclean        only remove executable and *.o
oh rebuild          clean and build
oh test             build and run tests
oh testbuild        build tests (without running)
oh rec              profile-guided optimization (build, run, rebuild)
oh fmt              format source code with clang-format
oh cmake            generate CMakeLists.txt
oh cmake ninja      generate CMakeLists.txt and build with ninja
oh ninja            build using existing CMakeLists.txt and ninja
oh ninja_install    install from ninja build
oh ninja_clean      clean ninja build
oh pro              generate QtCreator project file
oh install          install the project (PREFIX, DESTDIR)
oh pkg              package the project into pkg/
oh export           export a standalone Makefile and build.sh
oh make             generate a standalone Makefile
oh script           generate build.sh and clean.sh
oh valgrind         build and profile with valgrind
oh win64            cross-compile for 64-bit Windows
oh smallwin64       small win64 build
oh tinywin64        tiny win64 build
oh zap              build using zapcc++
oh version          show version
oh -C <dir> ...     run in the given directory
```

## Example Use

Create a **main.cpp** file:

```c++
#include <cstdlib>
#include <iomanip>
#include <iostream>
#include <ostream>
#include <string>

using namespace std::string_literals;

class Point {
public:
    double x;
    double y;
    double z;
};

std::ostream& operator<<(std::ostream& output, const Point& p)
{
    using std::setfill;
    using std::setw;
    output << "{ "s << setfill(' ') << setw(3) << p.x << ", "s << setfill(' ') << setw(3) << p.y
           << ", "s << setfill(' ') << setw(3) << p.z << " }"s;
    return output;
}

Point operator+(const Point& a, const Point& b)
{
    return Point { .x = a.x + b.x, .y = a.y + b.y, .z = a.z + b.z };
}

Point operator*(const Point& a, const Point& b)
{
    return Point { .x = a.x * b.x, .y = a.y * b.y, .z = a.z * b.z };
}

int main(int argc, char** argv)
{
    Point p1 { .x = 1, .y = 2, .z = 3 };
    Point p2 { .y = 42 };

    using std::cout;
    using std::endl;

    cout << "     p1 = " << p1 << endl;
    cout << "     p2 = " << p2 << endl;
    cout << "p1 + p2 = " << p1 + p2 << endl;
    cout << "p1 * p2 = " << p1 * p2 << endl;

    return EXIT_SUCCESS;
}
```

Then build and run:

```sh
oh run
```

Rebuild from scratch:

```sh
oh rebuild
```

Build with profile-guided optimization:

```sh
oh rec    # builds, runs (collecting profiling data), then rebuilds with PGO
oh        # subsequent builds use the profiling data
```

## Directory Structure

```
myproject/
├── main.cpp              # main source (or main.cc, main.c)
├── include/              # project headers (.h, .hpp)
│   └── hello.h
├── common/               # shared source files
│   ├── hello.cpp
│   └── hello_test.cpp    # test file (must contain main())
├── img/                  # images
├── shaders/              # shaders
├── data/                 # data files
└── shared/               # optional data files
```

* The main source file can live in the project root or `src/`.
* The executable name matches the parent directory name.
* Files ending with `_test.*` are compiled separately by `oh test`.
* `include/` and `common/` can also be at `../include` and `../common`.

## Defines

These defines are passed to the compiler, with paths that work both during development and after installation:

| Define | Development | Installed |
|---|---|---|
| `DATADIR` | `./data` or `../data` | `$PREFIX/share/$app/data` |
| `IMGDIR` | `./img` or `../img` | `$PREFIX/share/$app/img` |
| `SHADERDIR` | `./shaders` or `../shaders` | `$PREFIX/share/$app/shaders` |
| `SHAREDIR` | `./share` or `../share` | `$PREFIX/share/$app` |
| `RESOURCEDIR` | `./resources` or `../resources` | `$PREFIX/share/$app/resources` |
| `RESDIR` | `./res` or `../res` | `$PREFIX/share/$app/res` |

See `examples/sdl2`, `examples/win64crate` (uses `IMGDIR`) and `examples/mixer` (uses `RESOURCEDIR`).

## Testing

* Source files can have corresponding `_test` files (e.g. `quaternions.cc` → `quaternions_test.cc`).
* Each `_test.*` file must contain its own `main` function.
* Run with `oh test`.

## Library Auto-Detection

Orchideous auto-detects libraries from `#include` directives in your source files using `pkg-config`. Supported libraries include:

* **Graphics**: OpenGL, GLUT, GLFW, GLEW, GLM, Vulkan, SDL2, SFML (2 & 3), raylib
* **GUI**: GTK (2, 3 & 4), Qt6, VTE
* **Audio**: OpenAL, SDL2_mixer, PipeWire, rtaudio
* **Other**: Boost, libconfig++, FastCGI, ReactPhysics3D, libnotify, X11

For versioned libraries, the newest available version is preferred (e.g. GTK 4 over GTK 3, SFML 3 over SFML 2).

When a build fails due to a missing header, Orchideous will suggest which package to install (using `pkgfile` on Arch Linux or `apt-file` on Debian/Ubuntu).

## Examples

Over 40 examples are included in the `examples/` directory:

| Category | Examples |
|---|---|
| **Basics** | `hello`, `args`, `lambda`, `defer`, `invoke`, `visit`, `async`, `designated`, `entities`, `validorder`, `findfiles`, `platforms`, `config` |
| **Graphics** | `sfml`, `sfml_audio`, `bisqwit`, `sdl2`, `sdl2_opengl`, `gl4_spirv`, `gles2_glfw`, `gles3_glfw`, `gles3_sdl2`, `raylib`, `raylib5`, `vulkan`, `vulkan_glfw`, `x11`, `x11_opengl`, `smallpt` |
| **GUI** | `gtk4`, `gtk4ui`, `dunnetgtk`, `qt6` |
| **Audio** | `openal`, `synth`, `mixer`, `pipewire`, `rtaudio` |
| **Other** | `boost`, `boost_thread`, `notify`, `reactphysics`, `fastcgi`, `tinyhello`, `win64crate` |

Build all examples:

```sh
make examples
```

## Packaging

Install to a package directory:

```sh
DESTDIR="$pkgdir" PREFIX=/usr oh install
```

Or package into a local `pkg/` directory:

```sh
oh pkg
```

Generate standalone build files for users without `oh`:

```sh
oh export    # generates Makefile + build.sh + clean.sh
```

## Cross-Compilation

Build for 64-bit Windows (requires `x86_64-w64-mingw32-g++` or Docker):

```sh
oh win64
oh smallwin64
oh tinywin64
```

Test Windows executables with Wine:

```sh
oh run    # after oh win64, uses wine automatically
```

## Source Code Formatting

```sh
oh fmt    # formats source code using clang-format (Webkit style)
```

The formatting style is fixed and not configurable, on purpose.

## Requirements

* `g++` with C++20 support (or later)
* `pkg-config`
* `make` (for the project Makefile, not for building C++ projects)

### Optional

* `clang++` — build with `oh clang`
* `lldb` or `gdb` — for debugging
* `pkgfile` (Arch Linux) or `apt-file` (Debian/Ubuntu) — for missing-package suggestions
* `x86_64-w64-mingw32-g++` or `docker` — for Windows cross-compilation
* `wine` — for testing Windows executables
* `valgrind` — for profiling (`oh valgrind`)
* `clang-format` — for `oh fmt`
* `ninja` — for `oh ninja` / `oh cmake ninja`

### Arch Linux (all examples)

```sh
sudo pacman -S --needed base-devel boost fcgi freeglut glew glfw glibmm glm glu \
  gtk4 libconfig libpipewire libx11 openal qt6-base raylib reactphysics3d \
  rtaudio sdl2-compat sdl2_mixer sfml vte4 vulkan-headers vulkan-icd-loader
```

### Debian / Ubuntu (all examples)

```sh
sudo apt-get install -y build-essential pkg-config \
  libboost-all-dev libconfig++-dev libfcgi-dev libglew-dev libglfw3-dev \
  libglibmm-2.4-dev libglm-dev libglu1-mesa-dev libgtk-4-dev libopenal-dev \
  libpipewire-0.3-dev libsdl2-dev libsdl2-mixer-dev libsfml-dev \
  libvte-2.91-gtk4-dev libvulkan-dev libx11-dev freeglut3-dev qt6-base-dev
```

Note: `raylib` and `reactphysics3d` are not available in Ubuntu repositories. Ubuntu 24.04 ships SFML 2 and rtaudio 5, while the included examples use SFML 3 and rtaudio 6 APIs — those examples will be skipped on Ubuntu. Examples that depend on unavailable libraries are automatically skipped in CI.

## Platform Notes

### macOS

Install a recent GCC and dependencies with Homebrew:

```sh
brew install gcc pkg-config
```

### FreeBSD / NetBSD

Use `gmake` instead of `make`. Install dependencies:

```sh
# FreeBSD
pkg install pkgconf gmake

# NetBSD
pkgin install pkgconf gmake
```

### OpenBSD

Install g++ 11+ and build with `oh CXX=eg++`.

## Features and Limitations

* **No configuration files needed** — follows the directory structure conventions above.
* **Auto-detection** of compiler flags, includes and libraries via `pkg-config` and platform-specific package managers.
* **Incremental compilation** — only recompiles changed source files.
* **Profile-guided optimization** — `oh rec` collects profiling data, subsequent builds use it.
* Built-in support for testing, debugging, cross-compilation, and code generation.
* Meant for building **executables**, not libraries.
* Generated `CMakeLists.txt` is specific to the system it was generated on.

## General Info

* License: BSD-3
* Version: 1.0.2
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
