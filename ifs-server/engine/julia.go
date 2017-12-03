
package engine

import (
	"time"
	"log"
	"image/gif"
	"image"
	"math/cmplx"
	"math"
	"image/draw"
	"image/color"
	"image/color/palette"
)

func Julia (nFrames int, nWorkers int, paramPath string) {
	const (
		xmin, ymin, xmax, ymax = -2, -2, +2, +2
		width, height          = 1024, 1024
		delay                  = 8
	)

	// A paramFunc is a function that takes a frame number and number of frames as arguments
	// and returns a c value.  For example, watFunc varies the c parameter along the real axis
	// over a range from -1.45 to -1.25 (and back again) in increments determined by the number of frames.
	type paramFunc func(int, int) complex128

	// Create a map of parameter functions, keyed by name
	paramFuncs := map[string]paramFunc{
		"Angor":  watFunc,
		"Exp":    expFunc,
		"Wabbit": linFunc,
	}

	start := time.Now()

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

// watFunc varies c along the real axis, starting at -1.45, increasing to -1.25 (edge of the Mandelbrot set)
// and then returning to -1.45
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

// linFunc varies c about .3887 - .2158i, a point on the edge of the Mandelbrot set.
// The variation adds constant increments to both coordinates and then reduces along the same (linear) path.
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

// expFunc moves c around the circle, .7885e^i*alpha where alfpha goes from 0 to 2pi.
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