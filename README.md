![o](img/icon_128x128.png)

![Build](https://github.com/xyproto/o/workflows/Build/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/o)](https://goreportcard.com/report/github.com/xyproto/o) [![License](https://img.shields.io/badge/license-BSD-green.svg?style=flat)](https://raw.githubusercontent.com/xyproto/o/main/LICENSE) [![Stand With Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://stand-with-ukraine.pp.ua)

`o` is a text editor and a minimalistic IDE.

It might be a good fit for:

* Editing git commit messages (using `EDITOR=o git commit`).
* Editing `README.md` and `TODO.md` files.
* Write Markdown files and then export to PDF.
* Learning programming languages, like Rust or Zig.
* Editing files deep within larger Go or C++ projects.
* Solving Advent of Code tasks.
* Writing and maintaining to-do lists and project documentation in Markdown.
* Testing if your favorite package manager can handle single-letter package names.
* Being part of live ISO images, since it supports VT100, is small and self-contained, has a built-in man page viewer, can be used for viewing logs or ie. edit `/etc/fstab`.

For a more feature complete editor that is also written in Go, check out [micro](https://github.com/zyedidia/micro).

## Screenshots

Screenshot of the VTE GUI application that can be found in the `ko` directory, running the `o` editor:

![screenshot](img/screenshot2021sept.png)

Stepping through the assembly instructions of a Rust program by entering debug mode with the `ctrl-o` menu and then stepping with `ctrl-n`:

![debug rust](img/2022-05-11_debug_rust.png)

## Packaging status

[![Packaging status](https://repology.org/badge/vertical-allrepos/o-editor.svg)](https://repology.org/project/o-editor/versions) [![Packaging status](https://repology.org/badge/vertical-allrepos/o.svg)](https://repology.org/project/o/versions)

## Quick start

With Go 1.17 or later, `o` can be installed like this:

    go install github.com/xyproto/o/v2@latest

Alternatively, download and install a [release version](https://github.com/xyproto/o/releases). For example, for Raspberry Pi 2, 3 or 4 running Linux:

    curl -sL 'https://github.com/xyproto/o/releases/download/2.53.0/o-2.53.0-linux_armv7_static.tar.xz' | tar JxC /tmp && sudo install -Dm755 /tmp/o-2.53.0-linux_armv7_static/o /usr/bin/o && sudo install -Dm644 /tmp/o-2.53.0-linux_armv7_static/o.1.gz /usr/share/man/man1/o.1.gz

* Remember to use `tar zxC` if the release file for your platform ends with `.tar.gz`.
* The `sudo install` commands may be slightly different for FreeBSD and NetBSD.

## Setting `o` as the default editor for `git`

To set:

    git config --global core.editor o

To unset:

    git config --global --unset core.editor

## Viewing man pages

By setting the `MANPAGER` environment variable, it's possible to use `o` for viewing man pages:

    export MANPAGER=o

An alternative to viewing man pages in `o` is to use `less`:

    export MANPAGER='less -s -M +Gg'

## Unique features

These features are unique to `o`, as far as I am aware:

* If the loaded file is read-only, all text will be red by default.
* Smart cursor movement, trying to maintain the X position when moving up and down, across short and long lines.
* Press `ctrl-v` once to paste one line, press `ctrl-v` again to paste the rest.
* Press `ctrl-c` once to copy one line, press `ctrl-c` again to copy the rest (until a blank line).
* Open or close a portal with `ctrl-r`. When a portal is open, copy lines across files (or within the same file) with `ctrl-v`.
* Build code with `ctrl-space` and format code with `ctrl-w`, for a wide range of programming languages.
* Cycle git rebase keywords with `ctrl-r` or `ctrl-w`, when an interactive git rebase session is in progress.
* Jump to a line with `ctrl-l`. Either enter a number to jump to a line or just press `return` to jump to the top. Press `ctrl-l` and `return` again to jump to the bottom.
* If tab completion in the terminal went wrong and you are trying to open a `main.` file that does not exist, but `main.cpp` and `main.o` does exists, then `main.cpp` will be opened.
* For C-like languages, missing parentheses are added to statements like `if`, `for` and `while` when return is pressed.
* Search and replace by entering a search term (or unicode rune on the form `u+0000`), pressing `tab` entering the replacement string or rune, and then pressing `return`.
* When jumping to a specific line or percentage (ie. `50%`) of a file with `ctrl-l`, jumping to a fraction (ie. `0.5`) is also supported.

## Other features and limitations

* The syntax highlighting is instant.
* Can compile `"Hello, World"` in many popular programming languages simply by pressing `ctrl-space`.
* Configuration-free, for better and for worse.
* `ctrl-t` can jump between a C++ header and source file.
* Can only edit one file at the time, by design.
* Provides syntax highlighting for Go, C++, Markdown, Bash and several other languages. There is generic syntax highlighting built-in.
* Will jump to the last visited line when opening a recent file.
* Is provided as a single self-contained executable.
* Loads faster than both `vim` and `emacs`, for small files.
* Can render text to PDF either by itself or by using `pandoc`.
* Tested with `alacritty`, `st`, `urxvt`, `konsole`, `zutty` and `xfce4-terminal`.
* Tested on Arch Linux, Debian and FreeBSD.
* Never asks before saving or quitting. Be careful!
* The [`NO_COLOR`](https://no-color.org) environment variable can be set to disable all colors.
* Rainbow parentheses makes lines with many parentheses easier to read.
* Limited to VT100, so hotkeys like `ctrl-a` and `ctrl-e` must be used instead of `Home` and `End`.
* Compiles with either `go` or `gccgo`.
* Will strip trailing whitespace whenever it can.
* Must be given a filename at start.
* May provide smart indentation.
* Requires that `/dev/tty` is available.
* `xclip` (for X) or `wl-clipboard` (for Wayland) must be installed if the system clipboard should be used.
* May take a line number as the second argument, with an optional `+` or `:` prefix.
* If the filename is `COMMIT_EDITMSG`, the look and feel will be adjusted for git commit messages.
* Supports `UTF-8`, but some runes may be displayed incorrectly.
* Only UNIX-style line endings are supported (`\n`).
* Will convert DOS/Windows line endings (`\r\n`) to UNIX line endings (just `\n`), whenever possible.
* Will replace non-breaking space (`0xc2 0xa0`) with a regular space (`0x20`) whenever possible.
* If interactive rebase is launched with `git rebase -i`, then either `ctrl-w` or `ctrl-r` will cycle the keywords for the current line (`fixup`, `drop`, `edit` etc).
* If the editor executable is renamed to a word starting with `r` (or have a symlink with that name), the default theme will be red/black.
* If the editor executable is renamed to a word starting with `l` (or have a symlink with that name), the default theme will be suitable for light backgrounds.
* Want to quickly convert Markdown to PDF and have pandoc installed? Try `o filename.md`, press `ctrl-space` and quit with `ctrl-q`.
* Press `ctrl-w` to toggle the check mark in `- [ ] TODO item` boxes in Markdown.
* `o` is written mostly in `o`, with some use of NeoVim for the initial development.
* The default syntax highlighting theme aims to be as pretty as possible with less than 16 colors, but it mainly aims for clarity. It should be easy to spot a keyword, number, string or a stray parenthesis.

## Known bugs

* Using `tmux` and resizing the terminal emulator window may trigger text rendering issues. Try pressing `esc` to redraw the text.
* For some terminal emulators, if `o` is busy performing an operation, pressing `ctrl-s` may lock the terminal. Some terminal emulators, like `konsole`, can be configured to turn off this behavior. Press `ctrl-q` to unlock the terminal again (together with the unfortunate risk of quitting `o`). You can also use the `ctrl-o` menu for saving and quitting.
* Some unicode runes may disrupt the text flow. This is generally not a problem for editing code and configuration files, but may be an issue when editing files that contains text in many languages.
* `o` may have issues with large files (of several MB+). For normal text files or source code files, this is a non-issue.
* Using backspace near the end of lines that are longer than the terminal width may cause the cursor to jump.
* Using `ctrl-\` to comment or uncomment code at the very last line of a file may result in odd behavior.
* Middle-click pasting (instead of pasting with `ctrl-v`) may only paste the first character.
* The smart indentation is not always smart.

## Hotkeys

* `ctrl-s` - Save.
* `ctrl-q` - Quit.
* `ctrl-r` - Open or close a portal. Text can be pasted from the portal into another file with `ctrl-v`.
             For "git interactive rebase" mode (`git rebase -i`), this will cycle the rebase keywords.
* `ctrl-w` - Format the current file (see the table below).
* `ctrl-a` - Go to start of text, then start of line and then to the previous line.
* `ctrl-e` - Go to end of line and then to the next line.
* `ctrl-p` - Scroll up 10 lines, or go to the previous match if a search is active.
* `ctrl-n` - Scroll down 10 lines, or go to the next match if a search is active.
* `ctrl-k` - Delete characters to the end of the line, then delete the line.
* `ctrl-g` - Toggle a status line at the bottom for displaying: filename, line, column, Unicode number and word count.
* `ctrl-d` - Delete a single character.
* `ctrl-t` - For C and C++: jump between the current header and source file. For Agda, insert a symbol.
             For the rest, record and play back keypresses. Press escape to clear the current macro.
* `ctrl-o` - Open a command menu with actions that can be performed. The first menu item is always `Save and quit`.
* `ctrl-x` - Cut the current line. Press twice to cut a block of text (to the next blank line).
* `ctrl-c` - Copy one line. Press twice to copy a block of text.
* `ctrl-v` - Paste one trimmed line. Press twice to paste multiple untrimmed lines.
* `ctrl-space` - Build program, render to PDF or export to man page (see table below).
* `ctrl-j` - Join lines (or jump to the bookmark, if set).
* `ctrl-u` - Undo (`ctrl-z` is also possible, but may background the application).
* `ctrl-l` - Jump to a specific line number. Press `return` to jump to the top. If at the top, press `return` to jump to the bottom.
* `ctrl-f` - Search for a string. The search wraps around and is case sensitive. Press tab instead of return to search and replace.
* `ctrl-b` - Toggle a bookmark for the current line, or if set: jump to a bookmark on a different line.
* `ctrl-\` - Comment in or out a block of code.
* `ctrl-~` - Jump to a matching parenthesis.
* `esc` - Redraw everything and clear the last search.

## Build and format

* At the press of `ctrl-space`, `o` will try to build or export the current file.
* At the press of `ctrl-w`, `o` will try to format the current file, in an opinionated way. If the current file is empty, template text may be inserted.

| Programming language                            | File extensions                                           | Jump to error | Build command                                     | Format command ($filename is a temporary file)                                                                 |
|-------------------------------------------------|-----------------------------------------------------------|---------------|---------------------------------------------------|----------------------------------------------------------------------------------------------------------------|
| Ada                                             | `.ada`                                                    | WIP           | WIP                                               | WIP                                                                                                            |
| Agda                                            | `.agda`                                                   | yes           | `agda -c $filename`                               | N/A                                                                                                            |
| C and C++                                       | `.cpp`, `.cc`, `.cxx`, `.h`, `.hpp`, `.c++`, `.h++`, `.c` | yes           | `cxx`                                             | `clang-format -fallback-style=WebKit -style=file -i -- $filename`                                              |
| C#                                              | `.cs`                                                     | yes           | `csc -nologo -unsafe $filename`                   | `astyle -mode=cs main.cs`                                                                                      |
| Clojure                                         | `.clj`, `.cljs`, `.clojure`                               | WIP           | `lein uberjar`                                    | WIP                                                                                                            |
| Crystal                                         | `.cr`                                                     | yes           | `crystal build --no-color $filename`              | `crystal tool format $filename`                                                                                |
| D                                               | `.d`                                                      | yes           | `gdc`                                             | WIP                                                                                                            |
| Garnet                                          | `.gt`                                                     | WIP           | `garnetc -o $executable $filename`                | N/A                                                                                                            |
| Go                                              | `.go`                                                     | yes           | `go build`                                        | `goimports -w -- $filename`                                                                                    |
| Hare                                            | `.ha`                                                     | yes           | `hare build`                                      | N/A                                                                                                            |
| Haskell                                         | `.hs`                                                     | yes           | `ghc -dynamic $filename`                          | `brittany --write-mode=inplace $filename`                                                                      |
| Jakt                                            | `.jakt`                                                   | WIP           | WIP                                               | WIP                                                                                                            |
| Java                                            | `.java`                                                   | yes           | `javac` + `jar`, see details below                | `google-java-format -i $filename`                                                                              |
| JavaScript                                      | `.js`                                                     | WIP           | WIP                                               | `prettier --tab-width 4 -w $filename`                                                                          |
| Kotlin, if `kotlinc-native` is installed        | `.kt`                                                     | yes           | `kotlinc-native -nowarn -opt -Xallocator=mimalloc -produce program -linker-option '--as-needed' $filename` | `ktlint`                                              |
| Kotlin                                          | `.kt`                                                     | yes           | `kotlinc $filename -include-runtime -d`           | `ktlint`                                                                                                       |
| Lua                                             | `.lua`                                                    | yes           | `luac`                                            | `lua-format -i --no-keep-simple-function-one-line --column-limit=120 --indent-width=2 --no-use-tab $filename`  |
| Nim                                             | `.nim`                                                    | WIP           | `nim c`                                           | WIP                                                                                                            |
| Object Pascal                                   | `.pas`, `.pp`, `.lpr`                                     | yes           | `fpc`                                             | WIP                                                                                                            |
| OCaml                                           | `.ml`                                                     | WIP           | `ocamlopt -o $executable $filename`               | WIP                                                                                                            |
| Odin                                            | `.odin`                                                   | yes           | `odin build`                                      | N/A                                                                                                            |
| Python                                          | `.py`                                                     | yes           | `python -m py_compile $filename`                  | `autopep8 -i --maxline-length 120 $filename`                                                                   |
| Rust, if `Cargo.toml` or `../Cargo.toml` exists | `.rs`                                                     | yes           | `cargo build`                                     | `rustfmt $filename`                                                                                            |
| Rust                                            | `.rs`                                                     | yes           | `rustc $filename`                                 | `rustfmt $filename`                                                                                            |
| Scala                                           | `.scala`                                                  | yes           | `scalac` + `jar`, see details below               | WIP                                                                                                            |
| Standard ML                                     | `.sml`                                                    | yes           | `mlton`                                           | WIP                                                                                                            |
| TypeScript                                      | `.ts`                                                     | WIP           | WIP                                               | WIP                                                                                                            |
| V                                               | `.v`                                                      | yes           | `v build`                                         | `v fmt $filename`                                                                                              |
| Zig                                             | `.zig`                                                    | yes           | `zig build-exe -lc $filename`                     | `zig fmt $filename`                                                                                            |

`/etc/fstab` files are also supported, and can be formatted with `ctrl-w` if [`fstabfmt`](https://github.com/xyproto/fstabfmt) is installed.

| Markup language | File extensions | Jump to error | Format command ($filename is a temporary file) |
|----|----|----|----|
| HTML | `.htm`, `.html` | no | `tidy -w 120 -q -i -utf8 --show-errors 0 --show-warnings no --tidy-mark no --force-output yes -ashtml -omit no -xml no -m -c` |

* `o` will try to jump to the location where the error is and otherwise display `Success`.
* For regular text files, `ctrl-w` will word wrap the lines to a length of 99.
* If `kotlinc-native` is not available, this build command will be used instead: `kotlinc $filename -include-runtime -d $name.jar`

CXX can be downloaded here: [GitHub project page for CXX](https://github.com/xyproto/cxx).

| File type | File extensions  | Build or export command                                           |
|-----------|------------------|-------------------------------------------------------------------|
| AsciiDoc  | `.adoc`          | `asciidoctor -b manpage` (writes to `out.1`)                      |
| scdoc     | `.scd`, `.scdoc` | `scdoc` (writes to `out.1`)                                       |
| Markdown  | `.md`            | `pandoc -N --toc -V geometry:a4paper` (writes to `$filename.pdf`) |

## Debug support for C and C++

This is a brand new feature and needs more testing.

* If `gdb` is installed, it's possible to select "Debug mode" from the `ctrl-o` menu and then build and step through a program with `ctrl-space`, or set a breakpoint with `ctrl-b` and continue with `ctrl-space`.
* Messages printed to stdout are displayed as a status message when that line is reached.
* An indication of which line the program is at has not yet been added, and is a work in progress.
* There are status messages indicating when the debug session is started and ended.
* Therefore `o` is not only an editor, but also an integrated development environment (IDE).

## Manual installation

On Linux:

    git clone https://github.com/xyproto/o
    cd o
    make
    sudo install -Dm755 o /usr/bin/o
    gzip o.1
    sudo install -Dm644 o.1.gz /usr/share/man/man1/o.1.gz

## Dependencies

C++

* For building code with `ctrl-space`, [`cxx`](https://github.com/xyproto/cxx) must be installed.
* For formatting code with `ctrl-w`, `clang-format` must be installed.

Go

* For building code with `ctrl-space`, The `go` compiler must be installed.
* For formatting code with `ctrl-w`, [`goimports`](https://godoc.org/golang.org/x/tools/cmd/goimports) must be installed.

Zig

* For building and formatting Zig code, only the `zig` command is needed.

V

* For building and formatting V code, only the `v` command is needed.

Rust

* For building code with `ctrl-space`, `Cargo.toml` must exist and `cargo` must be installed.
* For formatting code with `ctrl-w`, `rustfmt` must be installed.

Haskell

* For building the current file with `ctrl-space`, the `ghc` compiler must be installed.
* For formatting code with `ctrl-w`, [`brittany`](https://github.com/lspitzner/brittany) must be installed.

Python

* `ctrl-space` only checks the syntax, without executing. This only requires `python` to be available.
* For formatting the code with `ctrl-w`, `autopep8` must be installed.

Crystal

* For building and formatting Crystal code, only the `crystal` command is needed.

Kotlin

* For building code with `ctrl-space`, `kotlinc` must be installed. A `.jar` file is created if the compilation succeeded.
* For formatting code with `ctrl-w`, `ktlint` must be installed.

Java

* For building code with `ctrl-space`, `javac` and `jar` must be installed. A `.jar` file is created if the compilation succeeded.
* For formatting code with `ctrl-w`, `google-java-format` must be installed.

Scala

* For building code with `ctrl-space`, `scalac` and `jar` must be installed. A `.jar` file is created if the compilation succeeded.
* The jar file can be executed with `java -jar main.jar`. Use `scalac -d main.jar MyFile.scala` if you want to produce a jar that can be executed with `scala main.jar`.
* For formatting code with `ctrl-w`, `scalafmt` must be installed.

D

* For building code with `ctrl-space`, `gdc` must be available.

JSON

* The JSON formatter is built-in. Note that for some files it may reorganize items in an undesirable order, so don't save the file if the result is unexpected.

fstab

* For formatting `fstab` files (usually `/etc/fstab`) with `ctrl-w`, [`fstabfmt`](https://github.com/xyproto/fstabfmt) must be installed.

JavaScript

* For formatting JavaScript code with , `prettier` must be installed.

## A note on Java and Scala

Since `kotlinc $filename -include-runtime -d` builds to a `.jar`, I though I should do the same for Java. The idea is to easily compile a single or a small collection of `.java` files, where one of the file has a `main` function.

If you know about an easier way to build a `.jar` file from `*.java`, without using something like gradle, please let me know by submitting a pull request. This is pretty verbose...

```sh
javaFiles=$(find . -type f -name '*.java')
for f in $javaFiles; do
  grep -q 'static void main' "$f" && mainJavaFile="$f"
done
className=$(grep -oP '(?<=class )[A-Z]+[a-z,A-Z,0-9]*' "$mainJavaFile" | head -1)
packageName=$(grep -oP '(?<=package )[a-z,A-Z,0-9,.]*' "$mainJavaFile" | head -1)
if [[ $packageName != "" ]]; then
  packageName="$packageName."
fi
mkdir -p _o_build/META-INF
javac -d _o_build $javaFiles
cd _o_build
echo "Main-Class: $packageName$className" > META-INF/MANIFEST.MF
classFiles=$(find . -type f -name '*.class')
jar cmf META-INF/MANIFEST.MF ../main.jar $classFiles
cd ..
rm -rf _o_build
```

For Scala, I'm using this code, to produce a `main.jar` file that can be run directly with `java -jar main.jar`:

```sh
#!/bin/sh
scalaFiles=$(find . -type f -name '*.scala')
for f in $scalaFiles; do
  grep -q 'def main' "$f" && mainScalaFile="$f"
  grep -q ' extends App ' "$f" && mainScalaFile="$f"
done
objectName=$(grep -oP '(?<=object )[A-Z]+[a-z,A-Z,0-9]*' "$mainScalaFile" | head -1);
packageName=$(grep -oP '(?<=package )[a-z,A-Z,0-9,.]*' "$mainScalaFile" | head -1);
if [[ $packageName != "" ]]; then
  packageName="$packageName."
fi
mkdir -p _o_build/META-INF
scalac -d _o_build $scalaFiles
cd _o_build
echo -e "Main-Class: $packageName$objectName\nClass-Path: /usr/share/scala/lib/scala-library.jar" > META-INF/MANIFEST.MF
classFiles=$(find . -type f -name '*.class')
jar cmf META-INF/MANIFEST.MF ../main.jar $classFiles
cd ..
rm -rf _o_build
```

If `/usr/share/scala/lib/scala-library.jar` is not found `scalac -d run_with_scala.jar` is used instead. This file can only be run with the `scala` command.

## Agda

`ctrl-t` brings up a menu with a selection of special symbols.

There are also these shortcuts:

  * Insert `⊤` by pressing `ctrl-t` and then `t`.
  * Insert `ℕ` by pressing `ctrl-t` and then `n`.


## Updating PKGBUILD files

When editing `PKGBUILD` files, it is possible to press `ctrl-o` and select `Call Guessica` to update the `pkgver=` and `source=` fields, by a combination of guesswork and online searching. This functionality depends on the [Guessica](https://github.com/xyproto/guessica) package update utility.

## List of optional runtime dependencies

* `agda` - for compiling Agda code
* `asciidoctor` - for writing man pages
* `astyle` - for formatting C# code
* `autopep8` - for formatting Python code
* `brittany` - for formatting Haskell code
* `cargo` - for compiling Rust
* `clang` - for formatting C++ code with `clang-format`
* `clojure` - for compiling Clojure
* `crystal` - for compiling Crystal
* [`cxx`](https://github.com/xyproto/cxx) - for compiling C++
* `fpc` - for compiling Object Pascal
* [`fstabfmt`](https://github.com/xyproto/fstabfmt) - for formatting `/etc/fstab`
* `g++` - for compiling C++ code
* `gdc` - for compiling D code
* `ghc` - for compiling Haskell code
* `go` - for compiling Go code
* `go-tools` - for formatting Go code and handling imports with `goimports`
* `google-java-format` - for formatting Java code
* `jad` - decompile `.class` files on the fly when opening them with `o`
* `java-environment` - for compiling Java code and creating `.jar` files with `javac` and `jar`
* `kotlin` - for compiling Kotlin
* `ktlint` - for formatting Kotlin code
* `lua` - for compiling Lua to bytecode
* `lua-format` - for formatting Lua code
* `mlton` - for compiling Standard ML
* `mono` - for compiling C# code
* `ocaml` - for compiling and formatting OCaml code
* `odin` - for compiling Odin
* `pandoc` - for exporting Markdown to PDF
* `prettier` - for formatting JavaScript, TypeScript and CSS
* `python` - for compiling Python to bytecode
* `rustc` - for compiling Rust
* `rustfmt` - for formatting Rust
* `scala` - for compiling Scala
* `sdoc` - for writing man pages
* `tidy` - for formatting HTML
* `v` - for compiling and formatting V code
* `zig` - for compiling and formatting Zig code

## Size

* The `o` executable is only **989k** when built with GCC 11.1.0 (for 64-bit Linux) and compressed with `upx`.
* This isn't as small as [e3](https://sites.google.com/site/e3editor/), an editor written in assembly (which is **234k**), but it's reasonably lean.

One way of building with `gccgo` and `upx`:

    go build -mod=vendor -gccgoflags '-Os -s' && upx o

It's **9.5M** when built with Go 1.17 and no particular build flags are given.

## Jumping to a specific line when opening a file

These four ways of opening `file.txt` at line `7` are supported:

* `o file.txt 7`
* `o file.txt +7`
* `o file.txt:7`
* `o file.txt+7`

This also means that filenames containing `+` or `:`, and then followed by a number, are not supported.

## Spinner

When loading files that are large or from a slow disk, an animated spinner will appear. The loading operation can be interrupted by pressing `esc`, `q` or `ctrl-q`.

![progress](img/progress.gif)

## Find and open

This shell function works in `zsh` and `bash` and may be useful for both searching for and opening a file at the given line number (works best if there is only one matching file, if not it will open several files in succession):

```bash
fo() { find . -type f -wholename "*$1" -exec /usr/bin/o {} $2 \;; }
```

If too many files are found, it is possible to stop opening them by selecting `Kill find` from the `ctrl-o` menu in `o`, which will execute `pkill find`.

Example use:

```sh
fo somefile.cpp 123
```

## Pandoc

When using `pandoc` to export from Markdown to PDF:

* If the `PAPERSIZE` environment variable is set to ie. `a4` or `letter`, it will be respected when exporting from Markdown to PDF using pandoc, at the press of `ctrl-space`.
* The `--pdf-engine=xelatex` and `--listings` flags are used, so `xelatex` and the `listings` package needs to be available. A standard installation of LaTeX and Pandoc should provide both.
* `Render to PDF with pandoc` will only appear on the `ctrl-o` menu when editing a Markdown file and `pandoc` is installed.

## Easter egg

Try the Konami code while in the `ctrl-o` menu to start a silly little game about feeding creatures with pellets before they are eaten.

## Recommended symlinks

* For starting `o` with the light theme: `ln -sf /usr/bin/o /usr/bin/lighted`.
* For starting `o` with the red & black theme: `ln -sf /usr/bin/o /usr/bin/redblack`.
* For starting `o` with the blue theme: `ln -sf /usr/bin/o /usr/bin/edit`.

## The GUI frontend `ko`

Build:

    make

Install:

    make gui-install

Installing a symlink for launching `ko` with a light theme:

    ln -sf /usr/bin/ko /usr/bin/lo

## Terminal settings

### Konsole

* Untick the `Flow control` option in the profile settings, to ensure that `ctrl-s` will never freeze the terminal.

## Stars

[![Stargazers over time](https://starchart.cc/xyproto/o.svg)](https://starchart.cc/xyproto/o)

## General info

* Version: 2.55.0
* License: 3-clause BSD
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
