package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"
)

const (
	helperModeEnv    = "GPE_BALLAST_HELPER"
	helperBallastEnv = "GPE_BALLAST_BYTES"
	helperChurnEnv   = "GPE_CHURN_BYTES"
	helperBlockEnv   = "GPE_BLOCK_BYTES"
	helperBatchEnv   = "GPE_BATCH_SIZE"
)

const (
	testBallastBytes = 64 * mib
	testChurnBytes   = 256 * mib
	testBlockBytes   = 8 * kib
	testBatchSize    = 64
)

// TestBallastMechanism validates mechanism direction, not wall-clock speed.
//
// The baseline and ballast cases run in separate processes so GC history,
// spans, scavenger state, and allocator reuse from one case do not contaminate
// the other.
//
// We intentionally do NOT assert ElapsedNs or PauseTotalNs because shared CI
// runners are too noisy for reliable timing thresholds. The proof's stable
// contract is that, under GOGC-based pacing, a live ballast raises the heap
// goal and materially reduces GC cycle frequency for the same churn workload.
func TestBallastMechanism(t *testing.T) {
	baseline := runProofChild(t, 0)
	withBallast := runProofChild(t, testBallastBytes)

	t.Logf(
		"baseline: heap=%s goal=%s gc=%d elapsed=%s",
		formatBytes(baseline.HeapAllocAfterGC),
		formatBytes(baseline.NextGC),
		baseline.GCCycles,
		formatDuration(baseline.ElapsedNs),
	)
	t.Logf(
		"ballast:  heap=%s goal=%s gc=%d elapsed=%s",
		formatBytes(withBallast.HeapAllocAfterGC),
		formatBytes(withBallast.NextGC),
		withBallast.GCCycles,
		formatDuration(withBallast.ElapsedNs),
	)

	const minimumVisibleBallast = 48 * mib

	if withBallast.HeapAllocAfterGC < baseline.HeapAllocAfterGC+uint64(minimumVisibleBallast) {
		t.Fatalf(
			"ballast did not materially increase live heap: baseline=%s ballast=%s",
			formatBytes(baseline.HeapAllocAfterGC),
			formatBytes(withBallast.HeapAllocAfterGC),
		)
	}

	if withBallast.NextGC < baseline.NextGC+uint64(minimumVisibleBallast) {
		t.Fatalf(
			"ballast did not materially increase next GC goal: baseline=%s ballast=%s",
			formatBytes(baseline.NextGC),
			formatBytes(withBallast.NextGC),
		)
	}

	if baseline.GCCycles < 4 {
		t.Fatalf(
			"baseline workload did not exercise GC enough: got %d cycles",
			baseline.GCCycles,
		)
	}

	if withBallast.GCCycles >= baseline.GCCycles {
		t.Fatalf(
			"expected fewer GC cycles with ballast: baseline=%d ballast=%d",
			baseline.GCCycles,
			withBallast.GCCycles,
		)
	}

	// The chosen test parameters intentionally create a large signal so this
	// proof does not depend on a fragile few-percent CI threshold.
	if uint64(withBallast.GCCycles)*2 > uint64(baseline.GCCycles) {
		t.Fatalf(
			"ballast effect was too small for a robust proof: baseline=%d ballast=%d",
			baseline.GCCycles,
			withBallast.GCCycles,
		)
	}

	if baseline.TotalAllocated < uint64(testChurnBytes) {
		t.Fatalf(
			"baseline did not allocate the requested churn: got=%s want-at-least=%s",
			formatBytes(baseline.TotalAllocated),
			formatBytes(uint64(testChurnBytes)),
		)
	}

	if withBallast.TotalAllocated < uint64(testChurnBytes) {
		t.Fatalf(
			"ballast case did not allocate the requested churn: got=%s want-at-least=%s",
			formatBytes(withBallast.TotalAllocated),
			formatBytes(uint64(testChurnBytes)),
		)
	}
}

// TestBallastHelperProcess is executed only by TestBallastMechanism in a child
// copy of the test binary. It prints exactly one JSON Result and exits before
// the testing harness can append PASS output.
func TestBallastHelperProcess(t *testing.T) {
	if os.Getenv(helperModeEnv) != "1" {
		t.Skip("helper process")
	}

	cfg := Config{
		BallastBytes: mustParseEnvInt64(helperBallastEnv),
		ChurnBytes:   mustParseEnvInt64(helperChurnEnv),
		BlockBytes:   int(mustParseEnvInt64(helperBlockEnv)),
		BatchSize:    int(mustParseEnvInt64(helperBatchEnv)),
	}

	result, err := runExperiment(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	os.Exit(0)
}

func runProofChild(t *testing.T, ballastBytes int64) Result {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=^TestBallastHelperProcess$")
	cmd.Env = append(
		os.Environ(),
		helperModeEnv+"=1",
		helperBallastEnv+"="+strconv.FormatInt(ballastBytes, 10),
		helperChurnEnv+"="+strconv.FormatInt(testChurnBytes, 10),
		helperBlockEnv+"="+strconv.FormatInt(testBlockBytes, 10),
		helperBatchEnv+"="+strconv.Itoa(testBatchSize),
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("child proof failed: %v\n%s", err, out)
	}

	var result Result
	if err := json.Unmarshal(bytes.TrimSpace(out), &result); err != nil {
		t.Fatalf("decode child result: %v\noutput: %q", err, out)
	}
	return result
}

func mustParseEnvInt64(name string) int64 {
	value := os.Getenv(name)
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s=%q: %v\n", name, value, err)
		os.Exit(2)
	}
	return n
}

func formatDuration(ns int64) string {
	return fmt.Sprintf("%.2fms", float64(ns)/1e6)
}
