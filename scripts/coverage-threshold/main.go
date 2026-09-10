package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Exit codes: usage/parse failures use 2 so a broken harness is distinguishable
// from a threshold violation (1) and from success (0).
const (
	exitOK             = 0
	exitBelowThreshold = 1
	exitUsage          = 2

	maxPercent = 100.0
)

// errNoTotalLine reports a report that is not `go tool cover -func` output.
var errNoTotalLine = errors.New("no total: line found; is this a `go tool cover -func` report?")

// coverage-threshold reads a `go tool cover -func` report on stdin and exits
// nonzero when the trailing total coverage line is below the threshold passed
// as the only argument (a percentage). It replaces a fragile awk one-liner:
// a malformed report now fails loudly instead of silently passing the gate.
//
// Usage: go tool cover -func=coverage.out | go run ./scripts/coverage-threshold 95
func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: coverage-threshold <threshold-percent> (report on stdin)")

		os.Exit(exitUsage)
	}

	threshold, err := strconv.ParseFloat(os.Args[1], 64)
	if err != nil || threshold < 0 || threshold > maxPercent {
		fmt.Fprintf(os.Stderr, "invalid threshold %q: want a percentage in [0, 100]\n", os.Args[1])

		os.Exit(exitUsage)
	}

	total, err := findTotalLine(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "coverage-threshold: %v\n", err)

		os.Exit(exitUsage)
	}

	pct, err := parseTotalPercent(total)
	if err != nil {
		fmt.Fprintf(os.Stderr, "coverage-threshold: %v\n", err)

		os.Exit(exitUsage)
	}

	fmt.Fprintln(os.Stdout, total)

	if pct < threshold {
		fmt.Fprintf(
			os.Stderr,
			"::error::Coverage %.2f%% is below %.2f%% threshold\n",
			pct,
			threshold,
		)

		os.Exit(exitBelowThreshold)
	}

	os.Exit(exitOK)
}

// findTotalLine scans the report for the single "total:" summary line.
func findTotalLine(f *os.File) (string, error) {
	total := ""

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		if line := scanner.Text(); strings.HasPrefix(line, "total:") {
			total = line
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading coverage report: %w", err)
	}

	if total == "" {
		return "", errNoTotalLine
	}

	return total, nil
}

// parseTotalPercent extracts the trailing percentage from a total line shaped
// like: `total:<space>(statements)<space>96.3%`.
func parseTotalPercent(total string) (float64, error) {
	fields := strings.Fields(total)

	last := fields[len(fields)-1]

	pct, err := strconv.ParseFloat(strings.TrimSuffix(last, "%"), 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse coverage from %q: %w", total, err)
	}

	return pct, nil
}
