// File: pkg/device/utils.go
// Chat Session: newtron-20250706-01
// Session Timestamp: 2025-07-06T15:24:00-07:00
package device

import (
	"fmt"
	"strconv"
	"strings"
)

// expandPortRanges is a helper to convert strings like "0-255,511" to a slice of strings.
func expandPortRanges(ranges []string) ([]string, error) {
	var ports []string
	for _, r := range ranges {
		parts := strings.Split(r, "-")
		if len(parts) == 1 {
			ports = append(ports, parts[0])
		} else if len(parts) == 2 {
			start, err1 := strconv.Atoi(parts[0])
			end, err2 := strconv.Atoi(parts[1])
			if err1 != nil || err2 != nil || start > end {
				return nil, fmt.Errorf("invalid range: %s", r)
			}
			for i := start; i <= end; i++ {
				ports = append(ports, strconv.Itoa(i))
			}
		} else {
			return nil, fmt.Errorf("invalid range format: %s", r)
		}
	}
	return ports, nil
}

