package engine

import (
	"image"
	"image/color"
	"image/png"
	"io"
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
		y := float64(py)/height*(ymax-ymin) + ymin
		for px := 0; px < width; px++ {
			x := float64(px)/width*(xmax-xmin) + xmin
			z := complex(x, y)
			img.Set(px, py, newtonIFS(z, 2000))
		}
	}
	png.Encode(w, img)
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
