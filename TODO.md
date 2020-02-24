# TODO

- [ ] ctrl-l, 0 and then searching sometimes causes issues. Investigate.
- [ ] Fix word wrapping when elongating a line from somewhere in the middle.
- [ ] Don't completely clear and reset the terminal at exit, to allow for scrolling.
- [ ] After ending a line with "\", indent two spaces relative to that line when pressing enter.
- [ ] Don't highlight the word "package" in red for PKGBUILD files.
- [ ] Build PKGBUILD files with `ctrl-space`.
- [ ] Rainbow paranthesis.
- [ ] Functionality for moving a block of code up or down. Perhaps a line-movement-mode that can also be used to reorder lines for `git rebase -i`.
- [ ] Handle long lines, but try to avoid horizontal scrolling. Perhaps open long lines in a new instance of the editor, but split at a custom rune, then join the line at exit.
- [ ] Make it easier to spot the cursor when scrolling or searching.
- [ ] Keep indentation when pasting a single line.
- [ ] Make the "smart dedentation" even smarter - let it consider the whitespace of the line above before dedenting.
- [ ] Go to definition, rename symbol, find references and suggestions while typing. Wonder which hotkey should be used for go to definition, though.
      Perhaps pressing `ctrl-g` three times.
- [ ] Functionality for commenting out a block of code.
- [ ] When entering a closing bracket, the smart indentation deindents one level too many. Fix this.
- [ ] Syntax highlighting of checkboxes in Markdown.
- [ ] Syntax highlighting of `..`, `::`, `:asdfasdf:` and `^^^` in .rst
