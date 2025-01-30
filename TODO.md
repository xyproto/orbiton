# TODO

## General

- [ ] Make it possible to search for double space ("  ").
- [ ] Support the Language Server Protocol, per language.
- [ ] When pressing esc several times to make the command menu appear (to aid ViM users),
      make the esc-pressing consistent. Either 3 or 4 times.
- [ ] Write a new syntax highlight module, the current one is a bit limited.
- [ ] At attempt 2 or 3 opening a locked file, just clear the lock and open it? This might not be a good idea.
- [ ] Make it possible to step through Odin programs in debug mode.
- [ ] Fix the syntax highlighting dependency to view strings with `-` as single words for CSS.
- [ ] Fix and rewrite the multiline string detection for Python and Starlark.
- [ ] Add a flag for only programming with arrow keys and space/return and esc, or joystick and A and B.
      Leverage Ollama to find good questions to ask and offer good options on screen.
      Use 2 to 4 large horizontal squares to choose between. Implement this is a new type of menu.
      Then package Orbiton as an app for Steam, Play Store and App Store, as some sort of programming game? Create a separate project for this.
- [ ] Have a progress indicator on the right side also for when `NO_COLOR=1` is set.
- [ ] Add a history for not only previous searches, but also for previous replacements.
- [ ] Fix the "jump to matching paren/bracket" feature so that it can jump anywhere in a file.
- [ ] Let `ctrl-space` show a preview of man pages instead of changing the syntax highlighting.
- [ ] When removing `-` in front of lines, do not move 1 to the right when encountering `}`.
- [ ] Let `ctrl-g` go to definition for more languages.
- [ ] Add a flag for using more colors, for nicer themes, perhaps `-2`.
- [ ] Support the `base16` themes.
- [ ] When pasting through a portal and reaching the end of the source, don't immediately start pasting from the clipboard. Require the cursor to be moved around first.
- [ ] When opening "main" and "main" is binary, while "main.c" is text, open "main.c" instead. Add a flag for not making these kinds of assumptions.
- [ ] Let `ctrl-space` when editing a man page toggle between viewing the code for the man page, and the rendered man page.
- [ ] New idea for a text editor: make it more like a multiplayer-game, where several people and AI agents can cooperate on the server side, with a nice client on top.
- [ ] If every other byte is 0x0 in a source code file, assume UTF-16 or Windows text formatting.
- [ ] When opening a file and pressing `ctrl-f` and then `return`: search for the previously searched for string.
- [ ] Let the status bar be toggled by the `ctrl-o` menu. Let `ctrl-g` when not on a definition do something useful, like cycle indenting a block 0 to 7 indentations.
- [ ] Make `echo asdf | o -c` work, for copying `asdf` to the clipboard.
- [ ] Run a specific test if the cursor is within a test function when double `ctrl-space` is pressed.
- [ ] When opening `file.txt+7`, only assume that 7 is the line number if no file named `file.txt+7` exists, but `file.txt` exists.
- [ ] `echo something | o -c` should be possible!
- [ ] Have many portal bookmarks. Add a menu option for selecting one of them, deleting all of them or deleting one of them.
- [ ] When pasting through a portal, show a little window with the filename and line number that is being pasted from. Drop the status message.
- [ ] When searching for text in Markdown or other text, use case-insensitive search. Use case-sensitive search in code.
- [ ] Drop the mutexes and have one "server" that deals with I/O and one "server" that deals with presentation.
- [ ] If running "o main" and "o main" + "o main.go" exists, open "main.go".
- [ ] Sorting lines does not handle indentation well. Examine why.
- [ ] Consider switching to [creack/pty](https://github.com/creack/pty).
- [ ] When pasting through a portal, make this even more apparent by changing the background color of lines being pasted in and also the background color of lines being pasted from, if in view.
- [ ] Instead of updating the entire screen when typing, keep track of the regions of the canvas that needs to be updated. Perhaps create version 2 of the vt100 Canvas.
- [ ] When the first word on a line in Kotlin is `const` followed by a space, expand it to `const val `, when it's being typed in.
- [ ] When calculating the progress, the algorithm assumes the cursor is at the top line of the canvas. If it's not, subtract some lines.
- [ ] For Go and "go to definition", let it be able to also discover packages in the parent directory.
- [ ] If a type is defined with `typealias`, then do not add an import to that type when formatting Kotlin code.
- [ ] When pressing `ctrl-space` twice, adjust the status message to indicate what is happening.
- [ ] Add a `Run` option to the ctrl-o menu that will only build first if needed.
- [ ] Draw a minimap with `silicon SOURCEFILE --theme gruvbox-dark --no-line-number --no-round-corner --no-window-controls --highlight-lines 10-20 --tab-width 4 --output IMAGEFILE` or create a custom minimap package.
- [ ] Do not highlight lines that start with `#` in gray, for Go. Or lines that starts with `//`, for shell scripts.
- [ ] Add a Markdown template with headers and checkboxes.
- [ ] Add support for `github.com/xyproto/ollamaclient` as an alternative to or instead of the OpenAI API.
      The `mistral` model is pretty fast and capable by now.
- [ ] Re-think the minimap feature.
- [ ] Let the `ctrl-o` menu have additional info, like time and date and GC stats.
- [ ] For man pages: if between "[-" and "]", do not color uppercase letters differently.
- [ ] For man pages: if the line contains "-*[a-z]" and then later "-*[a-z]" and a majority of words with "-", then color text red instead of blue (and consider the theme).
- [ ] Save a "custom words" and "ignored words" list to disk.
- [ ] If in man page mode, set the file as read-only and also let `q` quit.
- [ ] Let `ctrl-w` also format gzipped code, for instance when editing `main.cpp.gz`.
- [ ] Do not remove indentation from JS code in HTML when `ctrl-w` is pressed. See: https://github.com/yosssi/gohtml/issues/22
- [ ] When rebasing, look for the `>>>>` markers when opening the file and jump to the first one (and let `ctrl-n` search for the next one).
- [ ] When pasting with _double_ `ctrl-v`, let _one_ `ctrl-z` undo both keypresses.
- [ ] When pasting lines that start with `+` and it's not a diff/patch file, then replace `+` with a blank.
- [ ] When deleting lines with `ctrl-k` more than once, scroll the cursor line a bit up, to make it easier.
- [ ] If a file is passed through stdin and > 70% of the lines has a `:`, it might be a log file and not configuration.
- [ ] If a file is passed through stdin and has many similar lines and no comments or blank lines, it might be a log file and not configuration.
- [ ] HTTP client - scratch document style `.http` files.
- [ ] Add support for emojis. Perhaps by drawing lines differently if an emoji is detected.
- [ ] Parse some programming languages to improve the quality of the "go to defintion" feature.
- [ ] When the last line in a document is a long line ending with "}", make it possible to press return before the "}".
- [ ] Make it possible to export code to HTML or PNG, maybe by using Splash.
- [ ] Figure out why multi-line commenting sometimes stops after a few lines.
- [ ] Adjust the fuzzyness of the spell checker.
- [ ] Go through this file and remove all completed TODO items.

### Nano emulation mode

- [ ] If the file is huge, let ctrl-t time out instead of waiting for it to complete.
- [ ] When searching for a typo with ctrl-t, enable wrap-around for the search.
- [ ] Make the spell check dictionary persitent.
- [ ] Make it possible to set a marker with a hotkey before pressing ctrl-k.
- [ ] ctrl-\\ for replace (use ctrl-w, type in text to search for and then press Tab instead of Return).
- [ ] ctrl-q for searching backwards.
- [ ] alt-e for redo.
- [ ] alt-6 for copy (use ctrl-c instead).
- [ ] alt-w for next (use ctrl-n instead).
- [ ] alt-u for undo (use ctrl-z instead, if possible).
- [ ] alt-q for previous (use ctrl-p instead).
- [ ] alt-a to set mark.
- [ ] alt-] to jump to bracket.
- [ ] Support other themes, like the Mono Gray theme.

