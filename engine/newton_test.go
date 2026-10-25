package engine

import (
	"bytes"
	"image/color"
	"image/png"
	"testing"
)

// pixelFor returns the pixel coordinates that a given complex point z maps to under the
// top-down image orientation used by Newton and NewtonExt: row 0 (the top of the PNG)
// corresponds to ymax and row height-1 (the bottom) corresponds to ymin; column 0 (the
// left) corresponds to xmin and column width-1 (the right) corresponds to xmax. This is
// defined independently of the production code so it can catch a regression of the
// vertical-flip bug that was previously present in both functions.
func pixelFor(z complex128, xmin, ymin, xmax, ymax float64, width, height int) (int, int) {
	px := int((real(z) - xmin) / (xmax - xmin) * float64(width))
	py := int((ymax - imag(z)) / (ymax - ymin) * float64(height))
	return px, py
}

func TestNewton_RootPlacement(t *testing.T) {
	const xmin, ymin, xmax, ymax = -2, -2, 2, 2
	const width, height = 1024, 1024

	var buf bytes.Buffer
	Newton(&buf)
	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("failed to decode generated PNG: %v", err)
	}

	// Colors per newtonIFS: 1 -> red, -1 -> green, i -> blue, -i -> red+blue (purple).
	cases := []struct {
		name string
		root complex128
		want func(r, g, b uint32) bool
	}{
		{"root 1 is red", complex(1, 0), func(r, g, b uint32) bool { return r > 0 && g == 0 && b == 0 }},
		{"root -1 is green", complex(-1, 0), func(r, g, b uint32) bool { return g > 0 && r == 0 && b == 0 }},
		{"root i is blue", complex(0, 1), func(r, g, b uint32) bool { return b > 0 && r == 0 && g == 0 }},
		{"root -i is purple", complex(0, -1), func(r, g, b uint32) bool { return r > 0 && b > 0 && g == 0 }},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			px, py := pixelFor(c.root, xmin, ymin, xmax, ymax, width, height)
			r, g, b, _ := img.At(px, py).RGBA()
			if !c.want(r, g, b) {
				t.Errorf("pixel (%d,%d) for root %v has color (r=%d,g=%d,b=%d); orientation mapping may be wrong", px, py, c.root, r, g, b)
			}
		})
	}
}

// TestNewtonExt_RootPlacement is a regression test for the vertical-flip bug: it uses
// asymmetric roots (not related by reflection), renders an image, and checks that each
// root's basin appears at that root's own true coordinates rather than at a
// y-axis-flipped location.
func TestNewtonExt_RootPlacement(t *testing.T) {
	const xmin, ymin, xmax, ymax = -2, -2, 2, 2
	const width, height = 1024, 1024

	roots := []complex128{complex(1.5, 1.5), complex(-1, -1.5)}

	var buf bytes.Buffer
	NewtonExt(roots, &buf)
	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("failed to decode generated PNG: %v", err)
	}

	// With n=2, colorForRoot gives root 0 hue 0 (pure red) and root 1 hue 180 (cyan).
	cases := []struct {
		name string
		root complex128
		want func(r, g, b uint32) bool
	}{
		{"root 0 (1.5+1.5i) is red", roots[0], func(r, g, b uint32) bool { return r > 0 && g == 0 && b == 0 }},
		{"root 1 (-1-1.5i) is cyan", roots[1], func(r, g, b uint32) bool { return r == 0 && g > 0 && b > 0 }},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			px, py := pixelFor(c.root, xmin, ymin, xmax, ymax, width, height)
			r, g, b, _ := img.At(px, py).RGBA()
			if !c.want(r, g, b) {
				t.Errorf("pixel (%d,%d) for root %v has color (r=%d,g=%d,b=%d); orientation mapping may be wrong", px, py, c.root, r, g, b)
			}
		})
	}
}

