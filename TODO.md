# TODO

- [ ] When in "SuggestMode", typing should start filtering the list.
- [ ] Rainbow parenthesis.
- [ ] Backwards search + using `ctrl-p` to jump to previous location (or pop from a location stack).
- [ ] Search wraparound by going to line 0 and searching again.
- [ ] After ending a line with "\", indent two spaces relative to that line when pressing enter.
- [ ] Build PKGBUILD files with `ctrl-space` instead of displaying the time.
- [ ] Syntax highlighting of `..`, `::`, `:asdfasdf:` and `^^^` in `.rst` files.
- [ ] Spellcheck all comments. Highlight misspelled words. Make it possible to add/ignore words.
- [ ] Introduce a key for jumping between the two locations where you've spent most time the last 10 minutes.
- [ ] If `xclip` or `wl-clipboard` are not found when pasting, present a status message. Also check related env. vars.
- [ ] Smarter indentation for `}`. There are still a few cases where it's not too smart.
- [ ] If a word is typed 1 letter differently from all the other instances in the current file: color it red.
- [ ] Let the autocompletion also look at method definitions with matching variable names (ignoring types, for now).
- [ ] Find out which keys could be used to "go to definition" and "rename".
- [ ] Fix syntax highlighting for comments within strings, like `"/* hello */"`.
