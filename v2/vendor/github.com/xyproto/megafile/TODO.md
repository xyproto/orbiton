# Plans

- [ ] If the search filter starts with ".", then also filter on hidden files.
- [ ] When the path is wider than the terminal width, shorten the path a bit.
- [ ] When there is only one file to toggle between with `ctrl-n`/`ctrl-p`, then don't try to toggle anything.
- [ ] Go to next/prev file with `ctrl-n`/`ctrl-p` when the cursor is on a filename.
- [ ] Refactor the themes out of Orbiton into a `themes` package, and then use that package also from Megafile.
- [ ] If there is only one file in a directory, do not let `ctrl-n`/`ctrl-p` go to next/prev file in Orbiton.
- [ ] Let "tab" with no text in switch between the active directories instead of `ctrl-n`/`ctrl-p`.
- [ ] Let `ctrl-n`/`ctrl-p` scroll the preview text down/up.

## Already done?

- [ ] Move/refactor the terminal image viewing code from Orbiton to Megafile,
      then use this code from Orbiton too.
- [ ] Render SVG files using a Go package. Download and cache fonts, as needed.
