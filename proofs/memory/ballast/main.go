package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	var (
		ballastMiB = flag.Int64("ballast-mib", 0, "long-lived ballast size in MiB")
		churnMiB   = flag.Int64("churn-mib", 512, "short-lived allocation churn in MiB")
		blockKiB   = flag.Int("block-kib", 8, "allocation block size in KiB")
		batchSize  = flag.Int("batch", 64, "number of allocations kept live per batch")
		asJSON     = flag.Bool("json", false, "print machine-readable JSON")
	)
	flag.Parse()

	cfg := Config{
		BallastBytes: *ballastMiB * mib,
		ChurnBytes:   *churnMiB * mib,
		BlockBytes:   *blockKiB * int(kib),
		BatchSize:    *batchSize,
	}

	result, err := runExperiment(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ballast proof:", err)
		os.Exit(2)
	}

	if *asJSON {
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			fmt.Fprintln(os.Stderr, "ballast proof:", err)
			os.Exit(2)
		}
		return
	}

	printHumanResult(result)
}

func printHumanResult(r Result) {
	fmt.Printf("Go version:          %s\n", r.GoVersion)
	fmt.Printf("Ballast:             %s\n", formatBytes(r.BallastBytes))
	fmt.Printf("Allocation churn:    %s\n", formatBytes(r.ChurnBytes))
	fmt.Printf("Allocation block:    %s\n", formatBytes(r.BlockBytes))
	fmt.Printf("Batch size:          %d\n", r.BatchSize)
	fmt.Println()
	fmt.Printf("Heap after setup GC: %s\n", formatBytes(r.HeapAllocAfterGC))
	fmt.Printf("Next GC goal:        %s\n", formatBytes(r.NextGC))
	fmt.Printf("GC cycles in churn:  %d\n", r.GCCycles)
	fmt.Printf("GC pause in churn:   %s\n", time.Duration(r.PauseTotalNs))
	fmt.Printf("Allocated in churn:  %s\n", formatBytes(r.TotalAllocated))
	fmt.Printf("Elapsed:              %s\n", time.Duration(r.ElapsedNs))
	fmt.Printf("Checksum:             %d\n", r.Checksum)
}

func formatBytes(n uint64) string {
	const (
		kiB = uint64(1 << 10)
		miB = uint64(1 << 20)
		giB = uint64(1 << 30)
	)

	switch {
	case n >= giB:
		return fmt.Sprintf("%.2f GiB", float64(n)/float64(giB))
	case n >= miB:
		return fmt.Sprintf("%.2f MiB", float64(n)/float64(miB))
	case n >= kiB:
		return fmt.Sprintf("%.2f KiB", float64(n)/float64(kiB))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
