package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"deepAgent/internal/consts"
	"deepAgent/internal/infra"
	"deepAgent/internal/model"
)

// RunPlanner 是教学阶段的独立 Planner。
// 后面接 Graph 时，会拆成 load -> agent -> router 三个节点。
func RunPlanner(ctx context.Context, state *model.State) error {
	messages, err := loadPlannerMessages(ctx, state)
	if err != nil {
		return err
	}

	resp, err := infra.PlanModelFor(ctx).Generate(ctx, messages)
	if err != nil {
		return fmt.Errorf("call plan model: %w", err)
	}

	return routePlannerResult(ctx, state, resp)
}

// loadPlannerMessages 加载 Planner 的系统提示词和用户输入
func loadPlannerMessages(ctx context.Context, state *model.State) ([]*schema.Message, error) {
	sysPrompt, err := infra.GetPromptTemplate(ctx, "planner")
	if err != nil {
		return nil, err
	}

	var promptTemp *prompt.DefaultChatTemplate
	if state.EnableBackgroundInvestigation && len(state.BackgroundInvestigationResults) > 0 {
		promptTemp = prompt.FromMessages(
			schema.Jinja2,
			schema.SystemMessage(sysPrompt),
			schema.MessagesPlaceholder("user_input", true),
			schema.UserMessage(fmt.Sprintf(
				"background investigation results of user query: \n %s",
				state.BackgroundInvestigationResults,
			)),
		)
	} else {
		promptTemp = prompt.FromMessages(
			schema.Jinja2,
			schema.SystemMessage(sysPrompt),
			schema.MessagesPlaceholder("user_input", true),
		)
	}

	messages, err := promptTemp.Format(ctx, map[string]any{
		"locale":              state.Locale,
		"max_step_num":        state.MaxStepNum,
		"max_plan_iterations": state.MaxPlanIterations,
		"CURRENT_TIME":        time.Now().Format("2006-01-02 15:04:05"),
		"user_input":          state.Messages,
	})
	if err != nil {
		return nil, fmt.Errorf("format planner prompt: %w", err)
	}

	return messages, nil
}

func routePlannerResult(ctx context.Context, state *model.State, input *schema.Message) error {
	state.Goto = compose.END
	state.CurrentPlan = &model.Plan{}

	// Planner 必须输出 Plan JSON，这是后续 ResearchTeam 调度的结构化契约。
	// 但模型偶尔会把 JSON 包在 ```json ... ``` 里或前后带多余文字，
	// 直接 Unmarshal 失败就 return error 会让整张图崩溃退出。
	// 这里对齐 deer-go 的容错语义：解析失败不致命。
	if err := json.Unmarshal([]byte(extractPlanJSON(input.Content)), state.CurrentPlan); err != nil {
		log.Printf("[planner] parse plan json failed: %v, raw content: %q", err, input.Content)
		// 已经迭代过至少一次：降级到 Reporter，用已有结果兜底出报告。
		if state.PlanIterations > 0 {
			state.Goto = consts.Reporter
			return nil
		}
		// 从未成功生成过计划：直接结束，避免无计划死循环。
		state.Goto = compose.END
		return nil
	}

	state.PlanIterations++

	// 如果 Planner 判断上下文已经足够，就直接进入 Reporter。
	if state.CurrentPlan.HasEnoughContext {
		state.Goto = consts.Reporter
		return nil
	}

	// 否则先进入 Human Feedback。当前 console 模式下 AutoAcceptedPlan=true，会自动继续到 ResearchTeam。
	state.Goto = consts.Human
	return nil
}

// extractPlanJSON 从模型输出里提取 Plan JSON 文本。
// 优先取 ```json ... ``` 包裹的内容；没有 code fence 就按原文返回，交给 json.Unmarshal 决定成败。
func extractPlanJSON(content string) string {
	s := strings.TrimSpace(content)

	// 去掉开头的 ```json 或 ```
	if strings.HasPrefix(s, "```") {
		// 去掉首行 fence 标记
		if nl := strings.IndexByte(s, '\n'); nl > 0 {
			s = strings.TrimSpace(s[nl+1:])
		}
		// 去掉结尾的 ```
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = strings.TrimSpace(s[:idx])
		}
	}
	return s
}

// loadPlannerMsg 是 Planner 子图里的 load 节点。
// 它不直接接收 *model.State，而是通过 compose.ProcessState 读取 Graph 共享状态。
func loadPlannerMsg(ctx context.Context, input string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		output, err = loadPlannerMessages(ctx, state)
		return err
	})
	return output, err
}

// routerPlanner 是 Planner 子图里的 router 节点。
// 它解析 PlanModel 的输出，写入 state.CurrentPlan，并设置 state.Goto。
func routerPlanner(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		if err := routePlannerResult(ctx, state, input); err != nil {
			return err
		}
		output = state.Goto
		return nil
	})
	return output, err
}

// NewPlanner 构造与 deer-go 对齐的 Planner 子图：
//
//	START -> load -> agent -> router -> END
//
// 当前保留 RunPlanner 作为教学版直接调用入口；后续 Builder 会改为 AddGraphNode 使用此子图。
func NewPlanner[I, O any](ctx context.Context) *compose.Graph[I, O] {
	return buildLoadAgentRouter(
		compose.InvokableLambdaWithOption(loadPlannerMsg),
		withChatModel[I, O](infra.PlanModelFor(ctx)),
		compose.InvokableLambdaWithOption(routerPlanner),
	)
}
