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
	"log"
	"net/http"
	"strconv"
	"github.com/psteitz/ifs/ifs-server/engine"
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
	engine.Newton(w)
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
	engine.JuliaSingle(complex(re, im), w)
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
//  Wabbit:  The c values vary linearly about  .3887 - .2158i with both parameters
//           moving from .03 below to .03 above these values.
//
// Frames are generated concurrently by goroutines.
// The other request parameters are
//  numworkers:  the number of goroutines to exexute
//  numframes:   the number of frames in the animation
//
func julia(w http.ResponseWriter, r *http.Request) {

	// "Set" of the valid parameter paths
	// paramPaths[foo] will return false (zero value) if foo is not in the list.
	paramPaths := map[string]bool{
		"Angor":  true,
		"Exp":    true,
		"Wabbit": true,
	}

	// Get parameters from request querystring
	paramPath := r.URL.Query().Get("parampath")
	if !paramPaths[paramPath] {
		paramPath = "Angor"
		log.Println("parampath missing or invalid - settting to default")
	}
	nFrames, err := strconv.Atoi(r.URL.Query().Get("numframes"))
	if err != nil {
		nFrames = 64 // Ignore bad querystring value, replacing with default
		log.Println("numframes missing or invalid - settting to default")
	}
	nWorkers, err := strconv.Atoi(r.URL.Query().Get("numworkers"))
	if err != nil {
		nWorkers = 4 // Ignore bad querystring value, replacing with default
		log.Println("numworkers missing or invalid - settting to default")
	}

	engine.Julia(nFrames, nWorkers, paramPath, w)
}
