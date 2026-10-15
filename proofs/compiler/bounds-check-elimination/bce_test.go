package bce

import "testing"

func TestDecodeValid(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
		want uint32
	}{
		{"known", []byte{0x12, 0x34, 0x56, 0x78}, 0x78563412},
		{"trailing bytes", []byte{0x12, 0x34, 0x56, 0x78, 0xff, 0x00}, 0x78563412},
		{"zero", []byte{0, 0, 0, 0}, 0},
		{"all bits", []byte{0xff, 0xff, 0xff, 0xff}, 0xffffffff},
		{"low byte", []byte{0x80, 0, 0, 0}, 0x00000080},
		{"high byte", []byte{0, 0, 0, 0x80}, 0x80000000},
		{"alternating", []byte{0xaa, 0x55, 0xaa, 0x55}, 0x55aa55aa},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Explicit expected values are the oracle, not another decoder.
			if got, ok := DecodeGuarded(tc.data); !ok || got != tc.want {
				t.Errorf("Guarded = (%#08x, %t), want (%#08x, true)", got, ok, tc.want)
			}
			if got := DecodeForward(tc.data); got != tc.want {
				t.Errorf("Forward = %#08x, want %#08x", got, tc.want)
			}
			if got := DecodeHighFirst(tc.data); got != tc.want {
				t.Errorf("HighFirst = %#08x, want %#08x", got, tc.want)
			}
		})
	}
}

func TestDecodeShort(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"nil", nil},
		{"empty", []byte{}},
		{"one", []byte{0x12}},
		{"two", []byte{0x12, 0x34}},
		{"three", []byte{0x12, 0x34, 0x56}},
		{"spare capacity", make([]byte, 3, 8)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, ok := DecodeGuarded(tc.data); ok || got != 0 {
				t.Errorf("Guarded = (%#08x, %t), want (0, false)", got, ok)
			}
			for _, decoder := range []struct {
				name string
				fn   func([]byte) uint32
			}{
				{"Forward", DecodeForward},
				{"HighFirst", DecodeHighFirst},
			} {
				t.Run(decoder.name, func(t *testing.T) {
					defer func() {
						if recover() == nil {
							t.Error("short input must panic")
						}
					}()
					decoder.fn(tc.data)
				})
			}
		})
	}
}

var benchmarkSink uint64

func BenchmarkDecodeForward(b *testing.B) {
	data := []byte{0x12, 0x34, 0x56, 0x78}
	var sum uint64
	b.ReportAllocs()
	for b.Loop() {
		sum += uint64(DecodeForward(data))
	}
	benchmarkSink = sum
}

func BenchmarkDecodeHighFirst(b *testing.B) {
	data := []byte{0x12, 0x34, 0x56, 0x78}
	var sum uint64
	b.ReportAllocs()
	for b.Loop() {
		sum += uint64(DecodeHighFirst(data))
	}
	benchmarkSink = sum
}
