package falsesharing

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"
)

const (
	lineBytes      = 64 // Experimental precondition, not hardware detection.
	storageEntries = 23
	windowEntries  = 16
	packed         = "Packed"
	separated      = "Separated"
)

type counterLayout struct {
	storage []atomic.Uint64
	start   int
	a, b    *atomic.Uint64
}

func newLayout(kind string) (counterLayout, error) {
	return selectLayout(make([]atomic.Uint64, storageEntries), kind)
}

func selectLayout(storage []atomic.Uint64, kind string) (counterLayout, error) {
	if runtime.GOARCH != "amd64" {
		return counterLayout{}, fmt.Errorf("requires amd64; got %s", runtime.GOARCH)
	}
	bOffset := 1
	switch kind {
	case packed:
	case separated:
		bOffset = 8
	default:
		return counterLayout{}, fmt.Errorf("unknown layout %q", kind)
	}
	if len(storage) != storageEntries || unsafe.Sizeof(atomic.Uint64{}) != 8 {
		return counterLayout{}, fmt.Errorf("requires 23 elements of size 8 bytes")
	}
	// Integer addresses are used only for inspection, never converted back to
	// pointers. All accesses below use the original typed slice.
	for i := 1; i < len(storage); i++ {
		previous := uintptr(unsafe.Pointer(&storage[i-1]))
		current := uintptr(unsafe.Pointer(&storage[i]))
		if current-previous != 8 {
			return counterLayout{}, fmt.Errorf("element %d does not have an 8-byte stride", i)
		}
	}
	start := -1
	for i := range storage {
		if uintptr(unsafe.Pointer(&storage[i]))%lineBytes == 0 {
			start = i
			break
		}
	}
	if start < 0 || start+windowEntries > len(storage) || bOffset >= windowEntries {
		return counterLayout{}, fmt.Errorf("no complete aligned 16-element window")
	}
	a, b := &storage[start], &storage[start+bOffset]
	aAddr, bAddr := uintptr(unsafe.Pointer(a)), uintptr(unsafe.Pointer(b))
	lastAddr := uintptr(unsafe.Pointer(&storage[start+windowEntries-1]))
	if aAddr%lineBytes != 0 || lastAddr-aAddr+8 != 128 {
		return counterLayout{}, fmt.Errorf("window must be 64-byte aligned and span 128 bytes")
	}
	if bAddr-aAddr != uintptr(bOffset*8) || (aAddr/lineBytes == bAddr/lineBytes) != (kind == packed) {
		return counterLayout{}, fmt.Errorf("counter offsets or cache-line relationship do not match %s", kind)
	}
	return counterLayout{storage: storage, start: start, a: a, b: b}, nil
}

// Both layouts use exactly this hot loop; synchronization is outside it.
func update(counter *atomic.Uint64, count int) {
	for i := 0; i < count; i++ {
		counter.Add(1)
	}
}

type workerPair struct {
	start chan struct{}
	done  sync.WaitGroup
	want  [2]int
}

func prepareWorkers(layout counterLayout, total int) (*workerPair, error) {
	if total <= 0 {
		return nil, fmt.Errorf("total updates must be positive")
	}
	if layout.a == nil || layout.b == nil || layout.a == layout.b {
		return nil, fmt.Errorf("requires two distinct, non-nil counters")
	}
	if layout.a.Load() != 0 || layout.b.Load() != 0 {
		return nil, fmt.Errorf("both counters must start at zero")
	}
	pair := &workerPair{
		start: make(chan struct{}),
		want:  [2]int{total / 2, total - total/2},
	}
	var ready sync.WaitGroup
	ready.Add(2)
	pair.done.Add(2)
	for i, counter := range [2]*atomic.Uint64{layout.a, layout.b} {
		go func(counter *atomic.Uint64, count int) {
			ready.Done()
			<-pair.start
			update(counter, count)
			pair.done.Done()
		}(counter, pair.want[i])
	}
	// Both workers have announced readiness at the gate before timing begins.
	ready.Wait()
	return pair, nil
}

func checkCounts(layout counterLayout, want [2]int) error {
	a, b := layout.a.Load(), layout.b.Load()
	if a != uint64(want[0]) || b != uint64(want[1]) {
		return fmt.Errorf("counts = (%d, %d), want (%d, %d)", a, b, want[0], want[1])
	}
	return nil
}

func checkBenchmarkProcs(p int) error {
	if p != 1 && p != 2 {
		return fmt.Errorf("requires externally selected GOMAXPROCS=1 or 2; got %d", p)
	}
	return nil
}

