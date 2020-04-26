# TODO

- [ ] Rainbow parenthesis.
- [ ] Backwards search + using `ctrl-p` to jump to previous location (or pop from a location stack).
- [ ] After ending a line with "\", indent two spaces relative to that line when pressing enter.
- [ ] Build PKGBUILD files with `ctrl-space`.
- [ ] Functionality for marking two locations, then later swap those lines and move both locations one line down.
- [ ] Handle long lines with a dedicated mode for editing a long line, perhaps by breaking it into one word per line in a
      separate Editor struct, then joining them back together when that Editor quits (`/tmp/_o_longline_splitted.txt`)?
- [ ] Make it easier to spot the cursor when scrolling or searching.
- [ ] Press `ctrl-g` for "go to definition". Toggle the status bar if pressed on a blank line.
- [ ] `ctrl-r` to rename symbols, when editing code.
- [ ] Syntax highlighting of `..`, `::`, `:asdfasdf:` and `^^^` in `.rst` files.
- [ ] Spellcheck all comments. Highlight misspelled words. Make it possible to add/ignore words.
- [ ] Introduce a key for jumping between the two locations where you've spent most time the last 10 minutes.
- [ ] Stop `ctrl-g` from flickering when holding down `up`, `down`, `ctrl-n` or `ctrl-p`.
- [ ] If `xclip` or `wl-clipboard` are not found when pasting, present a status message. Also check related env. vars.
