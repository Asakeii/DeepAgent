package evalharness

import "testing"

func TestCompareSuitesFindsRegressionsImprovementsAndDeltas(t *testing.T) {
	baseline := SuiteResult{
		Total:    3,
		Passed:   2,
		Failed:   1,
		PassRate: 2.0 / 3.0,
		Averages: map[string]float64{MetricRoutingAccuracy: 1, MetricToolCallAccuracy: 0.5},
		Results: []CaseResult{
			{ID: "regressed", Pass: true, Scores: map[string]float64{MetricRoutingAccuracy: 1}},
			{ID: "improved", Pass: false, Scores: map[string]float64{MetricRoutingAccuracy: 0}, Issues: []string{"old fail"}},
			{ID: "changed", Pass: true, Scores: map[string]float64{MetricToolCallAccuracy: 0.5}},
		},
	}
	candidate := SuiteResult{
		Total:    3,
		Passed:   2,
		Failed:   1,
		PassRate: 2.0 / 3.0,
		Averages: map[string]float64{MetricRoutingAccuracy: 2.0 / 3.0, MetricToolCallAccuracy: 1},
		Results: []CaseResult{
			{ID: "regressed", Pass: false, Scores: map[string]float64{MetricRoutingAccuracy: 0}, Issues: []string{"new fail"}},
			{ID: "improved", Pass: true, Scores: map[string]float64{MetricRoutingAccuracy: 1}},
			{ID: "changed", Pass: true, Scores: map[string]float64{MetricToolCallAccuracy: 1}},
		},
	}

	comparison := CompareSuites(baseline, candidate)
	if comparison.RegressionCount != 1 || comparison.Regressions[0].ID != "regressed" {
		t.Fatalf("unexpected regressions: %+v", comparison.Regressions)
	}
	if comparison.ImprovementCount != 1 || comparison.Improvements[0].ID != "improved" {
		t.Fatalf("unexpected improvements: %+v", comparison.Improvements)
	}
	if len(comparison.ChangedScores) != 1 || comparison.ChangedScores[0].ID != "changed" {
		t.Fatalf("unexpected changed scores: %+v", comparison.ChangedScores)
	}
	if comparison.AverageDeltas[MetricRoutingAccuracy] >= 0 {
		t.Fatalf("routing delta should be negative: %+v", comparison.AverageDeltas)
	}
	if comparison.AverageDeltas[MetricToolCallAccuracy] != 0.5 {
		t.Fatalf("tool delta=%v, want 0.5", comparison.AverageDeltas[MetricToolCallAccuracy])
	}
}
