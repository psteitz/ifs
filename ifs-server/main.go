// A little image server that generates and serves PNG and animated GIF images
// illustrating the eventual behavior of some classic iterated function systems.
//
// The newton function generates a png showing Newton's method IFS seeking roots
// of p(z) = z^4 - 1.  The julia function generates Julia sets showing eventual
// behavior under the process z -> z^2 + c for a range of c values creating an
// animated GIF with each frame corresponding to a different c value.
//
// Portions of the image-generation code are a adapted from code in
// gopl.io/ch3/mandelbrot from the wonderful book _The Go Programming Language_
// by Alan A. A. Donovan & Brian W. Kernighan.
// The adapted code is provided under
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/
package main

import (
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"log"
	"math"
	"math/cmplx"
	"net/http"
	"time"
	"strconv"
)

func main() {
	http.HandleFunc("/newton", newton)    			// Single png 4th roots of unity
	http.HandleFunc("/julia", julia)      			// Animated GIF of Julia set images
	http.HandleFunc("/juliaSingle", juliaSingle)   	// Single png of a Julia set
	log.Fatal(http.ListenAndServe("localhost:8000", nil))
}

// Creates a PNG image showing eventual behavior of Newton's method IFS
// seeking 4th roots of unity.  Points in the complex plane are colored according
// to eventual behavior when they are taken as initial guesses.
func newton(w http.ResponseWriter, r *http.Request) {
	const (
		xmin, ymin, xmax, ymax = -2, -2, +2, +2
		width, height          = 1024, 1024
	)

	img := image.NewRGBA64(image.Rect(0, 0, width, height))
	for py := 0; py < height; py++ {
		y := float64(py)/height*(ymax-ymin) + ymin
		for px := 0; px < width; px++ {
			x := float64(px)/width*(xmax-xmin) + xmin
			z := complex(x, y)
			img.Set(px, py, newtonIFS(z, 2000))
		}
	}
	png.Encode(w, img)
}

// Creates a PNG image of a single Julia set for the process z->z^2 + c.
// The c parameter is constructed from the re and im request parameters.
func juliaSingle(w http.ResponseWriter, r *http.Request) {
	const (
		xmin, ymin, xmax, ymax = -2, -2, +2, +2
		width, height          = 1024, 1024
	)

	// Get c from request querystring
	re, err := strconv.ParseFloat(r.URL.Query().Get("re"), 64)
	if err != nil {
		re = -1.25
		log.Println("re missing or invalid - settting to -1.25")
	}
	im, err := strconv.ParseFloat(r.URL.Query().Get("im"), 64)
	if err != nil {
		im = 0
		log.Println("im missing or invalid - settting to 0")
	}
	c := complex(re, im)
	img := image.NewRGBA64(image.Rect(0, 0, width, height))
	for py := 0; py < height; py++ {
		y := float64(py)/height*(ymax-ymin) + ymin
		for px := 0; px < width; px++ {
			x := float64(px)/width*(xmax-xmin) + xmin
			z := complex(x, y)
			result := juliaIFS(z, c, 400, 10.0)
			co := color.RGBA64{0, 0, 0, 60000}
			if result > 0 {
				co = color.RGBA64{0, uint16(2000*result), 60000 - uint16(2000*result), 60000}
			}
			img.Set(px, py, co)
		}
	}
	png.Encode(w, img)
}

// frameParameter is an indexed c parameter for the process z -> z^2 + c
type frameParameter struct {
	index int
	c complex128
}

// frame is an indexed image
type frame struct {
	index int
	img *image.Paletted
}

// julia creates an animated GIF with frames displaying Julia sets for the process
//   z -> z^2 + c
// Each frame shows the Julia set for a different c value.  The progression of c values
// is determined by the parampath request paramter.  The recognized parampath values are:
//  Exp:     The c values are of the form .7885 e^ia where a ranges from 0 to 2pi.
//           As a goes from 0 to 2pi, c goes in and out of the Mandelbrot set.
//           This parameterization is borrowed from one of the examples in
//           https://en.wikipedia.org/wiki/Julia_set
//  Angor:   The c values range from -1.45 to 1.25 along the real axis
//  Wabbit:  The c values linearly about  .3887 - .2158i with both parameters
//           moving from .03 below to .03 above these values.
//
// Frames are generated concurrently by goroutines.
// The other request parameters are
//  numworkers:  the number of goroutines to exexute
//  numframes:   the number of frames in the animation
//
func julia(w http.ResponseWriter, r *http.Request) {
	const (
		xmin, ymin, xmax, ymax = -2, -2, +2, +2
		width, height          = 1024, 1024
		delay                  = 8
	)

	// Create a map of parameter functions, keyed by name
	type paramFunc func(int, int) complex128
	paramFuncs := map[string]paramFunc{
		"Angor":  watFunc,
		"Exp":    expFunc,
		"Wabbit": linFunc,
	}

	start := time.Now()

	// Get parameters from request querystring
	paramPath := r.URL.Query().Get("parampath")
	if paramFuncs[paramPath] == nil {
		paramPath = "Angor"
		log.Println("parampath missing or invalid - settting to default")
	}
	nFrames, err := strconv.Atoi(r.URL.Query().Get("numframes"))
	if err != nil {
		nFrames = 64  // Ignore bad querystring value, replacing with default
		log.Println("numframes missing or invalid - settting to default")
	}
	nWorkers, err := strconv.Atoi(r.URL.Query().Get("numworkers"))
	if err != nil {
		nWorkers = 4  // Ignore bad querystring value, replacing with default
		log.Println("numworkers missing or invalid - settting to default")
	}

	log.Printf(" Starting job with nframes = %d nworkers = %d parampath = %s \n", nFrames, nWorkers, paramPath)

	anim := gif.GIF{LoopCount: nFrames}          // The animated GIF we are building
	jobs := make(chan *frameParameter, nFrames)  // <i, c> pairs where c is the parameter for ith frame
	results := make(chan *frame, nFrames)        // Channel for workers to deliver completed frames
	frames := make ([] *image.Paletted, nFrames) // Completed frames

	for k := 0; k < nFrames; k++ { // Push frame generation jobs into the channel
		cp := paramFuncs[paramPath](k, nFrames)
		fp := frameParameter{
			k,
			cp,
		}
		jobs <- &fp
	}

	for i := 0; i < nWorkers; i++ { // Start the worker goroutines
		go frameWorker(jobs, results)
	}
	close(jobs) // Close the channel

	for i := 0; i < nFrames; i++ {
		frame := <-results
		frames[frame.index] = frame.img
	}

	for i := 0; i < nFrames; i++ { // add frames *in order*
		frame := frames[i]
		anim.Delay = append(anim.Delay, delay)
		anim.Image = append(anim.Image, frame)
	}
	elapsed := time.Since(start)
	log.Printf("Took %s", elapsed)
	gif.EncodeAll(w, &anim)
}