See also: https://staffwww.fullcoll.edu/sedwards/nano/nanokeyboardcommands.html

## Markdown

- [ ] Add a hotkey for inserting a TODO item.

## `o` to GUI frontend communication

- [ ] Use enet or another UDP protocol to communicate between the core editor and the GUI application. Or REST, just to make it even more accessible for developers?
- [ ] When changing themes from within the VTE/GKT3 frontend, let `o` be able to communicate a palette change per theme, using some sort of RPC.
- [ ] Use proper RPC between `o` and the VTE/GTK3 frontend. This also helps when upgrading to GTK4.
- [ ] Create an SDL2 frontend.

## Maybe

- [ ] Highlight changed lines if a file changed while monitoring it with `-m`.
- [ ] Move redrawing and clearing the statusbar to a separate goroutine.
- [ ] When searching for a number that does not exist in the document, jump there.
- [ ] `ctrl-g`, `up` could go to the previous function signature.
- [ ] `ctrl-g`, `down` could go to the next function signature.
- [ ] Re-implement `visudo` as a Go package and use that instead of `exec visudo` (if the executable is `osudo`).

## Autocompletion and AI generated code

- [ ] Primarily support Ollama instead of ChatGPT. Try one of the models with a large context. Try loading in all source files in a directory. Use my `ollamaclient` package.
- [ ] If ChatGPT is enabled, and there is just one error, and the fix proposed by ChatGPT is small, then apply the fix, but let the user press `ctrl-z` if they don't want it.
- [ ] If an API key is entered, save it to file in the cache directory.
- [ ] Add an environment variable for specifying the AI API endpoint.
- [ ] Embed https://github.com/nomic-ai/gpt4all + data files within the `o` executable, somehow.
- [ ] Add a way to generate git commit messages with ChatGPT
- [ ] Let the auto completion also look at method definitions with matching variable names (ignoring types, for now).
- [ ] Auto completion of filenames if the previous rune is `/` and tab is pressed.
- [ ] When generating code with ChatGPT, also send a list of function signatures and constants for the current file (+ header file).

