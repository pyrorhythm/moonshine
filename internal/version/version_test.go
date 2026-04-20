package version_test

import (
	"testing"

	"github.com/pyrorhythm/moonshine/internal/version"
)

func TestNormalize(t *testing.T) {
	cases := []struct{ in, want string }{
		{"2.41.0_1", "2.41.0"},
		{"2.41.0_12", "2.41.0"},
		{"2.41.0", "2.41.0"},
		{"v0.41.0", "0.41.0"},
		{"20.11.0", "20.11.0"},
		{"1.0.0_0", "1.0.0"},
		{"latest", "latest"},
	}
	for _, c := range cases {
		got := version.Normalize(c.in)
		if got != c.want {
			t.Errorf("Normalize(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestEqual(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"2.41.0", "2.41.0", true},
		{"2.41.0_1", "2.41.0", true},
		{"v0.41.0", "0.41.0", true},
		{"2.41.0", "2.42.0", false},
		{"1.0", "1.0.0", true},
	}
	for _, c := range cases {
		got := version.Equal(c.a, c.b)
		if got != c.want {
			t.Errorf("Equal(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "2.0.0", -1},
		{"2.0.0", "1.0.0", 1},
		{"2.41.0_1", "2.41.0", 0},
		{"1.10.0", "1.9.0", 1},
	}
	for _, c := range cases {
		got := version.Compare(c.a, c.b)
		if got != c.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
