package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino/schema"

	"deepAgent/conf"
	"deepAgent/internal/agent"
	"deepAgent/internal/consts"
	"deepAgent/internal/infra"
	"deepAgent/internal/model"
)

func main() {
	ctx := context.Background()

	// 1. 加载本地 YAML 配置，后续模型、MCP、运行参数都从这里读取。
	cfg, err := conf.Load(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// 2. 初始化 LLM：ChatModel 用于普通对话/Agent，PlanModel 用于 Planner 结构化输出。
	if err := infra.InitModel(ctx); err != nil {
		log.Fatal(err)
	}

	// 3. 初始化 MCP 客户端。Coder/Researcher 会在构建子图时把 MCP tools 挂到 ReAct Agent 上。
	if err := infra.InitMCP(ctx); err != nil {
		log.Fatal(err)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("请输入你的需求： ")
	userPrompt, _ := reader.ReadString('\n')
	userPrompt = strings.TrimSpace(userPrompt)

	// State 是整张图共享的运行时状态；每个节点只改 State，不直接调用下一个节点。
	state := &model.State{
		Messages:                      []*schema.Message{schema.UserMessage(userPrompt)},
		Goto:                          consts.Coordinator,
		Locale:                        "zh-CN",
		MaxPlanIterations:             cfg.Setting.MaxPlanIterations,
		MaxStepNum:                    cfg.Setting.MaxStepNum,
		AutoAcceptedPlan:              true,
		EnableBackgroundInvestigation: cfg.Setting.EnableBackgroundInvestigation,
	}

	// GenLocalState 告诉 Eino 每次运行这张图时使用哪一份本地状态。
	genFunc := func(ctx context.Context) *model.State {
		return state
	}

	// Builder 负责编译总图，内部把 Coordinator/Planner/Researcher 等子图连接起来。
	r, err := agent.Builder(ctx, genFunc)
	if err != nil {
		log.Fatal(err)
	}

	// 从 Coordinator 入口启动，最终 Reporter 子图会返回报告正文。
	report, err := r.Invoke(ctx, consts.Coordinator)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println()
	fmt.Println("========== FINAL REPORT ==========")
	fmt.Println()
	fmt.Println(report)
}
