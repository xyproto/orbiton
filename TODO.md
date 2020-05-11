# TODO

- [ ] If a word is typed 1 letter differently from all the other instances in the current file: color it red.
- [ ] Rainbow parenthesis.
- [ ] Highlight links in Markdown (perhaps color `[` and `]` yellow).
- [ ] When in "SuggestMode", typing should start filtering the list.
- [ ] Backwards search result browsing using `ctrl-p` (alternatively keep a location stack when using `ctrl-n` and pop from that one).
- [ ] More predictable search wraparound.
- [ ] Be able to edit `.txt.gz` and `.1.gz` files.
- [ ] Syntax highlighting of `..`, `::`, `:asdfasdf:` and `^^^` in `.rst` files.
- [ ] Spellcheck all comments. Highlight misspelled words. Make it possible to add/ignore words.
- [ ] `ctrl-f` and then `return` could jump to a location at least 10 lines away that has been most visited within the last 10 minutes.
- [ ] If `xclip` or `wl-clipboard` are not found when pasting, present a status message. Also check related env. vars.
- [ ] Smarter indentation for `}`. There are still a few cases where it's not too smart.
      Perhaps use the logic for tab-indenting for when dedenting `}`?
- [ ] Let the autocompletion also look at method definitions with matching variable names (ignoring types, for now).
- [ ] Find out which keys could be used for "go to definition" and "rename".
- [ ] If pressing return at the end of the document, after a full screen, then also scroll down 1 line.
- [ ] Plugins? When there's "txt2something" and "something2txt", o should be able to edit "something" files.
      This could be used for hex editing, editing ELF files etc.
- [ ] Tab in the middle of a line, especially on a `|` character, could insert spaces until the `|` alignes with the `|` above, if applicable
      (For Markdown tables).
- [ ] Auto-detect if a loaded file uses "\t", "  ", "   ", "    ", or "        " for indentation.
- [ ] Be able to browse the search history with arrow up and down when searching. Introduce a search history.
- [ ] Load ViM and Emacs location after the file loads into a bookmark that can be jumped to with `ctrl-j`, for increased loading speed. Benchmark.
