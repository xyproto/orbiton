# TODO

## General

- [x] Also enable text selection with shift-arrows for .txt files.
- [x] Also handle ctrl-left and ctrl-right under iTerm2.
- [ ] Add a poetry mode for writing, reading, rating and browsing poetry.
- [x] Add a book mode for writing books under Kitty or iTerm2, with the editor to the left and a preview to the right, nice text flow and a typewriter-like user experience.
      OR let the book mode be fullscreen and only use text rendering, but be a "zoomed in" view of a document, with font rendering.
      (Basic book mode added with `--book`/`-B`: word wrap at 72, no syntax highlighting, status bar shown.)
- [ ] There is some duplication between this source code and the source code of the `syntax` package. Refactor and fix.
- [x] When globbing, prioritize `.sh` over `.bat`, unless on Windows.
- [ ] Let the default theme have a 256-color variant that is like the default theme, only nicer.
- [ ] If Orbiton is launched by a symlink starting with `p` (for `preview`), then act as an auto-reloading Markdown preview program.
- [ ] `o -b` should also build Go projects, Rust projects etc. (similar to `ctrl-space`), not only C and C++.
- [ ] If the user saves a file that ends with `.sh`, and there is no shebang, but the first word of the first line is an executable in the PATH, then `chmod +x` the file when saving.
- [ ] Highlight `<tags>` differently in `JSX` files.
- [ ] When piping command line output to `o -m` or `o -m -`, Orbiton should behave as a log colorizer.
- [ ] If pressing `ctrl-e` when block editing, then move all the cursors to the end of the lines.
- [ ] Pressing `ctrl-space` by accident when editing a `PKGBUILD` file is annyoing. Remove or require 2x `ctrl-space`.
- [ ] Differentiate between `bash`, `zsh`, `tsch`, `fish` and `sh` shell scripts in the `mode` package (+ more?).
- [ ] Also support syntax highlighting for `.tape` files (Charmbracelet VHS).
- [ ] Fix the syntax highlighting dependency to view strings with `-` as single words for CSS.
- [ ] Do not remove indentation from JS code in HTML when `ctrl-w` is pressed. See: https://github.com/yosssi/gohtml/issues/22
- [ ] Fix and rewrite the multiline string detection for Python and Starlark.
- [ ] Fix compilation of a single Scala file on macOS.
- [ ] Fix pasting with midclick, also for multiline strings.
- [ ] Fix the "jump to matching paren/bracket" feature so that it can jump anywhere in a file.
- [ ] Searching for a rune like "U+2713" does not work.
- [ ] Add a `Run` option to the ctrl-o menu that will only build first if needed.
- [ ] Add syntax highlighting of "<" and ">" in C++.
- [ ] Have many portal bookmarks. Add a menu option for selecting one of them, deleting all of them or deleting one of them.
- [ ] Also support running tetris.el?
- [ ] Make it possible to disable the use of v2/ollama.go at compile time using build tags, for a slightly smaller executable.
- [ ] When goimport is downloaded, add some info for the user, like the spinner.
- [ ] For Go and "go to definition", let it be able to also discover packages in the parent directory.
- [ ] For man pages: if between "[-" and "]", do not color uppercase letters differently.
- [ ] Improve syntax highlighting of comments in JSON, ref test/tsconfig.json.
- [ ] Let `ctrl-g` go to definition for more languages.
- [ ] Let `ctrl-space` show a preview of man pages instead of changing the syntax highlighting.
- [ ] Port the scons/python/cxx tool over as a Go module and use that by default when building C or C++.
- [ ] Run a specific test if the cursor is within a test function when double `ctrl-space` is pressed.
- [ ] When pasting through a portal and reaching the end of the source, don't immediately start pasting from the clipboard. Require the cursor to be moved around first.
- [ ] When pasting through a portal, make this even more apparent by changing the background color of lines being pasted in and also the background color of lines being pasted from, if in view.
- [ ] When pasting through a portal, show a little window with the filename and line number that is being pasted from. Drop the status message.
- [ ] Add a history for not only previous searches, but also for previous replacements.
- [ ] Add a megafile first-time tutorial.
- [ ] Add support for emojis. Perhaps by drawing lines differently if an emoji is detected.
- [ ] If a type is defined with `typealias`, then do not add an import to that type when formatting Kotlin code.
- [ ] If the `ctrl-o` menu was opened by pressing `esc` repeatedly, add a `Help` menu option.
- [ ] Let the `ctrl-o` menu have additional info, like time and date and GC stats.
- [ ] Make it possible to export code to HTML or PNG, maybe by using Splash.
- [ ] Make it possible to search for a string (like with rg or ag) and then go to the next file+match with a hotkey or menu option.
- [ ] Make it possible to search for double space ("  ").
- [ ] Remove chorded keys (ctrl-l,? etc). Instead, display a menu when ctrl-l is pressed twice or something. Or at least add a visual indicator for when the first part of a chorded key is pressed.
- [ ] For man pages: if the line contains "-*[a-z]" and then later "-*[a-z]" and a majority of words with "-", then color text red instead of blue (and consider the theme).
- [ ] Adjust the fuzzyness of the spell checker.
- [ ] At attempt 2 or 3 opening a locked file, just clear the lock and open it? This might not be a good idea.
- [ ] Continue working on the ollama worker queue branch.
- [ ] ctrl-/ should also be able to toggle /* * */ comments, not only single line comments.
- [ ] Do not highlight lines that start with `#` in gray, for Go. Or lines that starts with `//`, for shell scripts.
- [ ] Drop the mutexes and have one "server" that deals with I/O and one "server" that deals with presentation.
- [ ] Figure out why multi-line commenting sometimes stops after a few lines.
- [ ] If `ctrl-c` is pressed thrice when not in a function: copy to the end of the file.
- [ ] If `ctrl-g` is pressed on a comment or multiline comment, toggle the status bar.
- [ ] If a file is passed through stdin and > 70% of the lines has a `:`, it might be a log file and not configuration.
- [ ] If a file is passed through stdin and has many similar lines and no comments or blank lines, it might be a log file and not configuration.
- [ ] If a line of code in C or C++ have two arrows, like `directory = S_ISDIR(p->fts_statp->st_mode);`, then color the second arrow differently from the first one.
- [ ] If every other byte is 0x0 in a source code file, assume UTF-16 or Windows text formatting.
- [ ] If parenthesis are unbalanced (too many `)`), then it's not a function name. Like `reinterpret_cast<char*>(&dq)) != 0= {`.
- [ ] If the cursor is at the end of the line on the final line of the screen view, and arrow right is pressed, move down to the next line.
- [ ] In LISP, all strings starting with " and ending with " can be multiline strings?
- [ ] Instead of updating the entire screen when typing, keep track of the regions of the canvas that needs to be updated. Perhaps create version 2 of the vt100 Canvas.
- [ ] Let `ctrl-w` also format gzipped code, for instance when editing `main.cpp.gz`.
- [ ] Let `delete` ask if a file should be deleted, when browsing files.
- [ ] Let `game` or the konami code from the file browser also launch the easter egg.
- [ ] Let altgr+space not insert c:160.
- [ ] Let the status bar be toggled by the `ctrl-o` menu. Let `ctrl-g` when not on a definition do something useful, like cycle indenting a block 0 to 7 indentations.
- [ ] Parse some programming languages to improve the quality of the "go to definition" feature.
- [ ] Pressing ctrl-g in a script should toggle the status bar unless the cursor is on a function call and the function was found.
- [ ] Re-think the minimap feature.
- [ ] Save a "custom words" and "ignored words" list to disk.
- [ ] Sorting lines does not handle indentation well. Examine why.
- [ ] When browsing a directory that contains hidden files, show: `show hidden files with ctrl-h`.
- [ ] When calculating the progress, the algorithm assumes the cursor is at the top line of the canvas. If it's not, subtract some lines.
- [ ] When deleting lines with `ctrl-k` more than once, scroll the cursor line a bit up, to make it easier.
- [ ] When editing files in connection with browsing files, let `ctrl-n` and `ctrl-p` preserve the cursor position across files.
- [ ] When jumping to a parenthesis with ctrl-p, remember to scroll horizontally if needed.
- [ ] When opening `file.txt+7`, only assume that 7 is the line number if no file named `file.txt+7` exists, but `file.txt` exists.
- [ ] When opening a file and pressing `ctrl-f` and then `return`: search for the previously searched for string.
- [ ] When pasting lines that start with `+` and it's not a diff/patch file, then replace `+` with a blank.
- [ ] When pasting with _double_ `ctrl-v`, let _one_ `ctrl-z` undo both keypresses.
- [ ] When pressing `ctrl-space` twice, adjust the status message to indicate what is happening.
- [ ] When pressing ctrl-c twice while on a function signature, copy the entire function.
- [ ] When pressing ctrl-f and then Tab without a search string, enter regexp search mode.
- [ ] When pressing ctrl-x twice while on a function signature, cut the entire function.
- [ ] When pressing esc several times to make the command menu appear (to aid ViM users), make the esc-pressing consistent. Either 3 or 4 times.
- [ ] When rebasing, look for the `>>>>` markers when opening the file and jump to the first one (and let `ctrl-n` search for the next one).
- [ ] When removing `-` in front of lines, do not move 1 to the right when encountering `}`.
- [ ] When searching for text in Markdown or other text, use case-insensitive search. Use case-sensitive search in code.
- [ ] When the first word on a line in Kotlin is `const` followed by a space, expand it to `const val `, when it's being typed in.
- [ ] Write a new syntax highlight module, the current one is a bit limited.
- [ ] Add a flag for using more colors, for nicer themes, perhaps `-2`.
- [ ] Add a Markdown template with headers and checkboxes.
- [ ] Support the `base16` themes.
- [ ] When the last line in a document is a long line ending with "}", make it possible to press return before the "}".
- [ ] Draw a minimap with `silicon SOURCEFILE --theme gruvbox-dark --no-line-number --no-round-corner --no-window-controls --highlight-lines 10-20 --tab-width 4 --output IMAGEFILE` or create a custom minimap package.
- [ ] HTTP client - scratch document style `.http` files.
- [ ] When inserting a .gitignore template, also ignore files with no extension with the same name as the current directory, and also the go.mod name (last part, after /)
- [ ] Add a flag for only programming with arrow keys and space/return and esc, or joystick and A and B. Leverage Ollama to find good questions to ask and offer good options on screen. Use 2 to 4 large horizontal squares to choose between. Implement this is a new type of menu. Then package Orbiton as an app for Steam, Play Store and App Store, as some sort of programming game? Create a separate project for this.
- [ ] New idea for a text editor: make it more like a multiplayer-game, where several people and AI agents can cooperate on the server side, with a nice client on top.

