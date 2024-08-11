package forge

// Arrange inserts/removes a element 'el' at index 'idx'.
//
// It will perform equality check with 'key(el)'.
// When key function is nil, it cannot compare elements.
// So 'elems' will be returned as untouched.
//
// If a key for 'el' already exists, it will override the value
// if user set 'override' to true.
// When it overrides, it will not touch the original index.
//
// idx < 0 means remove the element.
// idx >= len(elems) means append it to the last.
//
// ex) When elems is []string{"a", "b", "c"} with given arguments,
// results will be same as following.
//
//	elems = []string{"b", "c"}       where path = "a" and idx = -1
//	elems = []string{"a", "b", "c"}  where path = "a" and idx = 0
//	elems = []string{"b", "a", "c"}  where path = "a" and idx = 1
//	elems = []string{"b", "c", "a"}  where path = "a" and idx = 2
func Arrange[T any, K comparable](elems []T, el T, idx int, key func(a T) K, override bool) []T {
	if key == nil {
		return elems
	}
	findIdx := -1
	for i, e := range elems {
		if key(e) == key(el) {
			findIdx = i
			break
		}
	}
	if findIdx >= 0 {
		if override {
			// keep original index
			elems[findIdx] = el
			return elems
		}
		el = elems[findIdx]
		elems = append(elems[:findIdx], elems[findIdx+1:]...)
	}
	if idx < 0 {
		// remove
		return elems
	}
	if idx < len(elems) {
		// insert el at idx
		var t T
		elems = append(elems, t)
		copy(elems[idx+1:], elems[idx:])
		elems[idx] = el
		return elems
	}
	// append
	elems = append(elems, el)
	return elems
}
