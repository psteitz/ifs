// A little image server that generates and serves PNG and animated GIF images
// illustrating the eventual behavior of some classic iterated function systems.
//
// The newton function generates a png showing Newton's method IFS seeking roots
// of p(z) = z^4 - 1.  The julia and juliaMulti functions generate Julia sets
// showing eventual behavior under the process z -> z^2 + c for a range of c values
// creating an animated GIF with each frame corresponding to a different c value.
// The difference between these two is that juliaMulti executes the frame
// generation in concurrent goroutines.

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
)

func main() {
	http.HandleFunc("/newton", newton)
	http.HandleFunc("/julia", julia)
	http.HandleFunc("/juliaMulti", juliaMulti)
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

// julia creates an animated GIF with frames displaying Julia sets for the process
//   z -> z^2 + c
// The c values are of the form .7885 e^ia where a ranges from 0 to 2pi.
// As a goes from 0 to 2pi, c goes in and out of the Mandelbrot set.
//
// This parameterization is borrowed from one of the examples
// in https://en.wikipedia.org/wiki/Julia_set
func julia(w http.ResponseWriter, r *http.Request) {
	const (
		xmin, ymin, xmax, ymax = -2, -2, +2, +2
		width, height          = 1024, 1024
		nframes                = 64
		delay                  = 8
		delta                  = math.Pi / float64(nframes)
	)

	start := time.Now()
	anim := gif.GIF{LoopCount: nframes}
	alpha := 0.0
	for i := 0; i < nframes; i++ {
		img := image.NewRGBA64(image.Rect(0, 0, width, height))
		for py := 0; py < height; py++ {
			y := float64(py)/height*(ymax-ymin) + ymin
			for px := 0; px < width; px++ {
				x := float64(px)/width*(xmax-xmin) + xmin
				z := complex(x, y)
				j := juliaIFS(z, alpha)
				c := color.RGBA64{0, 0, 0, 0}
				if j > 0 {
					c = color.RGBA64{0, 0, 60000 - uint16(2000*j), 60000}
				}
				img.Set(px, py, c)
			}
		}

		// Convert img to a paletted image
		opts := gif.Options{
			NumColors: 256,
			Drawer:    draw.FloydSteinberg,
		}
		b := img.Bounds()
		pimg := image.NewPaletted(b, palette.Plan9[:opts.NumColors])
		if opts.Quantizer != nil {
			pimg.Palette = opts.Quantizer.Quantize(make(color.Palette, 0, opts.NumColors), img)
		}
		opts.Drawer.Draw(pimg, b, img, image.ZP)
		anim.Delay = append(anim.Delay, delay)
		anim.Image = append(anim.Image, pimg)

		alpha += delta
		log.Println("Finished frame number ", i)
	}
	gif.EncodeAll(w, &anim)
	elapsed := time.Since(start)
	log.Printf("Took %s", elapsed)
}

// juliaMulti is functionally equivalent to julia above, but frames are generated by goroutines.
func juliaMulti(w http.ResponseWriter, r *http.Request) {
	const (
		xmin, ymin, xmax, ymax = -2, -2, +2, +2
		width, height          = 1024, 1024
		nframes                = 64
		delay                  = 8
		delta                  = math.Pi / float64(nframes)
		nworkers               = 4
	)
	start := time.Now()
	anim := gif.GIF{LoopCount: nframes}     // The animated GIF we are building
	frames := make(map[int]*image.Paletted) // Frames to be added - key is frame number
	jobs := make(chan int, nframes)         // Frame numbers passed to workers
	done := make(chan struct{})             // Channel for workers to signal completion

	for k := 0; k < nframes; k++ { // Push frame generation jobs into the channel
		jobs <- k
	}
	for i := 0; i < nworkers; i++ { // Start the worker goroutines
		go frameWorker(jobs, frames, done)
	}
	close(jobs) // Close the channel

	for i := 0; i < nworkers; i++ { // Wait for workers to finish
		<-done
	}

	for i := 0; i < nframes; i++ { // add frames *in order*
		frame := frames[i]
		anim.Delay = append(anim.Delay, delay)
		anim.Image = append(anim.Image, frame)
	}
	elapsed := time.Since(start)
	log.Printf("Took %s", elapsed)
	gif.EncodeAll(w, &anim)
}

// frameworker is a worker goroutine to generate a frame.
// Takes a frame number i from the input channel and creates map[i], then signals
// completion on the (unbuffered) done channel.
func frameWorker(jobs <-chan int, frames map[int]*image.Paletted, done chan struct{}) {
	const (
		xmin, ymin, xmax, ymax = -2, -2, +2, +2
		width, height          = 1024, 1024
		nframes                = 64
		delay                  = 8
		delta                  = math.Pi / float64(nframes)
	)

	for i := range jobs {
		alpha := float64(i) * delta
		img := image.NewRGBA64(image.Rect(0, 0, width, height))
		for py := 0; py < height; py++ {
			y := float64(py)/height*(ymax-ymin) + ymin
			for px := 0; px < width; px++ {
				x := float64(px)/width*(xmax-xmin) + xmin
				z := complex(x, y)
				j := juliaIFS(z, alpha)
				c := color.RGBA64{0, 0, 0, 0}
				if j > 0 {
					c = color.RGBA64{0, 0, 60000 - uint16(2000*j), 60000}
				}
				img.Set(px, py, c)
			}
		}

		// Convert img to a paletted image
		opts := gif.Options{
			NumColors: 256,
			Drawer:    draw.FloydSteinberg,
		}
		b := img.Bounds()
		pimg := image.NewPaletted(b, palette.Plan9[:opts.NumColors])
		if opts.Quantizer != nil {
			pimg.Palette = opts.Quantizer.Quantize(make(color.Palette, 0, opts.NumColors), img)
		}
		opts.Drawer.Draw(pimg, b, img, image.ZP)
		frames[i] = pimg
		log.Println("Finished Frame number ", i)
	}
	done <- struct{}{} // Signal all frames completed
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

// juliaIFS iterates the process z -> z^2 + .7885 e^i*alpha starting at z until either 400 iterations have
// completed or the modulus of an iterate exceeds 10.  Returns 0 in the first case (no escape);
// otherwise the number of iterations required to escape.
func juliaIFS(z complex128, alpha float64) int {
	const (
		iterations = 400
		big        = 10.0
	)
	c := .7885 * cmplx.Exp(complex(0, alpha))
	for i := 0; i < iterations; i++ {
		z = z*z + c
		if cmplx.Abs(z) > big {
			return i
		}
	}
	return 0
}