## Building, debugging and testing programs

- [ ] Along with the per-file location, store the per-file last `ctrl-o` menu choice location. Or just move "Build" to the top, when on macOS.
- [ ] Jump to error for Erlang.
- [ ] Fix output parsing when running `go test` with `ctrl-space`.
- [ ] Jump to error when building with `ctrl-space` and `cargo`.
- [ ] When switching register pane layout with `ctrl-p`, save the contents of the old pane and use that.
- [ ] Make it possible to send custom commands to `gdb` with `ctrl-g` when in debug mode.
- [ ] Make it possible to step through Go programs as well.
- [ ] Build Jakt and Prolog programs with ctrl-space.
- [ ] Support for Prolog.
- [ ] Supporty for Red.
- [ ] Jump to error for Inko.

## Saving and loading

- [ ] When `somefile.go` and `somefile_test.go` exists, and only `somefile` is given, load `somefile.go`.
- [ ] When a filename is given, but it does not exist, and no extension is given, and the directory only contains one file, open that one.
- [ ] Show a spinner when reading a lot of data from stdin.
- [ ] When editing a man page, make it possible to toggle between the man page and the man page view mode.
- [ ] Auto-detect tabs/spaces when opening a file.
- [ ] When editing a file that then is deleted, `ctrl-s` should maybe create the file again?
      Or save it to `/tmp` or `~/.cache/o`? Or copy it to the clipboard?
- [ ] Be able to open and edit large text files (60M+).
- [ ] Auto-detect if a loaded file uses `\t` or 1, 2, 3, 4, or 8 spaces for indentation.
- [ ] Introduce a hexedit mode for binary files that will:
      * Not load the entire file into memory.
      * Display all bytes as a grid of "0xff" style fields, with the string representation to the right.
      * This might be better solved by having a separate hex editor?
