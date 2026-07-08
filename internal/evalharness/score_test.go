package evalharness

import "testing"

func TestEvaluateCasePassesExpectedRouteToolsAndFinal(t *testing.T) {
	result := EvaluateCase(Case{
		ID:             "case-1",
		ExpectedRoute:  "checkin",
		ExpectedTools:  []string{"record_checkin"},
		ForbiddenTools: []string{"web_search"},
		FinalContains:  []string{"已记录"},
	}, Observation{
		ID:    "case-1",
		Route: "checkin",
		Tools: []string{"record_checkin"},
		Final: "已记录 学习 打卡",
	})
	if !result.Pass {
		t.Fatalf("Pass=false issues=%v", result.Issues)
	}
	if result.Scores[MetricRoutingAccuracy] != 1 || result.Scores[MetricToolCallAccuracy] != 1 || result.Scores[MetricFinalAnswerCompleteness] != 1 {
		t.Fatalf("unexpected scores: %+v", result.Scores)
	}
}

func TestEvaluateCaseFailsOnWrongRouteAndTool(t *testing.T) {
	result := EvaluateCase(Case{
		ID:            "case-2",
		ExpectedRoute: "research",
		ExpectedTools: []string{"web_search"},
	}, Observation{
		ID:    "case-2",
		Route: "checkin",
		Tools: []string{"record_checkin"},
	})
	if result.Pass {
		t.Fatal("Pass=true, want false")
	}
	if result.Scores[MetricRoutingAccuracy] != 0 {
		t.Fatalf("routing score=%v, want 0", result.Scores[MetricRoutingAccuracy])
	}
	if result.Scores[MetricToolCallAccuracy] != 0 {
		t.Fatalf("tool score=%v, want 0", result.Scores[MetricToolCallAccuracy])
	}
}

func TestEvaluateSuiteMarksMissingObservation(t *testing.T) {
	result := EvaluateSuite([]Case{{ID: "missing", ExpectedRoute: "direct"}}, map[string]Observation{})
	if result.Passed != 0 || result.Failed != 1 || result.PassRate != 0 {
		t.Fatalf("unexpected suite result: %+v", result)
	}
	if len(result.Results) != 1 || result.Results[0].Pass {
		t.Fatalf("missing observation should fail: %+v", result.Results)
	}
}