func watFunc(i int, nFrames int) complex128 {
	const (
		paramWidth             = 0.2
		paramStart             = -1.45
	)
	halframes := nFrames/2
	delta := paramWidth / float64(halframes)
	var alpha float64;
	if (i < halframes) {
		alpha = float64(i) * delta
	} else {
		alpha = paramWidth - float64(i - halframes) * delta
	}
	return complex(paramStart + alpha, 0)
}

func linFunc(i int, nFrames int) complex128 {
	const (
		center = complex(.3887, -.2158)
	paramWidth = 0.06
	)
	halframes := nFrames/2
	delta := paramWidth / float64(halframes)
	var alpha float64;
	if (i < halframes) {
		alpha = float64(i) * delta
	} else {
		alpha = paramWidth - float64(i - halframes) * delta
	}
	return complex(real(center) + alpha, imag(center) + alpha)
}

func expFunc(i int, nFrames int) complex128 {
	return .7885 * cmplx.Exp(complex(0, float64(i) * 2 * math.Pi / float64(nFrames)))
}

// frameworker is a worker goroutine to generate a frame.
// Takes a frame index i from the input jobs channel and creates the image for the ith frame,
// returning the index and the completed image on the results channel.  The paramFunc parameter
// is applied to the int from the input channel to get the c value.
func frameWorker(jobs <-chan *frameParameter, results chan<- *frame) {
	const (
		xmin, ymin, xmax, ymax = -2, -2, +2, +2
		width, height          = 1024, 1024
		delay                  = 8
	)

	opts := gif.Options{
		NumColors: 256,
		Drawer:    draw.FloydSteinberg,
	}
	for fp := range jobs {
		img := image.NewRGBA64(image.Rect(0, 0, width, height))
		for py := 0; py < height; py++ {
			y := float64(py)/height*(ymax-ymin) + ymin
			for px := 0; px < width; px++ {
				x := float64(px)/width*(xmax-xmin) + xmin
				z := complex(x, y)
				j:= juliaIFS(z, fp.c, 400, 10.0)
				c := color.RGBA64{0, 0, 0, 0}
				if j > 0 {
					c = color.RGBA64{0, uint16(2000*j), 60000 - uint16(2000*j), 60000}
				}
				img.Set(px, py, c)
			}
		}

		// Convert img to a paletted image
		b := img.Bounds()
		pimg := image.NewPaletted(b, palette.Plan9[:opts.NumColors])
		opts.Drawer.Draw(pimg, b, img, image.ZP)
		results <- &frame{
			fp.index,
			pimg,
		}
		log.Println("Finished Frame number ", fp.index)
	}
}

// mewtomIFS iterates Newton's method to find a root of p(x) = x^4 - 1 starting with initial guess = z.
// Returns a color coded as follows:
//   if the iterates do not converge (max iterations and not close to any root), black
//   if the iterates converge, then
//      1 <-> red
//     -1 <-> blue
//      i <-> green
//     -i <-> purple
//     with saturation dampened by the number of iterations required for the iterations to converge.
func newtonIFS(z complex128, contrast int) color.RGBA64 {
	const (
		iterations = 400
		one        = complex(1, 0)
		minusOne   = complex(-1, 0)
		posI       = complex(0, 1)
		negI       = complex(0, -1)
		tol        = 1e-16
	)
	for i := 0; i < iterations; i++ {
		z -= (z - 1/(z*z*z)) / 4
		if cmplx.Abs(z-one) < tol {
			return color.RGBA64{60000 - uint16(contrast*i), 0, 0, 60000}
		}
		if cmplx.Abs(z-minusOne) < tol {
			return color.RGBA64{0, 60000 - uint16(contrast*i), 0, 60000}
		}
		if cmplx.Abs(z-posI) < tol {
			return color.RGBA64{0, 0, 60000 - uint16(contrast*i), 60000}
		}
		if cmplx.Abs(z-negI) < tol {
			return color.RGBA64{60000 - uint16(contrast*i), 0, 60000 - uint16(contrast*i), 60000}
		}
	}
	return color.RGBA64{0, 0, 0, 0}
}

// juliaIFS iterates the process z -> z^2 + c starting at z until either maxIter iterations have
// completed or the modulus of an iterate exceeds big.  Returns 0 in the first case (no escape);
// otherwise the number of iterations required to escape.
func juliaIFS(z complex128, c complex128, maxIter int, big float64) int {
	for i := 0; i < maxIter; i++ {
		z = z*z + c
		if cmplx.Abs(z) > big {
			return i
		}
	}
	return 0
}
