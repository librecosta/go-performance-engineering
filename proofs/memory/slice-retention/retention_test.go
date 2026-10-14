package retention

import (
	"bytes"
	"testing"
)

func TestPrefixOwnership(t *testing.T) {
	for _, tc := range []struct {
		name   string
		prefix func([]byte, int) []byte
		shares bool
		cap    int
	}{
		{"view", ViewPrefix, true, 256},
		{"cap", LimitedViewPrefix, true, 100},
		{"copy", CopyPrefix, false, 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := make([]byte, 256)
			for i := range src {
				src[i] = byte(i*31 + 7)
			}
			got := tc.prefix(src, 100)
			if len(got) != 100 || cap(got) != tc.cap || !bytes.Equal(got, src[:100]) {
				t.Fatalf("prefix mismatch: len=%d cap=%d", len(got), cap(got))
			}
			src[0] ^= 0xff
			if (got[0] == src[0]) != tc.shares {
				t.Fatal("source mutation has incorrect sharing behavior")
			}
			got[1] ^= 0xff
			if (got[1] == src[1]) != tc.shares {
				t.Fatal("result mutation has incorrect sharing behavior")
			}
		})
	}
}

func TestLimitedViewAppend(t *testing.T) {
	src := bytes.Repeat([]byte{7}, 256)
	limited := LimitedViewPrefix(src, 100)
	extended := append(limited, 42)
	extended[0] = 99
	if src[0] != 7 || src[100] != 7 || limited[0] != 7 || extended[100] != 42 {
		t.Fatal("append must allocate independent storage beyond limited capacity")
	}
	// The original limited view still aliases src, even after another slice grows.
	limited[0] = 13
	if src[0] != 13 {
		t.Fatal("capacity limiting unexpectedly detached the original view")
	}
}

func TestPrefixBounds(t *testing.T) {
	for _, prefix := range []func([]byte, int) []byte{ViewPrefix, LimitedViewPrefix, CopyPrefix} {
		if got := prefix(nil, 0); len(got) != 0 || cap(got) != 0 {
			t.Fatal("nil input with zero prefix must be empty")
		}
		src := []byte{1, 2, 3}
		if !bytes.Equal(prefix(src, len(src)), src) {
			t.Fatal("full prefix differs from source")
		}
		for _, n := range []int{-1, cap(src) + 1} {
			func() {
				defer func() {
					if recover() == nil {
						t.Errorf("prefix length %d must panic", n)
					}
				}()
				prefix(src, n)
			}()
		}
	}
}
