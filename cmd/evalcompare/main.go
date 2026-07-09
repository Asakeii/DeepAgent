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
	baselinePath := flag.String("baseline", "", "path to baseline observations jsonl")
	candidatePath := flag.String("candidate", "", "path to candidate observations jsonl")
	maxRegressions := flag.Int("max-regressions", 0, "maximum allowed case regressions")
	minPassRateDelta := flag.Float64("min-pass-rate-delta", 0, "minimum allowed candidate-baseline pass rate delta")
	flag.Parse()

	if *baselinePath == "" || *candidatePath == "" {
		fmt.Fprintln(os.Stderr, "-baseline and -candidate are required")
		os.Exit(2)
	}

	cases, err := evalharness.LoadCases(*casesPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load cases: %v\n", err)
		os.Exit(2)
	}
	baseline, err := loadSuite(*baselinePath, cases)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load baseline: %v\n", err)
		os.Exit(2)
	}
	candidate, err := loadSuite(*candidatePath, cases)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load candidate: %v\n", err)
		os.Exit(2)
	}

	comparison := evalharness.CompareSuites(baseline, candidate)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(comparison)
	if comparison.RegressionCount > *maxRegressions || comparison.PassRateDelta < *minPassRateDelta {
		os.Exit(1)
	}
}

func loadSuite(observationsPath string, cases []evalharness.Case) (evalharness.SuiteResult, error) {
	observations, err := evalharness.LoadObservations(observationsPath)
	if err != nil {
		return evalharness.SuiteResult{}, err
	}
	return evalharness.EvaluateSuite(cases, observations), nil
}
