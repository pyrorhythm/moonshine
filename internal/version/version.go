package version

import (
	"strconv"
	"strings"
)

// Normalize strips brew's bottle revision suffix (_N) and normalizes the string.
// "2.41.0_1" → "2.41.0", "20.11.0" → "20.11.0", "v0.41.0" → "0.41.0"
func Normalize(v string) string {
	v = strings.TrimPrefix(v, "v")
	if idx := strings.LastIndex(v, "_"); idx != -1 {
		rest := v[idx+1:]
		if isDigits(rest) {
			v = v[:idx]
		}
	}
	return v
}

// Equal reports whether two version strings represent the same version after normalization.
func Equal(a, b string) bool {
	return Compare(a, b) == 0
}

// Compare returns -1, 0, or 1 after normalizing both versions.
func Compare(a, b string) int {
	a = Normalize(a)
	b = Normalize(b)
	if a == b {
		return 0
	}
	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")
	maxLen := len(partsA)
	if len(partsB) > maxLen {
		maxLen = len(partsB)
	}
	for i := range maxLen {
		na := partInt(partsA, i)
		nb := partInt(partsB, i)
		if na < nb {
			return -1
		}
		if na > nb {
			return 1
		}
	}
	return 0
}

func partInt(parts []string, i int) int {
	if i >= len(parts) {
		return 0
	}
	n, err := strconv.Atoi(parts[i])
	if err != nil {
		return 0
	}
	return n
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
