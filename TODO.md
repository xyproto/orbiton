# TODO

- [ ] Be able to edit `.txt.gz` and `.1.gz` files.
- [ ] Rainbow parenthesis.
- [ ] When in "SuggestMode", typing should start filtering the list.
- [ ] Backwards search result browsing using `ctrl-p` (alternatively keep a location stack when using `ctrl-n` and pop from that one).
- [ ] If a word is typed 1 letter differently from all the other instances in the current file: color it red.
- [ ] Enable search wraparound.
- [ ] Smart indent: After ending a line with `\`, indent two spaces relative to that line when pressing enter.
- [ ] Syntax highlighting of `..`, `::`, `:asdfasdf:` and `^^^` in `.rst` files.
- [ ] Build PKGBUILD files with `ctrl-space` instead of displaying the time.
- [ ] Spellcheck all comments. Highlight misspelled words. Make it possible to add/ignore words.
- [ ] `ctrl-f` and then `return` could jump to a location at least 10 lines away that has been most visited within the last 10 minutes.
- [ ] If `xclip` or `wl-clipboard` are not found when pasting, present a status message. Also check related env. vars.
- [ ] Smarter indentation for `}`. There are still a few cases where it's not too smart.
- [ ] Let the autocompletion also look at method definitions with matching variable names (ignoring types, for now).
- [ ] Find out which keys could be used for "go to definition" and "rename". Perhaps `ctrl-r` could be repurposed to offer a menu that could be browsed with `tab` the arrow keys.
- [ ] Plugins? When there's "txt2something" and "something2txt", o should be able to edit "something" files. This could be used for hex editing, editing ELF files etc.
- [ ] Fix the tab/space positioning issue in the editor UpEnd function.
