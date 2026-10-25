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
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"

	"github.com/psteitz/ifs/engine"
)

func main() {
	http.HandleFunc("/newton", newton)           // Single png 4th roots of unity
	http.HandleFunc("/julia", julia)             // Animated GIF of Julia set images
	http.HandleFunc("/juliaSingle", juliaSingle) // Single png of a Julia set
	log.Fatal(http.ListenAndServe("localhost:8000", nil))
}

// rootParamRegexp matches querystring keys of the form roots[<index>][real] or
// roots[<index>][imag], used to specify an arbitrary list of polynomial roots for /newton.
var rootParamRegexp = regexp.MustCompile(`^roots\[(\d+)\]\[(real|imag)\]$`)

// Creates a PNG image showing eventual behavior of Newton's method IFS seeking roots of a
// polynomial.  Points in the complex plane are colored according to eventual behavior when
// they are taken as initial guesses.
//
// By default (no roots querystring parameters), the roots are the 4th roots of unity.
// Callers may instead supply an arbitrary list of complex roots z_0, ..., z_n-1 using
// querystring parameters of the form roots[i][real] and roots[i][imag], e.g.
//
//	/newton?roots[0][real]=1&roots[0][imag]=1&roots[1][real]=-1.5&roots[1][imag]=0.5&roots[2][real]=0&roots[2][imag]=-1.5
func newton(w http.ResponseWriter, r *http.Request) {
	roots, err := parseRoots(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(roots) == 0 {
		engine.Newton(w)
		return
	}
	engine.NewtonExt(roots, w)
}

// parseRoots extracts an ordered list of complex roots from querystring parameters of the
// form roots[i][real] and roots[i][imag] (i starting at 0).  Returns a nil slice (and nil
// error) if no such parameters are present.  Returns an error if an index is missing its
// real or imaginary part, or if a value cannot be parsed as a float.
func parseRoots(values url.Values) ([]complex128, error) {
	type rootParts struct {
		real, imag       float64
		hasReal, hasImag bool
	}
	partsByIndex := map[int]*rootParts{}
	for key, vals := range values {
		m := rootParamRegexp.FindStringSubmatch(key)
		if m == nil {
			continue
		}
		idx, err := strconv.Atoi(m[1])
		if err != nil {
			continue // regexp guarantees digits, so this should not happen
		}
		v, err := strconv.ParseFloat(vals[0], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid value for %s: %v", key, err)
		}
		p, ok := partsByIndex[idx]
		if !ok {
			p = &rootParts{}
			partsByIndex[idx] = p
		}
		if m[2] == "real" {
			p.real, p.hasReal = v, true
		} else {
			p.imag, p.hasImag = v, true
		}
	}

	if len(partsByIndex) == 0 {
		return nil, nil
	}

	indices := make([]int, 0, len(partsByIndex))
	for idx := range partsByIndex {
		indices = append(indices, idx)
	}
	sort.Ints(indices)

	roots := make([]complex128, 0, len(indices))
	for _, idx := range indices {
		p := partsByIndex[idx]
		if !p.hasReal || !p.hasImag {
			return nil, fmt.Errorf("root at index %d is missing its real or imaginary part", idx)
		}
		roots = append(roots, complex(p.real, p.imag))
	}
	return roots, nil
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
//
//	z -> z^2 + c
//
// Each frame shows the Julia set for a different c value.  The progression of c values
// is determined by the parampath request paramter.  The recognized parampath values are:
//
//	Exp:     The c values are of the form .7885 e^ia where a ranges from 0 to 2pi.
//	         As a goes from 0 to 2pi, c goes in and out of the Mandelbrot set.
//	         This parameterization is borrowed from one of the examples in
//	         https://en.wikipedia.org/wiki/Julia_set
//	Angor:   The c values range from -1.45 to 1.25 along the real axis
//	Wabbit:  The c values vary linearly about  .3887 - .2158i with both parameters
//	         moving from .03 below to .03 above these values.
//
// Frames are generated concurrently by goroutines.
// The other request parameters are
//
//	numworkers:  the number of goroutines to exexute
//	numframes:   the number of frames in the animation
func julia(w http.ResponseWriter, r *http.Request) {

	// "Set" of the valid parameter paths
	// paramPaths[foo] will return false (zero value) if foo is not in the list.
	paramPaths := map[string]bool{
		"Angor":  true,
		"Exp":    true,
		"Wabbit": true,
	}

	// Get parameters from request querystring
	paramPath := r.URL.Query().Get("paramPath")
	if !paramPaths[paramPath] {
		paramPath = "Exp"
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
