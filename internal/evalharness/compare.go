package evalharness

import "sort"

func CompareSuites(baseline, candidate SuiteResult) ABComparison {
	comparison := ABComparison{
		Baseline:      baseline,
		Candidate:     candidate,
		PassRateDelta: candidate.PassRate - baseline.PassRate,
		AverageDeltas: averageDeltas(baseline.Averages, candidate.Averages),
	}

	baseByID := resultsByID(baseline.Results)
	candidateByID := resultsByID(candidate.Results)
	ids := unionIDs(baseByID, candidateByID)
	for _, id := range ids {
		baseResult, hasBase := baseByID[id]
		candidateResult, hasCandidate := candidateByID[id]
		if !hasBase {
			baseResult = CaseResult{ID: id, Scores: map[string]float64{}, Issues: []string{"missing baseline result"}}
		}
		if !hasCandidate {
			candidateResult = CaseResult{ID: id, Scores: map[string]float64{}, Issues: []string{"missing candidate result"}}
		}
		delta := caseDelta(baseResult, candidateResult)
		if baseResult.Pass && !candidateResult.Pass {
			comparison.Regressions = append(comparison.Regressions, delta)
			continue
		}
		if !baseResult.Pass && candidateResult.Pass {
			comparison.Improvements = append(comparison.Improvements, delta)
			continue
		}
		if len(delta.ScoreDeltas) > 0 {
			comparison.ChangedScores = append(comparison.ChangedScores, delta)
		}
	}
	comparison.RegressionCount = len(comparison.Regressions)
	comparison.ImprovementCount = len(comparison.Improvements)
	return comparison
}

func averageDeltas(baseline, candidate map[string]float64) map[string]float64 {
	out := map[string]float64{}
	keys := unionMetricKeys(baseline, candidate)
	for _, key := range keys {
		delta := candidate[key] - baseline[key]
		if delta != 0 {
			out[key] = delta
		}
	}
	return out
}

func resultsByID(results []CaseResult) map[string]CaseResult {
	out := make(map[string]CaseResult, len(results))
	for _, result := range results {
		out[result.ID] = result
	}
	return out
}

func unionIDs(a, b map[string]CaseResult) []string {
	set := map[string]bool{}
	for id := range a {
		set[id] = true
	}
	for id := range b {
		set[id] = true
	}
	out := make([]string, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func unionMetricKeys(a, b map[string]float64) []string {
	set := map[string]bool{}
	for key := range a {
		set[key] = true
	}
	for key := range b {
		set[key] = true
	}
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func caseDelta(baseline, candidate CaseResult) CaseDelta {
	delta := CaseDelta{
		ID:              candidate.ID,
		BaselinePass:    baseline.Pass,
		CandidatePass:   candidate.Pass,
		BaselineScores:  baseline.Scores,
		CandidateScores: candidate.Scores,
		BaselineIssues:  baseline.Issues,
		CandidateIssues: candidate.Issues,
		ScoreDeltas:     averageDeltas(baseline.Scores, candidate.Scores),
	}
	if delta.ID == "" {
		delta.ID = baseline.ID
	}
	if len(delta.ScoreDeltas) == 0 {
		delta.ScoreDeltas = nil
	}
	return delta
}
