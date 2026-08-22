// Command atlas-bench exercises a real Atlas HTTP path and emits machine-readable
// latency/throughput metrics JSON (load/bench product path for M5).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics is the machine-readable bench report.
type Metrics struct {
	Target       string  `json:"target"`
	Path         string  `json:"path"`
	Requests     int     `json:"requests"`
	Concurrency  int     `json:"concurrency"`
	Successes    int64   `json:"successes"`
	Failures     int64   `json:"failures"`
	DurationSec  float64 `json:"duration_sec"`
	ThroughputRPS float64 `json:"throughput_rps"`
	LatencyP50MS float64 `json:"latency_p50_ms"`
	LatencyP95MS float64 `json:"latency_p95_ms"`
	LatencyP99MS float64 `json:"latency_p99_ms"`
	ErrorRatio   float64 `json:"error_ratio"`
	GeneratedAt  string  `json:"generated_at"`
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("atlas-bench", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	target := fs.String("url", "http://127.0.0.1:8080", "base URL of gateway or orchestrator")
	path := fs.String("path", "/healthz", "path to hit (real service path)")
	n := fs.Int("n", 50, "total requests")
	c := fs.Int("c", 5, "concurrency")
	out := fs.String("out", "", "optional metrics JSON file (also printed to stdout)")
	timeout := fs.Duration("timeout", 5*time.Second, "per-request timeout")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *n < 1 {
		*n = 1
	}
	if *c < 1 {
		*c = 1
	}
	client := &http.Client{Timeout: *timeout}
	url := strings.TrimRight(*target, "/") + *path

	var (
		successes, failures int64
		latMu               sync.Mutex
		latencies           []float64
		wg                  sync.WaitGroup
		sem                 = make(chan struct{}, *c)
	)
	start := time.Now()
	for i := 0; i < *n; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			t0 := time.Now()
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				atomic.AddInt64(&failures, 1)
				return
			}
			resp, err := client.Do(req)
			elapsed := float64(time.Since(t0).Microseconds()) / 1000.0
			latMu.Lock()
			latencies = append(latencies, elapsed)
			latMu.Unlock()
			if err != nil {
				atomic.AddInt64(&failures, 1)
				return
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				atomic.AddInt64(&successes, 1)
			} else {
				atomic.AddInt64(&failures, 1)
			}
		}()
	}
	wg.Wait()
	dur := time.Since(start).Seconds()
	if dur <= 0 {
		dur = 0.001
	}
	sort.Float64s(latencies)
	m := Metrics{
		Target:        *target,
		Path:          *path,
		Requests:      *n,
		Concurrency:   *c,
		Successes:     successes,
		Failures:      failures,
		DurationSec:   dur,
		ThroughputRPS: float64(*n) / dur,
		LatencyP50MS:  percentile(latencies, 50),
		LatencyP95MS:  percentile(latencies, 95),
		LatencyP99MS:  percentile(latencies, 99),
		ErrorRatio:    float64(failures) / float64(*n),
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	raw, _ := json.MarshalIndent(m, "", "  ")
	fmt.Println(string(raw))
	if *out != "" {
		if err := os.WriteFile(*out, raw, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "write:", err)
			return 1
		}
	}
	// Fail if every request failed (service completely down) — still emitted metrics.
	if successes == 0 {
		return 1
	}
	return 0
}

func percentile(sorted []float64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}
	idx := (p * (len(sorted) - 1)) / 100
	return sorted[idx]
}

