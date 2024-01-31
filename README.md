# What is this?
Some experimentation with Go's image library, goroutines, complex numbers and http server to generate and serve ifs-based fractal images.

Some of the core image-generation code is adapted from the [mandelbrot](https://github.com/adonovan/gopl.io/tree/master/ch3/mandelbrot) example in [The Go Programming Language](http://www.gopl.io/).

# How to run it
0. Install go if you do not already have it installed (see https://go.dev/doc/install).
1. Clone this repo ``git clone https://github.com/psteitz/ifs.git``
2. From the base directory of the cloned repo, ``go run main.go``
3. Open a browser and hit ``http://localhost:8080/julia`` to see an example animated fractal image.

Step 2. starts an http server. When you are finished playing with it, use ctrl-C to kill it.

# What it does
The generated images are related to [Julia sets](https://en.wikipedia.org/wiki/Julia_set).  The brightest points in the images are close to points in the Julia set associated with the process. The request path ``http://localhost:8080/juliaSingle`` expects two request parameters, ``re`` and ``im``. The generated image shows the eventual behavior of the iterative function system ``z -> z^2 + c`` where ``z`` is a complex number corresponding to a point in the window of the image and ``c`` is the complex number with real part equal to ``re`` and imaginary part equal to ``im``.  

The window is the square from -1.5 to 1.5 in both real and imaginary dimensions in the complex plane.  If a point is colored black, that means that when ``z`` is set initially to that point and ``z -> z^2 + c`` is iterated repeatedly, the value remains small in modulus (i.e., the point does not "escape to infinity"). Points that do escape to infinity are colored according to how many iterations it takes for their modulus to exceed 10.  The very bright points are likely close the the Julia set for the process.  For example, ``http://localhost:8000/juliaSingle?re=-0.8&im=0.156`` generates an image whose brightest points correspond to points in the Julia set for ``z -> z^2 + (-0.8 + 0.156i)``

The request path ``http://localhost:8080/julia`` generates animated gifs that do what ``http://localhost:8080/juliaSingle`` does, but for a range of ``c`` values that move in and out of the [Mandelbrot Set](https://en.wikipedia.org/wiki/Julia_set).  The path that ``c`` traverses is determined by the ``paramPath`` request parameter. 
1. ``Angor`` moves ``c`` along the real axis, back and forth between -1.25 and 1.25 (near edges of the Mandelbrot set).
2. ``Exp`` moves ``c`` around the circle, ``.7885e^i*alpha`` where ``alfpha`` goes from 0 to 2pi.
3. ``Wabbit`` moves ``c`` back and forth along a line near the point ``.3887 - .2158i`` which is near the boundary of the Mandelbrot set.  Use ``http://localhost:8080/julia?paramPath=Wabbit``or ``Angor`` to see animations along the other paths.

The request path ``http://localhost:8080/newton`` generates a single image showing the eventual behavior of [Newton's method](https://en.wikipedia.org/wiki/Newton%27s_method) applied to find complex roots of the equation ``z^4 - 1 = 0`` (primitive 4th roots of unity) when starting with a point in the complex plane.  In this case, the window goes from -2 to 2 in both real and complex coordinates and points are colored according to which root the iterates converge to:
| Root       | Color        |          
|:-----------:|:-------------:|
| 1 | red |
| -1 | blue |
| i | green |
| -i | purple|

Points that don't converge to any root are colored black and brightness of the colored points is determined by how long the iterates take to converge to the respective root.
 
