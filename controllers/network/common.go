package network

var (
	ignoredSecretEntries = []string{".pub", "nats-sidecar", "edgefarm-sys"}
)

const (
	participatingNodeStatePending     = "pending"
	participatingNodeStateActive      = "active"
	participatingNodeStateTerminating = "terminating"
)

// empty struct (0 bytes)
type void struct{}

// missingInSlice returns the values in the first slice that are not in the second slice.
func missingInSlice(a []string, b []string) []string {
	// create map with length of the 'a' slice
	ma := make(map[string]void, len(a))
	diffs := []string{}
	// Convert first slice to map with empty struct (0 bytes)
	for _, ka := range a {
		ma[ka] = void{}
	}
	// find missing values in a
	for _, kb := range b {
		if _, ok := ma[kb]; !ok {
			diffs = append(diffs, kb)
		}
	}
	return diffs
}

func sliceDiffMissingNew(a []string, b []string) ([]string, []string) {
	new := missingInSlice(a, b)
	missing := missingInSlice(b, a)
	return new, missing
}

// // sliceDiff returns the difference between two slices as a slice
// func sliceDiff(s1 []string, s2 []string) []string {
// 	var diff []string
// 	// Loop two times, first to find s1 strings not in s2,
// 	// second loop to find s2 strings not in s2
// 	for i := 0; i < 2; i++ {
// 		for _, s1 := range s1 {
// 			found := false
// 			for _, s2 := range s2 {
// 				if s1 == s2 {
// 					found = true
// 					break
// 				}
// 			}
// 			// String not found. We add it to return slice
// 			if !found {
// 				diff = append(diff, s1)
// 			}
// 		}
// 		// Swap the slices, only if it was the first loop
// 		if i == 0 {
// 			s1, s2 = s2, s1
// 		}
// 	}
// 	if len(diff) == 0 {
// 		return nil
// 	}
// 	return diff
// }

// // sliceEqual returns true if the two slices have equal values, false otherwise.
// func sliceEqual(s1 []string, s2 []string) bool {
// 	if len(s1) != len(s2) {
// 		return false
// 	}
// 	diff := sliceDiff(s1, s2)
// 	if len(diff) == 0 {
// 		return true
// 	}
// 	return false
// }
