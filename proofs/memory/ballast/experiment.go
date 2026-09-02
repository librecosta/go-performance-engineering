package main

import (
	"fmt"
	"math"
	"runtime"
	"runtime/debug"
	"time"
)

const (
	mib = int64(1 << 20)
	kib = int64(1 << 10)
)

// Config defines one isolated ballast experiment.
//
// The experiment deliberately fixes GOGC at 100 and disables the runtime
// memory limit so that the observed behavior is driven by GOGC-based pacing.
type Config struct {
	BallastBytes int64
	ChurnBytes   int64
	BlockBytes   int
	BatchSize    int
}

// Result records mechanism evidence and observational timing data.
//
// HeapAllocAfterGC and NextGC are sampled immediately after a forced GC and
// before the allocation-churn workload starts. GCCycles and PauseTotalNs cover
// only the churn phase.
type Result struct {
	GoVersion        string `json:"go_version"`
	BallastBytes     uint64 `json:"ballast_bytes"`
	ChurnBytes       uint64 `json:"churn_bytes"`
	BlockBytes       uint64 `json:"block_bytes"`
	BatchSize        int    `json:"batch_size"`
	HeapAllocAfterGC uint64 `json:"heap_alloc_after_gc"`
	NextGC           uint64 `json:"next_gc"`
	GCCycles         uint32 `json:"gc_cycles"`
	PauseTotalNs     uint64 `json:"pause_total_ns"`
	TotalAllocated   uint64 `json:"total_allocated"`
	ElapsedNs        int64  `json:"elapsed_ns"`
	Checksum         uint64 `json:"checksum"`
}

func defaultConfig(ballastBytes int64) Config {
	return Config{
		BallastBytes: ballastBytes,
		ChurnBytes:   512 * mib,
		BlockBytes:   int(8 * kib),
		BatchSize:    64,
	}
}

func validateConfig(cfg Config) error {
	switch {
	case cfg.BallastBytes < 0:
		return fmt.Errorf("ballast bytes must be >= 0")
	case cfg.ChurnBytes <= 0:
		return fmt.Errorf("churn bytes must be > 0")
	case cfg.BlockBytes <= 0:
		return fmt.Errorf("block bytes must be > 0")
	case cfg.BatchSize <= 0:
		return fmt.Errorf("batch size must be > 0")
	}
	return nil
}

func runExperiment(cfg Config) (Result, error) {
	if err := validateConfig(cfg); err != nil {
		return Result{}, err
	}

	// This proof isolates the historical GOGC-based ballast mechanism.
	// A binding GOMEMLIMIT would intentionally change the pacer's behavior and
	// would make the experiment answer a different question.
	oldGCPercent := debug.SetGCPercent(100)
	defer debug.SetGCPercent(oldGCPercent)

	oldMemoryLimit := debug.SetMemoryLimit(math.MaxInt64)
	defer debug.SetMemoryLimit(oldMemoryLimit)

	// Remove as much previous process state as practical before establishing
	// the experiment's live heap.
	runtime.GC()

	var ballast []byte
	if cfg.BallastBytes > 0 {
		ballast = make([]byte, int(cfg.BallastBytes))
	}

	// Force one collection so HeapAlloc and NextGC reflect a heap in which the
	// ballast is live before the churn workload begins.
	runtime.GC()

	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	start := time.Now()
	checksum := runAllocationChurn(cfg.ChurnBytes, cfg.BlockBytes, cfg.BatchSize)
	elapsed := time.Since(start)

	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	// The ballast exists only to remain reachable throughout the churn phase.
	// KeepAlive makes that lifetime requirement explicit instead of relying on
	// lexical scope or current compiler liveness behavior.
	runtime.KeepAlive(ballast)

	return Result{
		GoVersion:        runtime.Version(),
		BallastBytes:     uint64(cfg.BallastBytes),
		ChurnBytes:       uint64(cfg.ChurnBytes),
		BlockBytes:       uint64(cfg.BlockBytes),
		BatchSize:        cfg.BatchSize,
		HeapAllocAfterGC: before.HeapAlloc,
		NextGC:           before.NextGC,
		GCCycles:         after.NumGC - before.NumGC,
		PauseTotalNs:     after.PauseTotalNs - before.PauseTotalNs,
		TotalAllocated:   after.TotalAlloc - before.TotalAlloc,
		ElapsedNs:        elapsed.Nanoseconds(),
		Checksum:         checksum,
	}, nil
}

// runAllocationChurn creates a fixed amount of short-lived heap allocation.
//
// Objects remain reachable in small batches and are then released. This keeps
// the live working set small while producing enough allocation traffic for the
// GC pacer to react. The checksum makes the object contents observably used.
func runAllocationChurn(totalBytes int64, blockBytes, batchSize int) uint64 {
	batch := make([][]byte, batchSize)
	var (
		remaining = totalBytes
		checksum  uint64
		sequence  uint64
	)

	for remaining > 0 {
		n := batchSize

		for i := 0; i < n && remaining > 0; i++ {
			size := blockBytes
			if int64(size) > remaining {
				size = int(remaining)
			}

			buf := make([]byte, size)
			buf[0] = byte(sequence)
			buf[len(buf)-1] ^= byte(size)
			batch[i] = buf

			remaining -= int64(size)
			sequence++

			if remaining == 0 {
				n = i + 1
				break
			}
		}

		for i := 0; i < n; i++ {
			buf := batch[i]
			checksum += uint64(buf[0])
			checksum += uint64(buf[len(buf)-1])
			batch[i] = nil
		}
	}

	runtime.KeepAlive(batch)
	return checksum
}
