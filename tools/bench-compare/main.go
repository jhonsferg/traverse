package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// BenchmarkResult represents a single benchmark result line
type BenchmarkResult struct {
	name  string
	count int64
	time  float64 // nanoseconds
	bytes int64
	alloc int64
}

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: bench-compare <new.txt> <baseline.txt>\n")
		os.Exit(1)
	}

	newFile := args[0]
	baselineFile := args[1]

	newResults := parseBenchmarks(newFile)
	baselineResults := parseBenchmarks(baselineFile)

	fmt.Printf("Benchmark Comparison: %s vs %s\n", newFile, baselineFile)
	fmt.Println("(threshold: ±10% is acceptable)")
	fmt.Println()

	anyRegression := false

	for name, newBench := range newResults {
		baseBench, exists := baselineResults[name]
		if !exists {
			fmt.Printf("NEW     %s\n", name)
			continue
		}

		// Compare ns/op
		delta := float64(newBench.time-baseBench.time) / float64(baseBench.time) * 100
		status := "✓"
		if delta > 10 {
			status = "⚠️ REGRESSION"
			anyRegression = true
		} else if delta < -10 {
			status = "✓ IMPROVEMENT"
		}

		fmt.Printf("%s %s: %v ns/op (baseline: %v ns/op, %+.1f%%)\n",
			status, name, newBench.time, baseBench.time, delta)
	}

	fmt.Println()
	if anyRegression {
		fmt.Println("⚠️  Performance regressions detected!")
		os.Exit(1)
	}
	fmt.Println("✓ No regressions detected")
}

func parseBenchmarks(filename string) map[string]BenchmarkResult {
	results := make(map[string]BenchmarkResult)

	file, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening file: %v\n", err)
		return results
	}
	defer file.Close()

	// Pattern: "Benchmark..." lines from `go test -bench`
	benchRegex := regexp.MustCompile(`^Benchmark(\S+)\s+\d+\s+(\d+(?:\.\d+)?)\s+ns/op`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "Benchmark") {
			continue
		}

		matches := benchRegex.FindStringSubmatch(line)
		if len(matches) < 3 {
			continue
		}

		name := matches[1]
		timeStr := matches[2]

		time, _ := strconv.ParseFloat(timeStr, 64)
		results[name] = BenchmarkResult{
			name: name,
			time: time,
		}
	}

	return results
}
