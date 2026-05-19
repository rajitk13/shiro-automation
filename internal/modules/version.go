package modules

import (
	"fmt"
	"strconv"
	"strings"
)

// Version represents a semantic version
type Version struct {
	Major int
	Minor int
	Patch int
}

// ParseVersion parses a semantic version string
func ParseVersion(version string) (Version, error) {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid version format: %s", version)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Version{}, fmt.Errorf("invalid patch version: %s", parts[2])
	}

	return Version{Major: major, Minor: minor, Patch: patch}, nil
}

// String returns the string representation of the version
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Compare compares two versions
// Returns -1 if v < other, 0 if v == other, 1 if v > other
func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}

	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}

	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}

	return 0
}

// IsGreaterThan checks if v is greater than other
func (v Version) IsGreaterThan(other Version) bool {
	return v.Compare(other) == 1
}

// IsLessThan checks if v is less than other
func (v Version) IsLessThan(other Version) bool {
	return v.Compare(other) == -1
}

// IsEqual checks if v is equal to other
func (v Version) IsEqual(other Version) bool {
	return v.Compare(other) == 0
}
