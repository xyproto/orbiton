# TODO

- [ ] Keybinding from cutting the line at the bookmark and inserting it at the current location.
      Pressing this key repeatedly can be used for moving a block of code from one place to another.
- [ ] Backwards search + using `ctrl-p` to jump to previuous location.
- [ ] ctrl-l, 0 and then searching sometimes causes issues. Investigate.
- [ ] Don't completely clear and reset the terminal at exit, to allow for scrolling.
- [ ] After ending a line with "\", indent two spaces relative to that line when pressing enter.
- [ ] Rainbow paranthesis.
- [ ] Build PKGBUILD files with `ctrl-space`.
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
- [ ] Add a key to toggle "block mode" where cut/copy/paste applies to the current block of code, or just always use block mode for "cut" (and then for paste as well). The main objective is to be able to move blocks of code, while copy and paste can still work for single lines. Paste can paste a block if cut has recently been used. Something along those lines, for simplicity. The user can still copy and delete single lines with ctrl-c and ctrl-k.
