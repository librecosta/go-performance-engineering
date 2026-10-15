package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"

	retention "github.com/librecosta/go-performance-engineering/proofs/memory/slice-retention"
)

// This is the only root retaining the experiment's result. No goroutines or
// closures capture it; release it only after the retained profile is written.
var retainedRoot []byte

type heapStats struct {
	HeapAlloc   uint64 `json:"heap_alloc"`
	HeapInuse   uint64 `json:"heap_inuse"`
	HeapObjects uint64 `json:"heap_objects"`
}

type report struct {
	Mode           string            `json:"mode"`
	SourceBytes    int               `json:"source_bytes"`
	KeepBytes      int               `json:"keep_bytes"`
	Length         int               `json:"len"`
	Capacity       int               `json:"cap"`
	Checksum       uint64            `json:"checksum"`
	Baseline       heapStats         `json:"baseline"`
	Retained       heapStats         `json:"retained"`
	Released       heapStats         `json:"released"`
	Profile        string            `json:"profile"`
	GoVersion      string            `json:"go_version"`
	GOOS           string            `json:"goos"`
	GOARCH         string            `json:"goarch"`
	GOMAXPROCS     int               `json:"gomaxprocs"`
	NumCPU         int               `json:"num_cpu"`
	MemProfileRate int               `json:"mem_profile_rate"`
	Environment    map[string]string `json:"environment"`
	Command        []string          `json:"command"`
}

func main() {
	mode := flag.String("mode", "view", "view, cap, or copy (one mode per process)")
	sizeMiB := flag.Int("size-mib", 100, "source size in MiB (1..128)")
	keepBytes := flag.Int("keep-bytes", 100, "prefix size in bytes (1..source size)")
	profile := flag.String("profile", "", "retained heap profile path (optional; overwritten if present)")
	flag.Parse()
	if err := run(*mode, *sizeMiB, *keepBytes, *profile); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(mode string, sizeMiB, keepBytes int, profile string) error {
	if flag.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", flag.Args())
	}
	if mode != "view" && mode != "cap" && mode != "copy" {
		return fmt.Errorf("mode must be view, cap, or copy")
	}
	if sizeMiB < 1 || sizeMiB > 128 {
		return fmt.Errorf("size-mib must be between 1 and 128")
	}
	size := sizeMiB << 20
	if keepBytes < 1 || keepBytes > size {
		return fmt.Errorf("keep-bytes must be between 1 and %d", size)
	}
	r := report{
		Mode: mode, SourceBytes: size, KeepBytes: keepBytes, Profile: profile,
		GoVersion: runtime.Version(), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH,
		GOMAXPROCS: runtime.GOMAXPROCS(0), NumCPU: runtime.NumCPU(),
		MemProfileRate: runtime.MemProfileRate, Command: os.Args,
		Environment: make(map[string]string),
	}
	for _, key := range []string{"GOGC", "GOMEMLIMIT", "GOMAXPROCS", "GODEBUG", "GOEXPERIMENT", "GOFLAGS"} {
		r.Environment[key] = os.Getenv(key)
	}
	r.Baseline = snapshot()
	retainedRoot = makeResult(mode, size, keepBytes)
	r.Length, r.Capacity = len(retainedRoot), cap(retainedRoot)
	r.Retained = snapshot()
	// Read metrics before profiling: the profiler has its own allocation cost.
	if profile != "" {
		if err := writeProfile(profile); err != nil {
			return err
		}
	}
	for _, value := range retainedRoot {
		r.Checksum += uint64(value)
	}
	runtime.KeepAlive(retainedRoot)
	retainedRoot = nil
	r.Released = snapshot()
	// Formatting and output occur after all three measurements.
	return json.NewEncoder(os.Stdout).Encode(r)
}

// Keep the source's stack frame separate from measurement. Only the returned
// prefix survives this call; keeping src elsewhere would invalidate copy mode.
//
//go:noinline
func makeResult(mode string, size, keepBytes int) []byte {
	src := make([]byte, size)
	for i := range src {
		src[i] = byte(i*31 + 7)
	}
	switch mode {
	case "view":
		return retention.ViewPrefix(src, keepBytes)
	case "cap":
		return retention.LimitedViewPrefix(src, keepBytes)
	default: // run validates the mode before allocating.
		return retention.CopyPrefix(src, keepBytes)
	}
}

func snapshot() heapStats {
	// Memory profiles may lag by two GC cycles. Use the same two collections at
	// each phase, with no change to GC pacing, memory limits, or sampling rate.
	runtime.GC()
	runtime.GC()
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return heapStats{stats.HeapAlloc, stats.HeapInuse, stats.HeapObjects}
}

func writeProfile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create retained profile: %w", err)
	}
	writeErr := pprof.WriteHeapProfile(f)
	closeErr := f.Close()
	if writeErr != nil {
		return fmt.Errorf("write retained profile: %w", writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close retained profile: %w", closeErr)
	}
	return nil
}
