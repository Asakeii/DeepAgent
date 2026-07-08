package evalharness

import (
	"fmt"
	"strings"
)

const (
	MetricRoutingAccuracy         = "routing_accuracy"
	MetricToolCallAccuracy        = "tool_call_accuracy"
	MetricFinalAnswerCompleteness = "final_answer_completeness"
)

func EvaluateCase(c Case, obs Observation) CaseResult {
	result := CaseResult{
		ID:     c.ID,
		Pass:   true,
		Scores: map[string]float64{},
	}

	if c.ExpectedRoute != "" {
		score := boolScore(equalFoldTrim(c.ExpectedRoute, obs.Route))
		result.Scores[MetricRoutingAccuracy] = score
		if score == 0 {
			result.Pass = false
			result.Issues = append(result.Issues, fmt.Sprintf("route=%q want %q", obs.Route, c.ExpectedRoute))
		}
	}

	if len(c.ExpectedTools) > 0 || len(c.ForbiddenTools) > 0 {
		score := scoreTools(c, obs.Tools)
		result.Scores[MetricToolCallAccuracy] = score
		if score < 1 {
			result.Pass = false
			result.Issues = append(result.Issues, "tool calls did not match expectations")
		}
	}

	if len(c.FinalContains) > 0 {
		score := scoreContains(obs.Final, c.FinalContains)
		result.Scores[MetricFinalAnswerCompleteness] = score
		if score < 1 {
			result.Pass = false
			result.Issues = append(result.Issues, "final answer missed required content")
		}
	}

	if len(obs.Errors) > 0 {
		result.Pass = false
		result.Issues = append(result.Issues, "observation contains errors: "+strings.Join(obs.Errors, "; "))
	}

	return result
}

func EvaluateSuite(cases []Case, observations map[string]Observation) SuiteResult {
	result := SuiteResult{
		Total:    len(cases),
		Results:  make([]CaseResult, 0, len(cases)),
		Averages: map[string]float64{},
	}
	metricCounts := map[string]int{}

	for _, c := range cases {
		obs, ok := observations[c.ID]
		if !ok {
			result.Failed++
			result.Results = append(result.Results, CaseResult{
				ID:     c.ID,
				Pass:   false,
				Scores: map[string]float64{},
				Issues: []string{"missing observation"},
			})
			continue
		}
		caseResult := EvaluateCase(c, obs)
		if caseResult.Pass {
			result.Passed++
		} else {
			result.Failed++
		}
		for metric, score := range caseResult.Scores {
			result.Averages[metric] += score
			metricCounts[metric]++
		}
		result.Results = append(result.Results, caseResult)
	}

	if result.Total > 0 {
		result.PassRate = float64(result.Passed) / float64(result.Total)
	}
	for metric, count := range metricCounts {
		if count > 0 {
			result.Averages[metric] /= float64(count)
		}
	}
	return result
}

func scoreTools(c Case, actual []string) float64 {
	actualSet := stringSet(actual)
	total := len(c.ExpectedTools) + len(c.ForbiddenTools)
	if total == 0 {
		return 1
	}

	passed := 0
	for _, name := range c.ExpectedTools {
		if actualSet[normalize(name)] {
			passed++
		}
	}
	for _, name := range c.ForbiddenTools {
		if !actualSet[normalize(name)] {
			passed++
		}
	}
	return float64(passed) / float64(total)
}

func scoreContains(final string, expected []string) float64 {
	if len(expected) == 0 {
		return 1
	}
	final = strings.ToLower(final)
	passed := 0
	for _, item := range expected {
		if strings.Contains(final, strings.ToLower(item)) {
			passed++
		}
	}
	return float64(passed) / float64(len(expected))
}

func stringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[normalize(value)] = true
	}
	return out
}

func equalFoldTrim(a, b string) bool {
	return normalize(a) == normalize(b)
}

func normalize(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func boolScore(ok bool) float64 {
	if ok {
		return 1
	}
	return 0
}