### Nano emulation mode

- [ ] When searching for a typo with ctrl-t, enable wrap-around for the search.
- [ ] Make it possible to set a marker with a hotkey before pressing ctrl-k.
- [ ] alt-6 for copy (use ctrl-c instead).
- [ ] alt-] to jump to bracket.
- [ ] alt-a to set mark.
- [ ] alt-e for redo.
- [ ] alt-q for previous (use ctrl-p instead).
- [ ] alt-u for undo (use ctrl-z instead, if possible).
- [ ] alt-w for next (use ctrl-n instead).
- [ ] ctrl-\\ for replace (use ctrl-w, type in text to search for and then press Tab instead of Return).
- [ ] ctrl-q for searching backwards.
- [ ] If the file is huge, let ctrl-t time out instead of waiting for it to complete.
- [ ] Make the spell check dictionary persistent.
- [ ] Support other themes, like the Mono Gray theme.

See also: https://staffwww.fullcoll.edu/sedwards/nano/nanokeyboardcommands.html

## Markdown

- [ ] Add a hotkey for inserting a TODO item.

## `o` to GUI frontend communication

- [ ] Create an SDL2 frontend.
- [ ] Use enet or another UDP protocol to communicate between the core editor and the GUI application. Or REST, just to make it even more accessible for developers?
- [ ] Use proper RPC between `o` and the VTE/GTK3 frontend. This also helps when upgrading to GTK4.
- [ ] When changing themes from within the VTE/GTK3 frontend, let `o` be able to communicate a palette change per theme, using some sort of RPC.

