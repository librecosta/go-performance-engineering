package retention

import (
	"flag"
	"runtime"
	"testing"
)

var benchmarkSizeMiB = flag.Int("bench-size-mib", 100, "source MiB (1..128); use 8 only for smoke checks")

// Identical sinks make every result escape and consume its contents. These
// benchmarks run sequentially; sink storage is cleared outside each timer.
var prefixSink []byte
var byteSink byte

func BenchmarkPrefix(b *testing.B) {
	if *benchmarkSizeMiB < 1 || *benchmarkSizeMiB > 128 {
		b.Fatal("bench-size-mib must be between 1 and 128")
	}
	// One shared input, allocated and initialized outside every timed loop.
	src := make([]byte, *benchmarkSizeMiB<<20)
	for i := range src {
		src[i] = byte(i*31 + 7)
	}
	b.Run("View", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			prefixSink = ViewPrefix(src, 100)
			byteSink = prefixSink[0] ^ prefixSink[99]
		}
		prefixSink = nil
	})
	b.Run("LimitedView", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			prefixSink = LimitedViewPrefix(src, 100)
			byteSink = prefixSink[0] ^ prefixSink[99]
		}
		prefixSink = nil
	})
	b.Run("Copy", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			prefixSink = CopyPrefix(src, 100)
			byteSink = prefixSink[0] ^ prefixSink[99]
		}
		prefixSink = nil
	})
	runtime.KeepAlive(src)
}
