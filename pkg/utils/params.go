package utils

import (
	"errors"
	"strconv"
)

// ParseUintID parses a path/query ID string into uint (must be > 0).
func ParseUintID(s string) (uint, error) {
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil || id == 0 {
		return 0, errors.New("invalid id")
	}
	return uint(id), nil
}