## Maybe

- [ ] Re-implement `visudo` as a Go package and use that instead of `exec visudo` (if the executable is `osudo`).
- [ ] `ctrl-g`, `down` could go to the next function signature.
- [ ] `ctrl-g`, `up` could go to the previous function signature.
- [ ] Highlight changed lines if a file changed while monitoring it with `-m`.
- [ ] Move redrawing and clearing the statusbar to a separate goroutine.
- [ ] When searching for a number that does not exist in the document, jump there.

## Autocompletion and AI generated code

- [ ] If ChatGPT is enabled, and there is just one error, and the fix proposed by ChatGPT is small, then apply the fix, but let the user press `ctrl-z` if they don't want it.
- [ ] Add a way to generate git commit messages with ChatGPT
- [ ] Add an environment variable for specifying the AI API endpoint.
- [ ] Primarily support Ollama instead of ChatGPT. Try one of the models with a large context. Try loading in all source files in a directory. Use my `ollamaclient` package.
- [ ] Auto completion of filenames if the previous rune is `/` and tab is pressed.
- [ ] Embed https://github.com/nomic-ai/gpt4all + data files within the `o` executable, somehow.
- [ ] If an API key is entered, save it to file in the cache directory.
- [ ] Let the auto completion also look at method definitions with matching variable names (ignoring types, for now).
- [ ] When generating code with ChatGPT, also send a list of function signatures and constants for the current file (+ header file).

