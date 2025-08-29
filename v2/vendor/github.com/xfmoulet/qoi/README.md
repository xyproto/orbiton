# QOI - The “Quite OK Image” format for fast, lossless image compression - in Go 

Package and small utilities in pure Go, quite OK implementation

See [qoi.h](https://github.com/phoboslab/qoi/blob/master/qoi.h) for
the documentation.

More info at https://qoiformat.org/ 

## Performance

Performance is currently around half C version (optimized at `-O3`)

## Example Usage

- `cmd/qoiconv` converts between png <> qoi
- `cmd/qoibench` bench the en/decoding vs. golang png implementation
