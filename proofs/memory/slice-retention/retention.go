// Package retention demonstrates the storage ownership of slice prefixes.
package retention

// ViewPrefix returns a prefix sharing src's backing array.
// As with ordinary slicing, n must be between zero and cap(src).
func ViewPrefix(src []byte, n int) []byte {
	return src[:n]
}

// LimitedViewPrefix limits append capacity, but still shares the backing array.
// Reducing capacity does not detach the prefix or release the rest of the array.
func LimitedViewPrefix(src []byte, n int) []byte {
	return src[:n:n]
}

// CopyPrefix returns independent storage for the prefix.
// The intentional copy allows the source to be collected when no other owner
// retains it. Replacing this with a view would restore the retention mechanism.
func CopyPrefix(src []byte, n int) []byte {
	prefix := src[:n]
	dst := make([]byte, n)
	copy(dst, prefix)
	return dst
}
