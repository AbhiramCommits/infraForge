package config

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ParseMemory parses a human-readable byte string such as "512", "256M", or
// "1G" into a byte count. Suffixes are interpreted as binary multiples and
// may be given as K, KB, KiB, M, MB, MiB, G, GB, GiB, T, TB, or TiB,
// case-insensitively.
func ParseMemory(s string) (int64, error) {
	orig := s
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty memory value")
	}
	split := len(s)
	for split > 0 && isAlpha(s[split-1]) {
		split--
	}
	num, suffix := strings.TrimSpace(s[:split]), strings.ToLower(s[split:])
	mult, ok := map[string]int64{
		"":  1,
		"b": 1,
		"k": 1 << 10, "kb": 1 << 10, "kib": 1 << 10,
		"m": 1 << 20, "mb": 1 << 20, "mib": 1 << 20,
		"g": 1 << 30, "gb": 1 << 30, "gib": 1 << 30,
		"t": 1 << 40, "tb": 1 << 40, "tib": 1 << 40,
	}[suffix]
	if !ok {
		return 0, fmt.Errorf("invalid size %q: unknown suffix %q", orig, suffix)
	}
	n, err := strconv.ParseInt(num, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size %q: %w", orig, err)
	}
	if n < 0 {
		return 0, fmt.Errorf("invalid size %q: must not be negative", orig)
	}
	if mult > 1 && n > math.MaxInt64/mult {
		return 0, fmt.Errorf("invalid size %q: overflows int64", orig)
	}
	return n * mult, nil
}

func isAlpha(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}