## Building, debugging and testing programs

- [ ] Make it possible to send custom commands to `gdb` with `ctrl-g` when in debug mode.
- [ ] Fix output parsing when running `go test` with `ctrl-space`.
- [ ] Jump to error when building with `ctrl-space` and `cargo`.
- [ ] Jump to error for Erlang.
- [ ] Jump to error for Inko.
- [ ] Along with the per-file location, store the per-file last `ctrl-o` menu choice location. Or just move "Build" to the top, when on macOS.
- [ ] Build Jakt and Prolog programs with ctrl-space.
- [ ] Make it possible to step through Go programs as well.
- [ ] Support for Prolog.
- [ ] Support for Red.
- [ ] When switching register pane layout with `ctrl-p`, save the contents of the old pane and use that.

## Saving and loading

- [ ] Show a spinner when reading a lot of data from stdin.
- [ ] When editing a file that then is deleted, `ctrl-s` should maybe create the file again? Or save it to `/tmp` or `~/.cache/o`? Or copy it to the clipboard?
- [ ] Auto-detect if a loaded file uses `\t` or 1, 2, 3, 4, or 8 spaces for indentation.
- [ ] Auto-detect tabs/spaces when opening a file.
- [ ] Be able to open and edit large text files (60M+).
- [ ] Introduce a hexedit mode for binary files that will: * Not load the entire file into memory. * Display all bytes as a grid of "0xff" style fields, with the string representation to the right. * This might be better solved by having a separate hex editor?
- [ ] Plugins. When there's `txt2something` and `something2txt`, o should be able to edit "something" files in general. This could be used for hex editing, editing ELF files etc.
- [ ] When a filename is given, but it does not exist, and no extension is given, and the directory only contains one file, open that one.
- [ ] When the editor executable is `list`, just list the contents and exit?

## Code navigation

- [ ] Make it possible to have groups of bookmarks per file, and then name them, somehow.
- [ ] Let ctrl-p also jump to a matching parenthesis, if the last pressed key was an arrow key.
- [ ] When pressing `ctrl-g` or `F12` and there's a filename under the cursor that exists, go to that file.

## Code editing

