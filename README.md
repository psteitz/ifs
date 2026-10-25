# What is this?
Some experimentation with Go's image library, goroutines, complex numbers and http server to generate and serve ifs-based fractal images.

Some of the core image-generation code is adapted from the [mandelbrot](https://github.com/adonovan/gopl.io/tree/master/ch3/mandelbrot) example in [The Go Programming Language](http://www.gopl.io/).

# How to run it
0. Install go if you do not already have it installed (see https://go.dev/doc/install).
1. Clone this repo ``git clone https://github.com/psteitz/ifs.git``
2. From the base directory of the cloned repo, ``go run main.go``
3. Open a browser and hit ``http://localhost:8000/julia`` to see an example animated fractal image.

Step 2. starts an http server. When you are finished playing with it, use ctrl-C to kill it.

# What it does
The generated images are related to [Julia sets](https://en.wikipedia.org/wiki/Julia_set).  The brightest points in the images are close to points in the Julia set associated with the process. 

## Julia sets for the Mandelbrot map, $z \to z^2 + c$

The request path ``http://localhost:8080/juliaSingle`` expects two request parameters, ``re`` and ``im``. The generated image shows the eventual behavior of the iterative function system $z \to z^2 + c$ where $z$ is a complex number corresponding to a point in the window of the image and $c$ is the complex number with real part equal to ``re`` and imaginary part equal to ``im``.  

The window is the square from -1.5 to 1.5 in both real and imaginary dimensions in the complex plane.  If a point is colored black, that means that when ``z`` is set initially to that point and $z \to z^2 + c$  is iterated repeatedly, the value remains small in modulus (i.e., the point does not "escape to infinity"). Points that do escape to infinity are colored according to how many iterations it takes for their modulus to exceed 10.  The very bright points are likely close the the Julia set for the process.  For example, ``http://localhost:8000/juliaSingle?re=-0.8&im=0.156`` generates an image whose brightest points correspond to points in the Julia set for $z \to z^2 + (-0.8 + 0.156i)$

The request path ``http://localhost:8080/julia`` generates animated gifs that do what ``http://localhost:8080/juliaSingle`` does, but for a range of ``c`` values that move in and out of the [Mandelbrot Set](https://en.wikipedia.org/wiki/Julia_set).  The path that ``c`` traverses is determined by the ``paramPath`` request parameter. 
1. ``Angor`` moves $c$ along the real axis, back and forth between -1.25 and 1.25 (near edges of the Mandelbrot set).
2. ``Exp`` moves $c$ around the circle, $.7885e^{i\alpha}$ where $\alpha$ goes from 0 to $2\pi$.
3. ``Wabbit`` moves $c$ back and forth along a line near the point $.3887 - .2158i$ which is near the boundary of the Mandelbrot set.  Use ``http://localhost:8080/julia?paramPath=Wabbit``or ``Angor`` to see animations along the other paths.

## Newton's method IFS
The request path ``http://localhost:8080/newton`` generates a single image showing the eventual behavior of [Newton's method](https://en.wikipedia.org/wiki/Newton%27s_method) applied to find complex roots of the equation $z^4 - 1 = 0$ (primitive 4th roots of unity) when starting with a point in the complex plane.  In this case, the window goes from -2 to 2 in both real and complex coordinates and points are colored according to which root the iterates converge to:
| Root       | Color        |          
|:-----------:|:-------------:|
| 1 | red |
| -1 | blue |
| i | green |
| -i | purple|

Points that don't converge to any root are colored black and brightness of the colored points is determined by how long the iterates take to converge to the respective root.

The ``/newton`` path also supports a more general form that allows you to specify a list of arbitrary complex numbers $z_0, ..., z_{n-1}$ for the roots.  See the request parameters table for instructions on how to provide the roots in the querystring.  

In this case,
Newton's method is applied to the polynomial 
$$ p(z) = (z - z_0)(z - z_1) \dots (z - z_{n-1}) $$
The standard Newton-Raphson formula is
$$z_{k+1}=z_{k}-\frac{p(z_{k})}{p^{\prime}(z_{k})}$$
which for $p$ as defined above is:
$$z_{k+1}=z_{k}-\frac{1}{\sum _{i=0}^{n}\frac{1}{z_{k}-z_{i}}}$$

This simple form can be derived using logarithmic differentiation. Start by taking the natural logarithm of both sides of the polynomial's defining formula:
$$p(z)=\prod _{i=0}^{n}(z-z_{i})$$
$$\ln (p(z))=\ln \left(\prod _{i=0}^{n}(z-z_{i})\right)=\sum _{i=0}^{n}\ln (z-z_{i})$$
By the chain rule,
$$\frac{d}{dz}[\ln(p(z))] = \frac{p'(z)}{p(z)}$$
but
$$\frac{d}{dz}\left[\sum_{i=0}^{n}\ln(z-z_{i})\right] = \sum_{i=0}^{n}\frac{1}{z-z_{i}}$$
so
$$\frac{p^{\prime }(z)}{p(z)}=\sum_{i=0}^{n}\frac{1}{z-z_{i}}$$
and therefore
$$\frac{p(z)}{p^{\prime }(z)}=\frac{1}{\sum _{i=0}^{n}\frac{1}{z-z_{i}}}$$
Substitute back into Newton's method formula gives
$$z_{k+1}=z_{k}-\frac{1}{\sum _{i=0}^{n}\frac{1}{z_{k}-z_{i}}}$$

 
# Request parameters

```/juliaSingle``` has two request parameters:
| Parameter       | Meaning      | Default value |   
|-------------|-------------|-------------|
| re | Real part of c parameter | -1.25  |
| im | Imaginary part of c parameter | 0  |
***


```/julia``` recognizes 3 parameters:
| Parameter       | Meaning      | Default value |  
|-------------|-------------|-------------|
| paramPath | name of paramter path function | Exp  |
| numframes | Number of frames to compute along paramPath | 64  |
| numworkers | Number of goroutines to concurrently build frames | 4 |
***

```/newton``` recognizes ```numframes``` and ```numworkers``` as above.

Increasing the number of frames will make the animation go more slowly and smoothly, but will take longer to compute.  Increasing the number of workers can speed things up if the run host has a lot of available compute.

```/newton``` also optionally accepts a list of complex roots, specified using ``roots[i][real]`` and ``roots[i][imag]`` parameters, where ``i`` is the (0-based) index of the root in the list. For example,
```
http://localhost:8000/newton?roots[0][real]=1&roots[0][imag]=1&roots[1][real]=-1.5&roots[1][imag]=0.5&roots[2][real]=0&roots[2][imag]=-1.5
```
applies Newton's method to find the roots of $(z - (1+i))(z - (-1.5+.5i))(z - (-1.5i))$. Each root is assigned a distinct color spaced evenly around the color wheel, with brightness determined by how long the iterates take to converge, as in the default case above. If no ``roots`` parameters are supplied, ``/newton`` defaults to the 4th-roots-of-unity example described above. Every supplied root index must have both a ``real`` and an ``imag`` value or the request fails with a 400 error.

