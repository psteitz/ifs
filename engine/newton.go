package engine

import (
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"math/cmplx"
)

// Creates a PNG image showing eventual behavior of Newton's method IFS
// seeking 4th roots of unity.  Points in the complex plane are colored according
// to eventual behavior when they are taken as initial guesses.
func Newton(w io.Writer) {
	const (
		xmin, ymin, xmax, ymax = -2, -2, +2, +2
		width, height          = 1024, 1024
	)

	img := image.NewRGBA64(image.Rect(0, 0, width, height))
	for py := 0; py < height; py++ {
		// Image row 0 is the top of the PNG, so map it to ymax (not ymin) to keep the
		// imaginary axis oriented the usual way, with positive values at the top.
		y := ymax - float64(py)/height*(ymax-ymin)
		for px := 0; px < width; px++ {
			x := float64(px)/width*(xmax-xmin) + xmin
			z := complex(x, y)
			img.Set(px, py, newtonIFS(z, 2000))
		}
	}
	png.Encode(w, img)
}

// NewtonExt creates a PNG image showing the eventual behavior of Newton's method IFS seeking
// the roots of the polynomial p(z) = (z - roots[0])(z - roots[1]) ... (z - roots[n-1]) for an
// arbitrary, caller-supplied list of complex roots. Points in the complex plane are colored
// according to which root the iterates started from that point converge to - see newtonIFSExt
// and colorForRoot for details of the iteration and coloring scheme.
func NewtonExt(roots []complex128, w io.Writer) {
	const (
		xmin, ymin, xmax, ymax = -2, -2, +2, +2
		width, height          = 1024, 1024
	)

	img := image.NewRGBA64(image.Rect(0, 0, width, height))
	for py := 0; py < height; py++ {
		// Image row 0 is the top of the PNG, so map it to ymax (not ymin) to keep the
		// imaginary axis oriented the usual way, with positive values at the top.
		y := ymax - float64(py)/height*(ymax-ymin)
		for px := 0; px < width; px++ {
			x := float64(px)/width*(xmax-xmin) + xmin
			z := complex(x, y)
			img.Set(px, py, newtonIFSExt(z, roots, 2000))
		}
	}
	png.Encode(w, img)
}

// mewtomIFS iterates Newton's method to find a root of p(x) = x^4 - 1 starting with initial guess = z.
// Returns a color coded as follows:
//
//	if the iterates do not converge (max iterations and not close to any root), black
//	if the iterates converge, then
//	   1 <-> red
//	  -1 <-> blue
//	   i <-> green
//	  -i <-> purple
//	  with saturation dampened by the number of iterations required for the iterations to converge.
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

// newtonIFSExt generalizes newtonIFS to seek the roots of an arbitrary polynomial
//
//	p(z) = (z - roots[0])(z - roots[1]) ... (z - roots[n-1])
//
// given explicitly as a list of complex roots, starting the iteration from z. As derived
// in the README, Newton's method applied to p simplifies to
//
//	z_{k+1} = z_k - 1 / sum_{i=0}^{n-1} 1/(z_k - roots[i])
//
// Returns a color coded as follows:
//   - if the iterates converge to roots[idx], a color uniquely associated with idx (see
//     colorForRoot), with brightness dampened by the number of iterations required to converge
//     (scaled by contrast), mirroring newtonIFS.
//   - if the iterates do not converge within the iteration budget, black.
func newtonIFSExt(z complex128, roots []complex128, contrast int) color.RGBA64 {
	const (
		iterations = 400
		tol        = 1e-9
	)
	n := len(roots)
	for i := 0; i < iterations; i++ {
		// If z has (numerically) landed on one of the roots, stop and color accordingly.
		for idx, root := range roots {
			if cmplx.Abs(z-root) < tol {
				return colorForRoot(idx, n, i, contrast)
			}
		}
		var sum complex128
		for _, root := range roots {
			sum += 1 / (z - root)
		}
		z -= 1 / sum
	}
	return color.RGBA64{0, 0, 0, 0}
}

// colorForRoot returns a color for the root at index idx out of n total roots. Roots are
// assigned evenly spaced hues around the HSV color wheel (hue = 360*idx/n) so that each root
// gets a visually distinct, stable color regardless of how many roots are supplied. The HSV
// "value" (brightness) is dampened by contrast*i, where i is the number of iterations Newton's
// method took to converge to the root, so points that converge quickly are brighter than ones
// that take longer - consistent with the dampening used in newtonIFS.
func colorForRoot(idx int, n int, i int, contrast int) color.RGBA64 {
	const maxValue = 60000
	hue := 360 * float64(idx) / float64(n)
	damp := contrast * i
	value := maxValue
	if damp < maxValue {
		value -= damp
	} else {
		value = 0
	}
	r, g, b := hsvToRGBA64(hue, 1.0, float64(value))
	return color.RGBA64{r, g, b, maxValue}
}

// hsvToRGBA64 converts a hue (in degrees, [0, 360)), saturation ([0, 1]) and value
// (interpreted directly as the maximum magnitude of an RGBA64 color channel) to RGBA64
// red, green and blue channel values using the standard HSV -> RGB conversion.
func hsvToRGBA64(hue, saturation, value float64) (r, g, b uint16) {
	c := value * saturation
	hPrime := hue / 60
	x := c * (1 - math.Abs(math.Mod(hPrime, 2)-1))
	var r1, g1, b1 float64
	switch {
	case hPrime < 1:
		r1, g1, b1 = c, x, 0
	case hPrime < 2:
		r1, g1, b1 = x, c, 0
	case hPrime < 3:
		r1, g1, b1 = 0, c, x
	case hPrime < 4:
		r1, g1, b1 = 0, x, c
	case hPrime < 5:
		r1, g1, b1 = x, 0, c
	default:
		r1, g1, b1 = c, 0, x
	}
	m := value - c
	return uint16(r1 + m), uint16(g1 + m), uint16(b1 + m)
}