- [ ] Indentation in Rust is sometimes wonky.
- [ ] Introduce the concept of soft and hard breaks, to keep track of where lines were broken automatically and be able to reflow the text.
- [ ] When `}` is the last character of a file, sometimes pressing enter right before it does not work.
- [ ] If joining a line that starts with a single-line comment with a line below that also starts with a single line comment, remove the extra comment marker.
- [ ] If there are four lines: not comment, comment, not comment, comment, let ctrl+/ behave differently.
- [ ] Let ctrl-k first delete until "{" and then until the end of the line if there is no "{"?
- [ ] Smarter indentation for `}`. There are still a few cases where it's not too smart. Perhaps use the logic for tab-indenting for when dedenting `}`?
- [ ] Tab in the middle of a line, especially on a `|` character, could insert spaces until the `|` aligns with the `|` above, if applicable (For Markdown tables).
- [ ] When changing a file from tabs to spaces, or the other way around, also modify indentations after comment markers.
- [ ] When commenting out a block, move comment markers closer to the beginning of the text.
- [ ] When in `SuggestMode`, typing should start filtering the list.
- [ ] When sorting comma-separated strings that do not start with (, [ or {, make sure to keep the same trailing comma status.
- [ ] Sort lines in a less opaque and unusual way than `left,up,right` `sort` `return` before documenting the feature.

## Syntax highlighting

- [ ] Fix syntax highlighting of `'tokens` in Clojure.
- [ ] Fix syntax highlighting of `(* ... *)` comments at the end of a line in OCaml.
- [ ] Opening a read-only file in the Linux terminal should not display different red colors when moving to the bottom.
- [ ] Syntax highlighting of `..`, `::`, `:asdfasdf:` and `^^^` in `.rst` files.
- [ ] Also enable rainbow parenthesis for lines that ends with a single-line comment.
- [ ] Spellcheck all comments that are in English. Highlight misspelled words. Make it possible to add/ignore words.
- [ ] When viewing man pages, respect the current theme.
- [ ] // within a ` block should not be recognized
- [ ] `-- ` comments in Ada should be recognized.
- [ ] Also highlight hexadecimal numbers.
- [ ] Don't highlight regular text in Nroff files.
- [ ] Hash strings (like sha256 hash sums), could be colored light yellow and dark yellow for every 2 characters
- [ ] Highlight links in Markdown (perhaps color `[` and `]` yellow).
- [ ] If a word over N letters is typed 1 letter differently from all the other instances in the current file: color it differently!
- [ ] Ignore multiline comments within multiline comments.
- [ ] Let `<<EOF` be considered the start of a multiline string in Shell, and `EOF` the end.
- [ ] Rainbow parenthesis should be able to span multiple lines, especially for Clojure, Common Lisp, Scheme and Emacs Lisp.
- [ ] Check that the right theme is loaded under `uxterm`.
- [ ] Let a struct for a Theme contain both the light and the dark version, if there are two.

## Documentation

- [ ] Replace ` in o.1 with \b.
- [ ] Document that pressing the arrow keys in rapid succession and typing in `!sort` can sort a block of text with the external `sort` command.

## Cut, copy, paste and portals

- [ ] Pressing `ctrl-v` to paste does not work across X/Wayland sessions. It would be nice to find a more general clipboard solution.
- [ ] Figure out why copy/paste is wonky on Wayland.
- [ ] When starting o, hash sum the clipboards it can find. When pasting, use the latest changed clipboard. If nothing changed, use the one for Wayland or X11, depending on environment variables.
- [ ] Use `wl-copy` for copy and cut. Use the same type of implementation as for `wl-paste`.
- [ ] If `xclip` or similar tool is not available, cut/copy/paste via a file.
- [ ] Add a command menu option to copy the build command to the clipboard.
- [ ] Add a command menu option to copy the entire file to the clipboard.
- [ ] Re-enable cross-user portals?
- [ ] Cross user portals? Possibly by using `TMPDIR/oportal.dat`.
- [ ] GUI: Look into the clipboard functions for VTE and if they can be used for mouse copy + paste.
- [ ] Let `ctrl-t` take a line and move it through the portal?
- [ ] Make it possible to double press `ctrl-c` again, to also copy the next block of text.
- [ ] Let the cut/copy/paste line state be part of the editor state, because of undo.

## Encoding

- [ ] Quotestate Process can not recognize triple runes, like the previous previous rune is ", the previous rune is " and the current rune is ". The wrong arguments are passed to the function. Figure out why.
- [ ] Detect ISO-8859-1 and convert the file to UTF-8 before opening.
- [ ] Open text files with Chinese/Japanese/Korean characters without breaking the text flow.

## Command menu

- [ ] Add one or more of these commands: regex search, go to definition, rename symbol, find references and disassembly.
- [ ] Add a menu option for listing all functions in the current directory, alphabetically, and be able to jump to any one of them.
- [ ] Make it easy to make recordings of the editing process (can already use asciinema?).

## Localization

- [ ] Localize all status messages and menu options.

## External programs

- [ ] Let rendering with `pandoc` have a spinner, since it can take a little while.
- [ ] Draw inspiration from [kilo](https://github.com/antirez/kilo).
- [ ] Extract the functionality for searching a MessagePack file to a `mpgrep` utility, that has a `-B` flag (like `grep`).

## Unit tests

- [ ] Add tests for the smart indent feature: for pressing return, tab and space, especially in relation with `{` and `}`.

## Refactoring

- [ ] Create a Terminal type that implement the context.Context interface, then pass that to functions that would otherwise take both a `vt100.Canvas`, `vt100.TTY` and a `StatusBar`.
- [ ] Abstract the editor, so that sending in keypresses and examining the result can be tested with Go tests.
- [ ] Consider switching over to `github.com/creack/pty`, for better multi-platform support.
- [ ] Inherit from the Line struct (with interfaces+types+methods) by adding per-language markers: start of block, end of block, indentation compared to the line above, dedentation compared to the line above
- [ ] Create a Go package for detecting: * Language (specifically C++98 for instance) * Language family (C-like, ML-like etc) * tabs, spaces, indentations, mixed tabs/spaces * clang style, so that the same style may be used? * emacs tag * vim tag * what else? * then translate this to a struct * also think about how this can be skipped is the file is enormous and should be read in block-by-block
- [ ] Introduce a type for screen coordinates, a type for screen coordinates + scroll offset, and another type for data coordinates.
- [ ] Or go for a server/client type of model, where the server deals with moving around in very large files, for instance.
- [ ] Refactor the code to handle a line as a Line struct/object that has these markers: start of line, start of text, start of scroll view, end of scroll view, end of text, one after end of text, end of line including whitespace.
- [ ] Rewrite `insertRune`. Improve word-wrap related functionality.

## Built-in game

- [ ] Two pellets next to each other should combine.
