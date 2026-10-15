// Package bce compares bounds proofs for four-byte little-endian reads.
package bce

// DecodeGuarded returns the first four bytes as a little-endian value, or
// (0, false) for short input. The length guard remains necessary even when
// subsequent index checks are eliminated.
//
//go:noinline
func DecodeGuarded(b []byte) (uint32, bool) {
	if len(b) < 4 {
		return 0, false
	}
	v := uint32(b[0]) |
		uint32(b[1])<<8 |
		uint32(b[2])<<16 |
		uint32(b[3])<<24
	return v, true
}

// DecodeForward reads indices from low to high and panics on short input.
// All three functions stay out of line to isolate their own bounds proofs
// from caller-specific length information.
//
//go:noinline
func DecodeForward(b []byte) uint32 {
	a0 := b[0]
	a1 := b[1]
	a2 := b[2]
	a3 := b[3]
	return uint32(a0) | uint32(a1)<<8 |
		uint32(a2)<<16 | uint32(a3)<<24
}

// DecodeHighFirst reads the highest index first, establishing a length bound
// for the remaining reads. It panics on short input, potentially at a different
// index than DecodeForward; the read order is the mechanism under study.
//
//go:noinline
func DecodeHighFirst(b []byte) uint32 {
	a3 := b[3]
	a2 := b[2]
	a1 := b[1]
	a0 := b[0]
	return uint32(a0) | uint32(a1)<<8 |
		uint32(a2)<<16 | uint32(a3)<<24
}
