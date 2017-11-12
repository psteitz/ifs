# ifs
Some experimentation with Go's image library, goroutines, complex numbers and http server to generate and serve ifs-based fractal images.

There are 4 implementations of an algorithm that generates an animated GIF with frames containing Julia sets for a range of parameter values:

* `julia` is single-threaded, generating all of the image frames in main
* `juliaMulti` generates frames in concurrent goroutines, using a (non-threadsafe) map to store the frames
* `juliaMultiS` is `juliaMulti` modified to use a `sync.Map` to store the frames
* `juliaMultiSm` is `juliaMulti` modified to use a manually synchronized map

Comparing the run times for the different julia routines shows the expected gains from concurrency and the expected order among `juliaMulti` (fastest), `juliaMultiS` (second) and `juliaMultiSm` (slowest).  The usage in `juliaMulti` matches the use case described in the [sync.Map godoc](https://golang.org/pkg/sync/#Map).

Some of the core image-generation code is adapted from the [mandelbrot](https://github.com/adonovan/gopl.io/tree/master/ch3/mandelbrot) example in [The Go Programming Language]([](http://www.gopl.io/)).

