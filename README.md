# ifs
Some experimentation with Go's image library, goroutines, complex numbers and http server to generate and serve ifs-based fractal images.

Comparing the run time for julia and juliaMulti shows impressive gains from using goroutines to concurrently generate image frames.

Some of the core image-generation code is adapted from the [mandelbrot](https://github.com/adonovan/gopl.io/tree/master/ch3/mandelbrot) example in [The Go Programming Language]([](http://www.gopl.io/)).

