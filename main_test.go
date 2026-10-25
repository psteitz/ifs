package main

import (
	"net/url"
	"testing"
)

func TestParseRoots_NoParams(t *testing.T) {
	roots, err := parseRoots(url.Values{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if roots != nil {
		t.Errorf("parseRoots(empty) = %v, want nil", roots)
	}
}

func TestParseRoots_IgnoresUnrelatedParams(t *testing.T) {
	values := url.Values{
		"numframes":  []string{"64"},
		"paramPath":  []string{"Exp"},
		"roots[foo]": []string{"1"}, // does not match the regexp (non-numeric index)
	}
	roots, err := parseRoots(values)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if roots != nil {
		t.Errorf("parseRoots(unrelated params) = %v, want nil", roots)
	}
}

func TestParseRoots_SingleRoot(t *testing.T) {
	values := url.Values{
		"roots[0][real]": []string{"5"},
		"roots[0][imag]": []string{"3"},
	}
	roots, err := parseRoots(values)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []complex128{complex(5, 3)}
	if len(roots) != len(want) || roots[0] != want[0] {
		t.Errorf("parseRoots(single root) = %v, want %v", roots, want)
	}
}

func TestParseRoots_MultipleRootsOrderedByIndex(t *testing.T) {
	// Keys are supplied out of index order; parseRoots must still return roots
	// ordered by index (0, 1, 2), not by querystring order.
	values := url.Values{
		"roots[2][real]": []string{"0"},
		"roots[2][imag]": []string{"-2"},
		"roots[0][real]": []string{"1.5"},
		"roots[0][imag]": []string{"1.5"},
		"roots[1][real]": []string{"-1"},
		"roots[1][imag]": []string{"-1.5"},
	}
	roots, err := parseRoots(values)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []complex128{
		complex(1.5, 1.5),
		complex(-1, -1.5),
		complex(0, -2),
	}
	if len(roots) != len(want) {
		t.Fatalf("parseRoots(multiple roots) returned %d roots, want %d", len(roots), len(want))
	}
	for i := range want {
		if roots[i] != want[i] {
			t.Errorf("parseRoots(multiple roots)[%d] = %v, want %v", i, roots[i], want[i])
		}
	}
}

func TestParseRoots_MissingPartIsError(t *testing.T) {
	values := url.Values{
		"roots[0][real]": []string{"5"},
		// roots[0][imag] is missing
	}
	_, err := parseRoots(values)
	if err == nil {
		t.Fatal("parseRoots(missing imag) returned nil error, want an error")
	}
}

func TestParseRoots_InvalidFloatIsError(t *testing.T) {
	values := url.Values{
		"roots[0][real]": []string{"not-a-number"},
		"roots[0][imag]": []string{"3"},
	}
	_, err := parseRoots(values)
	if err == nil {
		t.Fatal("parseRoots(invalid float) returned nil error, want an error")
	}
}
