package admin

import (
	"strconv"
	"strings"
)

const MinInvokeVersion = "6.12.0"

// CheckInvokeVersion returns true if the given version string meets the minimum requirement.
func CheckInvokeVersion(version string) bool {
	if version == "" {
		return false
	}
	minParts := parseVersion(MinInvokeVersion)
	actParts := parseVersion(version)

	for i := 0; i < 3; i++ {
		min := 0
		act := 0
		if i < len(minParts) {
			min = minParts[i]
		}
		if i < len(actParts) {
			act = actParts[i]
		}
		if act > min {
			return true
		}
		if act < min {
			return false
		}
	}
	return true // equal
}

func parseVersion(v string) []int {
	// Strip leading "v" and anything after a dash or space (e.g. "6.13.0.rc2" → "6.13.0")
	v = strings.TrimPrefix(v, "v")
	// Handle extra dots beyond major.minor.patch (e.g. "6.13.0.rc2")
	// and pre-release suffixes
	parts := strings.FieldsFunc(v, func(r rune) bool {
		return r == '.' || r == '-' || r == ' '
	})

	var nums []int
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			break // stop at non-numeric part (e.g. "rc2")
		}
		nums = append(nums, n)
	}
	return nums
}
