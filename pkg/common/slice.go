package common

import "reflect"

// sliceDiff returns the difference between two slices as a slice
func SliceDiff[T any](s1 []T, s2 []T) []T {
	var diff []T
	// Loop two times, first to find s1 strings not in s2,
	// second loop to find s2 strings not in s2
	for i := 0; i < 2; i++ {
		for _, s1 := range s1 {
			found := false
			for _, s2 := range s2 {
				if reflect.DeepEqual(s1, s2) {
					found = true
					break
				}
			}
			// String not found. We add it to return slice
			if !found {
				diff = append(diff, s1)
			}
		}
		// Swap the slices, only if it was the first loop
		if i == 0 {
			s1, s2 = s2, s1
		}
	}
	if len(diff) == 0 {
		return nil
	}
	return diff
}

// sliceEqual returns true if the two slices have equal values, false otherwise.
func SliceEqual[T any](s1 []T, s2 []T) bool {
	if len(s1) != len(s2) {
		return false
	}
	diff := SliceDiff(s1, s2)
	if len(diff) == 0 {
		return true
	}
	return false
}
