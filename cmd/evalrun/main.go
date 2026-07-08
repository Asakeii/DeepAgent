package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"deepAgent/internal/evalharness"
)

func main() {
	casesPath := flag.String("cases", "evals/cases/routing.jsonl", "path to eval cases jsonl")
	observationsPath := flag.String("observations", "", "path to observations jsonl")
	minPassRate := flag.Float64("min-pass-rate", 1, "minimum pass rate required for exit code 0")
	flag.Parse()

	if *observationsPath == "" {
		fmt.Fprintln(os.Stderr, "-observations is required")
		os.Exit(2)
	}

	cases, err := evalharness.LoadCases(*casesPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load cases: %v\n", err)
		os.Exit(2)
	}
	observations, err := evalharness.LoadObservations(*observationsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load observations: %v\n", err)
		os.Exit(2)
	}

	result := evalharness.EvaluateSuite(cases, observations)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(result)
	if result.PassRate < *minPassRate {
		os.Exit(1)
	}
}
