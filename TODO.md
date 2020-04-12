# TODO

- [ ] Make it possible to save 16x16 16-grayscale favicon.ico files, using 0-F in a 16x16 grid, but only open favicon files with few enough colors.
- [ ] Keybinding for cutting the line at the bookmark and inserting it at the current location.
      Pressing this key repeatedly can be used for moving a block of code from one place to another.
- [ ] Backwards search + using `ctrl-p` to jump to previous location.
- [ ] After ending a line with "\", indent two spaces relative to that line when pressing enter.
- [ ] Rainbow parenthesis.
- [ ] Build PKGBUILD files with `ctrl-space`.
- [ ] Functionality for moving a block of code up or down. Perhaps a line-movement-mode that can also be used to reorder lines for `git rebase -i`.
- [ ] Handle long lines, but try to avoid horizontal scrolling. Perhaps open long lines in a new instance of the editor, but split at a custom rune, then join the line at exit.
- [ ] Make it easier to spot the cursor when scrolling or searching.
- [ ] Make the "smart dedentation" even smarter - let it consider the whitespace of the line above before dedenting.
- [ ] Go to definition, rename symbol, find references and suggestions while typing. Wonder which hotkey should be used for go to definition, though.
      Perhaps pressing `ctrl-g` three times.
- [ ] When entering a closing bracket, the smart indentation deindents one level too many. Fix this.
- [ ] Syntax highlighting of checkboxes in Markdown.
- [ ] Block mode, where operations work on the current block of text?
- [ ] Syntax highlighting of `..`, `::`, `:asdfasdf:` and `^^^` in .rst
