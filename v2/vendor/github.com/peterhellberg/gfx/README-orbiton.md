# Local fork of github.com/peterhellberg/gfx

This directory is a vendored-in-source fork of
[github.com/peterhellberg/gfx](https://github.com/peterhellberg/gfx),
referenced from Orbiton's `go.mod` via:

```
replace github.com/peterhellberg/gfx => ./third_party/gfx
```

The only divergence from upstream is `http.go`: the original imports
`net/http` for four convenience helpers (`Get`, `GetPNG`, `GetImage`,
`GetTileset`) that Orbiton does not use. They are replaced here with
error-returning stubs so Orbiton's binary does not drag in `net/http`,
`crypto/tls` and their transitive dependencies.

To resync with upstream:

1. `go get github.com/peterhellberg/gfx@<new-version>` (temporarily drop the
   `replace` directive, or point it at a tag).
2. Copy the upstream sources over this directory.
3. Re-apply the `http.go` stub (`git checkout -- http.go` will do if this
   file is committed).
4. Re-add the `replace` line and run `go mod tidy && go mod vendor`.