func benchmarkFalseSharing(b *testing.B, kind string) {
	b.StopTimer()
	if runtime.GOARCH != "amd64" {
		b.Skipf("64-byte layout experiment supports amd64 only; got %s", runtime.GOARCH)
	}
	p := runtime.GOMAXPROCS(0)
	if err := checkBenchmarkProcs(p); err != nil {
		b.Fatal(err)
	}
	layout, err := newLayout(kind)
	if err != nil {
		b.Fatal(err)
	}
	pair, err := prepareWorkers(layout, b.N)
	if err != nil {
		b.Fatal(err)
	}
	// Reset also discards the benchmark harness's initial timed entry overhead.
	// b.N is TOTAL updates, including the odd-N calibration split.
	b.ResetTimer()
	b.StartTimer()
	close(pair.start)
	pair.done.Wait()
	b.StopTimer()
	err = checkCounts(layout, pair.want)
	runtime.KeepAlive(layout.storage)
	if err != nil {
		b.Fatal(err)
	}
	b.Logf("layout=%s GOMAXPROCS=%d total=%d split=%d/%d; addresses checked assuming 64-byte lines",
		kind, p, b.N, pair.want[0], pair.want[1])
}

func BenchmarkFalseSharingPacked(b *testing.B) {
	benchmarkFalseSharing(b, packed)
}

func BenchmarkFalseSharingSeparated(b *testing.B) {
	benchmarkFalseSharing(b, separated)
}

func TestFalseSharingLayout(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("layout precondition: amd64 only")
	}
	for _, kind := range []string{packed, separated} {
		t.Run(kind, func(t *testing.T) {
			layout, err := newLayout(kind)
			if err != nil {
				t.Fatal(err)
			}
			if len(layout.storage) != 23 || layout.start < 0 || layout.start > 7 || layout.start+16 > len(layout.storage) {
				t.Fatal("invalid storage or aligned window bounds")
			}
			for i := 0; i < layout.start; i++ {
				if uintptr(unsafe.Pointer(&layout.storage[i]))%64 == 0 {
					t.Fatal("did not select the first aligned element")
				}
			}
			a, b := uintptr(unsafe.Pointer(layout.a)), uintptr(unsafe.Pointer(layout.b))
			wantDistance := uintptr(8)
			if kind == separated {
				wantDistance = 64
			}
			if layout.a != &layout.storage[layout.start] || a%64 != 0 || b-a != wantDistance || (a/64 == b/64) != (kind == packed) {
				t.Fatal("incorrect actual counter addresses")
			}
			if err := checkCounts(layout, [2]int{0, 0}); err != nil {
				t.Fatal(err)
			}
			runtime.KeepAlive(layout.storage)
		})
	}
}

func TestFalseSharingUpdates(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("layout precondition: amd64 only")
	}
	for _, kind := range []string{packed, separated} {
		for _, tc := range []struct {
			total int
			want  [2]int
		}{{1, [2]int{0, 1}}, {2, [2]int{1, 1}}, {1000, [2]int{500, 500}}, {1001, [2]int{500, 501}}} {
			t.Run(fmt.Sprintf("%s/%d", kind, tc.total), func(t *testing.T) {
				layout, err := newLayout(kind)
				if err != nil {
					t.Fatal(err)
				}
				pair, err := prepareWorkers(layout, tc.total)
				if err != nil {
					t.Fatal(err)
				}
				// Readiness must not allow any updates before the common release.
				before := checkCounts(layout, [2]int{0, 0})
				close(pair.start)
				pair.done.Wait()
				if before != nil {
					t.Error(before)
				}
				if err := checkCounts(layout, tc.want); err != nil {
					t.Error(err)
				}
				runtime.KeepAlive(layout.storage)
			})
		}
	}
}

func TestFalseSharingPreconditions(t *testing.T) {
	for _, p := range []int{0, 1, 2, 3, 16} {
		if valid := checkBenchmarkProcs(p) == nil; valid != (p == 1 || p == 2) {
			t.Errorf("incorrect benchmark acceptance for GOMAXPROCS=%d", p)
		}
	}
	if runtime.GOARCH != "amd64" {
		if _, err := newLayout(packed); err == nil {
			t.Fatal("unsupported architecture was accepted")
		}
		return
	}
	if _, err := newLayout("unknown"); err == nil {
		t.Fatal("unknown layout was accepted")
	}
	if _, err := selectLayout(make([]atomic.Uint64, 22), packed); err == nil {
		t.Fatal("incorrect storage length was accepted")
	}
	layout, err := newLayout(packed)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []int{-1, 0} {
		if _, err := prepareWorkers(layout, n); err == nil {
			t.Fatalf("invalid update count %d was accepted", n)
		}
	}
	for _, invalid := range []counterLayout{{}, {a: layout.a}, {a: layout.a, b: layout.a}} {
		if _, err := prepareWorkers(invalid, 2); err == nil {
			t.Fatal("missing or shared counter was accepted")
		}
	}
	layout.a.Store(1)
	if _, err := prepareWorkers(layout, 2); err == nil {
		t.Fatal("nonzero initial counter was accepted")
	}
}