- [ ] Be able to edit `.txt.gz` and `.1.gz` files.
- [ ] Plugins. When there's `txt2something` and `something2txt`, o should be able to edit "something" files in general.
      This could be used for hex editing, editing ELF files etc.
- [ ] When the editor executable is `list`, just list the contents and exit?

## Code navigation

- [ ] Make it possible to have groups of bookmarks per file, and then name them, somehow.
- [ ] When pressing `ctrl-g` or `F12` and there's a filename under the cursor that exists, go to that file.
- [ ] Let ctrl-p also jump to a matching parenthesis, if the last pressed key was an arrow key.

## Code editing

- [ ] When commenting out a block, move comment markers closer to the beginning of the text.
- [ ] When sorting comma-separated strings that do not start with (, [ or {, make sure to keep the same trailing comma status.
- [ ] When `}` is the last character of a file, sometimes pressing enter right before it does not work.
- [ ] If there are four lines: not comment, comment, not comment, comment, let ctrl+/ behave differently.
- [ ] Indentation in Rust is sometimes wonky.
- [ ] When changing a file from tabs to spaces, or the other way around, also modify indentations after comment markers.
- [ ] Tab in the middle of a line, especially on a `|` character, could insert spaces until the `|` aligns with the `|` above,
      if applicable (For Markdown tables).
- [ ] Smarter indentation for `}`. There are still a few cases where it's not too smart.
      Perhaps use the logic for tab-indenting for when dedenting `}`?
- [ ] If joining a line that starts with a single-line comment with a line below that also starts with a single line comment,
      remove the extra comment marker.
- [ ] When in `SuggestMode`, typing should start filtering the list.
- [ ] Introduce the concept of soft and hard breaks, to keep track of where lines were broken automatically and be able to reflow the text.
- [ ] Sort lines in a less opaque and unusual way than `left,up,right` `sort` `return` before documenting the feature.
- [ ] Let ctrl-k first delete until "{" and then util the end of the line if there is no "{"?

## Syntax highlighting

- [ ] Fix syntax highlighting of `(* ... *)` comments at the end of a line in OCaml.
- [ ] When viewing man pages, respect the current theme.
- [ ] Let `<<EOF` be considered the start of a multiline string in Shell, and `EOF` the end.
- [ ] Let a struct for a Theme contain both the light and the dark version, if there are two.
- [ ] Check that the right theme is loaded under `uxterm`.
- [ ] Also highlight hexadecimal numbers.
- [ ] Fix syntax highlighting of `'tokens` in Clojure.
- [ ] Don't highlight regular text in Nroff files.
- [ ] `-- ` comments in Ada should be recognized.
- [ ] // within a ` block should not be recognized
- [ ] Opening a read-only file in the Linux terminal should not display different red colors when moving to the bottom.
- [ ] If a word over N letters is typed 1 letter differently from all the other instances in the current file: color it differently!
- [ ] Rainbow parenthesis should be able to span multiple lines, especially for Clojure, Common Lisp, Scheme and Emacs Lisp.
- [ ] Hash strings (like sha256 hash sums), could be colored light yellow and dark yellow for every 2 characters
- [ ] Spellcheck all comments that are in English. Highlight misspelled words. Make it possible to add/ignore words.
- [ ] Ignore multiline comments within multiline comments.
- [ ] Also enable rainbow parenthesis for lines that ends with a single-line comment.
- [ ] Syntax highlighting of `..`, `::`, `:asdfasdf:` and `^^^` in `.rst` files.
- [ ] Highlight links in Markdown (perhaps color `[` and `]` yellow).

## Documentation

- [ ] Replace ` in o.1 with \b.
- [ ] Document that pressing the arrow keys in rapid succession and typing in `!sort` can sort a block of text with the external `sort` command.

## Cut, copy, paste and portals

- [ ] Support pbcopy/pbpaste (`echo asdf | pbcopy -Prefer txt` and `echo $(pbpaste -Prefer txt)`).
- [ ] Make it possible to double press `ctrl-c` again, to also copy the next block of text.
- [ ] Let `ctrl-t` take a line and move it through the portal?
- [ ] GUI: Look into the clipboard functions for VTE and if they can be used for mouse copy + paste.
- [ ] Re-enable cross-user portals?
- [ ] Paste with middle mouse button.
- [ ] When starting o, hash sum the clipboards it can find. When pasting, use the latest changed clipboard. If nothing changed, use the one for Wayland or X11, depending on environment variables.
- [ ] Use `wl-copy` for copy and cut. Use the same type of implementation as for `wl-paste`.
- [ ] Figure out why copy/paste is wonky on Wayland.
- [ ] Check if Wayland is in use (env var) before using `wl-*`.
- [ ] Add a command menu option to copy the entire file to the clipboard.
- [ ] Add a command menu option to copy the build command to the clipboard.
- [ ] If `xclip` or similar tool is not available, cut/copy/paste via a file.
- [ ] Pressing `ctrl-v` to paste does not work across X/Wayland sessions. It would be nice to find a more general clipboard solution.
- [ ] Let the cut/copy/paste line state be part of the editor state, because of undo.
- [ ] Cross user portals? Possibly by using `TMPDIR/oportal.dat`.

## Encoding

- [ ] Detect ISO-8859-1 and convert the file to UTF-8 before opening.
- [ ] Open text files with Chinese/Japanese/Korean characters without breaking the text flow.
- [ ] Quotestate Process can not recognize triple runes, like the previous previous rune is ", the previous rune is " and the current rune is ". The wrong arguments are passed to the function. Figure out why.

## Command menu

- [ ] Add a menu option for listing all functions in the current directory, alphabetically, and be able to jump to any one of them.
- [ ] Add one or more of these commands: regex search, go to definition, rename symbol, find references and disassembly.
- [ ] Make it easy to make recordings of the editing process (can already use asciinema?).

## Localization

- [ ] Localize all status messages and menu options.

## External programs

- [ ] Let rendering with `pandoc` have a spinner, since it can take a little while.
- [ ] Extract the functionality for searching a MessagePack file to a `mpgrep` utility, that has a `-B` flag (like `grep`).
- [ ] Draw inspiration from [kilo](https://github.com/antirez/kilo).

## Unit tests

- [ ] Add tests for the smart indent feature: for pressing return, tab and space, especially in relation with `{` and `}`.

## Refactoring

- [ ] Replace and/or rewrite the syntax highlighting system.
- [ ] Refactor the code to handle a line as a Line struct/object that has these markers: start of line, start of text, start of scroll view,
      end of scroll view, end of text, one after end of text, end of line including whitespace.
- [ ] Inherit from the Line struct (with interfaces+types+methods) by adding per-language markers: start of block, end of block,
      indentation compared to the line above, dedentation compared to the line above
- [ ] Or go for a server/client type of model, where the server deals with moving around in very large files, for instance.
- [ ] Create a Go package for detecting:
      * Language (specifically C++98 for instance)
      * Language family (C-like, ML-like etc)
      * tabs, spaces, indentations, mixed tabs/spaces
      * clang style, so that the same style may be used?
      * emacs tag
      * vim tag
      * what else?
      * then translate this to a struct
      * also think about how this can be skipped is the file is enormous and should be read in block-by-block
- [ ] Abstract the editor, so that sending in keypresses and examining the result can be tested with Go tests.
- [ ] Rewrite `insertRune`. Improve word-wrap related functionality.
- [ ] Introduce a type for screen coordinates, a type for screen coordinates + scroll offset, and another type for data coordinates.
- [ ] Create a Terminal type that implement the context.Context interface, then pass that to functions that
      would otherwise take both a `vt100.Canvas`, `vt100.TTY` and a `StatusBar`.
- [ ] Consider switching over to `github.com/creack/pty`, for better multi-platform support.

## Built-in game

- [ ] Two pellets next to each other should combine.
