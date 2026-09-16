package clientdata

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func optionalReservedFloat(raw string) (float32, error) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(raw), 32)
	if err != nil {
		return 0, fmt.Errorf("reserved random centre %q: %w", raw, err)
	}
	value := float32(parsed)
	if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
		return 0, fmt.Errorf("reserved random centre %q is not finite", raw)
	}
	return value, nil
}
