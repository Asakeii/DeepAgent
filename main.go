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

	cfg, err := conf.Load(ctx)
	if err != nil {
		log.Fatal(err)
	}

	if err := infra.InitModel(ctx); err != nil {
		log.Fatal(err)
	}

	if err := infra.InitMCP(ctx); err != nil {
		log.Fatal(err)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("请输入你的需求： ")
	userPrompt, _ := reader.ReadString('\n')
	userPrompt = strings.TrimSpace(userPrompt)

	state := &model.State{
		Messages:          []*schema.Message{schema.UserMessage(userPrompt)},
		Goto:              consts.Coordinator,
		Locale:            "zh-CN",
		MaxPlanIterations: cfg.Setting.MaxPlanIterations,
		MaxStepNum:        cfg.Setting.MaxStepNum,
		AutoAcceptedPlan:  true,
	}

	genFunc := func(ctx context.Context) *model.State {
		return state
	}

	r, err := agent.Builder(ctx, genFunc)
	if err != nil {
		log.Fatal(err)
	}

	report, err := r.Invoke(ctx, consts.Coordinator)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println()
	fmt.Println("========== FINAL REPORT ==========")
	fmt.Println()
	fmt.Println(report)
}