func TestNewtonIFS_ConvergesToClosestRoot(t *testing.T) {
	const tiny = 1e-12
	cases := []struct {
		name string
		z    complex128
		want func(c color.RGBA64) bool
	}{
		{"near 1 -> red", complex(1+tiny, 0), func(c color.RGBA64) bool { return c.R > 0 && c.G == 0 && c.B == 0 }},
		{"near -1 -> green", complex(-1+tiny, 0), func(c color.RGBA64) bool { return c.G > 0 && c.R == 0 && c.B == 0 }},
		{"near i -> blue", complex(0, 1+tiny), func(c color.RGBA64) bool { return c.B > 0 && c.R == 0 && c.G == 0 }},
		{"near -i -> purple", complex(0, -1+tiny), func(c color.RGBA64) bool { return c.R > 0 && c.B > 0 && c.G == 0 }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := newtonIFS(c.z, 2000)
			if got.A == 0 {
				t.Fatalf("newtonIFS(%v) did not converge, got %v", c.z, got)
			}
			if !c.want(got) {
				t.Errorf("newtonIFS(%v) = %+v; color does not match expected root", c.z, got)
			}
		})
	}
}

func TestNewtonIFSExt_ConvergesToRoot(t *testing.T) {
	roots := []complex128{complex(2, 0), complex(-1, 1), complex(0, -2)}
	const tiny = 1e-12
	for idx, root := range roots {
		z := root + complex(tiny, 0)
		got := newtonIFSExt(z, roots, 2000)
		if got.A == 0 {
			t.Fatalf("newtonIFSExt(%v) did not converge, got %+v", z, got)
		}
		want := colorForRoot(idx, len(roots), 0, 2000)
		if got != want {
			t.Errorf("newtonIFSExt(%v) = %+v, want color for root %d (%+v)", z, got, idx, want)
		}
	}
}

func TestNewtonIFSExt_NoConvergenceIsBlack(t *testing.T) {
	// z=0 is exactly equidistant from roots 1 and -1, so sum := 1/(0-1) + 1/(0-(-1))
	// is exactly 0, making the next iterate 1/sum a division by zero that produces
	// NaN/Inf components. Those never satisfy the convergence check, so the
	// iteration exhausts its budget and newtonIFSExt should report black.
	roots := []complex128{complex(1, 0), complex(-1, 0)}
	got := newtonIFSExt(complex(0, 0), roots, 2000)
	want := color.RGBA64{0, 0, 0, 0}
	if got != want {
		t.Errorf("newtonIFSExt(0) = %+v, want black %+v", got, want)
	}
}

func TestColorForRoot(t *testing.T) {
	const n = 2
	const contrast = 2000

	// Zero iterations (i=0) means full brightness.
	red := colorForRoot(0, n, 0, contrast)
	if red.R == 0 || red.G != 0 || red.B != 0 {
		t.Errorf("colorForRoot(0, %d, 0, %d) = %+v, want pure red at full brightness", n, contrast, red)
	}
	cyan := colorForRoot(1, n, 0, contrast)
	if cyan.R != 0 || cyan.G == 0 || cyan.B == 0 {
		t.Errorf("colorForRoot(1, %d, 0, %d) = %+v, want cyan at full brightness", n, contrast, cyan)
	}

	// Brightness should decrease as the iteration count i increases...
	dim := colorForRoot(0, n, 10, contrast)
	if dim.R >= red.R {
		t.Errorf("colorForRoot(0, %d, 10, %d).R = %d, want less than i=0 brightness %d", n, contrast, dim.R, red.R)
	}

	// ...and saturate at black once the damping exceeds the maximum value.
	black := colorForRoot(0, n, 1000, contrast)
	if black.R != 0 || black.G != 0 || black.B != 0 {
		t.Errorf("colorForRoot(0, %d, 1000, %d) = %+v, want black once damping saturates", n, contrast, black)
	}
}

func TestHsvToRGBA64(t *testing.T) {
	const value = 60000
	cases := []struct {
		hue     float64
		r, g, b uint16
	}{
		{0, 60000, 0, 0},       // red
		{60, 60000, 60000, 0},  // yellow
		{120, 0, 60000, 0},     // green
		{180, 0, 60000, 60000}, // cyan
		{240, 0, 0, 60000},     // blue
		{300, 60000, 0, 60000}, // magenta
	}
	for _, c := range cases {
		r, g, b := hsvToRGBA64(c.hue, 1.0, value)
		if r != c.r || g != c.g || b != c.b {
			t.Errorf("hsvToRGBA64(%v, 1.0, %v) = (%d,%d,%d), want (%d,%d,%d)", c.hue, value, r, g, b, c.r, c.g, c.b)
		}
	}
}
