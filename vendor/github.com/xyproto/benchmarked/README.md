# Benchmarked

The quest to find a faster `bytes.Equal` function.

So far, this function is 24% faster than `bytes.Equal`:

```go
func Equal(a, b []byte) bool {
    la := len(a)
    lb := len(b)
    if la != lb {
        return false
    }
    for i := 0; i < lb; i++ {
        // Be able to escape while finding the lenght of b
        if i >= la || a[i] != b[i] {
            return false
        }
    }
    return true
}
```

For comparison, `bytes.Equal` looks like this:

```go
func Equal(a, b []byte) bool {
    return string(a) == string(b)
}
```

Benchmark results:

```
goos: linux
goarch: amd64
pkg: github.com/xyproto/benchmarked
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkEqual/Equal6-12         	  988906	      1025 ns/op
BenchmarkEqual/Equal3-12         	 1427120	       851.0 ns/op
BenchmarkEqual/bytes.Equal-12    	 1622636	       728.2 ns/op
BenchmarkEqual/Equal1-12         	 1643167	       740.3 ns/op
BenchmarkEqual/Equal8-12         	 1681600	       711.9 ns/op
BenchmarkEqual/Equal10-12        	 1688145	       710.6 ns/op
BenchmarkEqual/Equal12-12        	 1810796	       664.9 ns/op
BenchmarkEqual/Equal11-12        	 1876164	       636.7 ns/op
BenchmarkEqual/Equal2-12         	 1892454	       627.3 ns/op
BenchmarkEqual/Equal4-12         	 1897585	       639.5 ns/op
BenchmarkEqual/Equal7-12         	 1910326	       628.7 ns/op
BenchmarkEqual/Equal5-12         	 1932075	       626.8 ns/op
BenchmarkEqual/Equal13-12        	 2119182	       561.9 ns/op
BenchmarkEqual/Equal9-12         	 2168968	       552.3 ns/op
PASS
ok  	github.com/xyproto/benchmarked	17.929s
```

I am aware that perfect benchmarking is a tricky.

Please let me know if you have improvements to how the functions are benchmarked!

## General info

* Version: 0.0.1
* License: BSD
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
