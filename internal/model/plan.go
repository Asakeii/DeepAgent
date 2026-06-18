package model

type StepType string

const (
	Research   StepType = "research"
	Processing StepType = "processing"
)

type Step struct {
	NeedWebSearch bool     `json:"need_web_search"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	StepType      StepType `json:"step_type"`
	ExecutionRes  *string  `json:"execution_res,omitempty"`
}

type Plan struct {
	Locale           string `json:"locale"`
	HasEnoughContext bool   `json:"has_enough_context"`
	Thought          string `json:"thought"`
	Title            string `json:"title"`
	Steps            []Step `json:"steps"`
}

