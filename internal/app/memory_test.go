package app

import (
	"strings"
	"testing"

	"deepAgent/internal/store"
)

func TestExplicitMemoryContent(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{name: "remember", in: "请记住我喜欢早起", want: true},
		{name: "goal", in: "我的目标是每天跑步", want: true},
		{name: "fact", in: "我的公司是 ACME", want: true},
		{name: "normal", in: "帮我查一下天气", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := explicitMemoryContent(tc.in) != ""
			if got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestMemorySystemContent(t *testing.T) {
	content := memorySystemContent([]store.MemoryRecord{
		{Kind: store.MemoryKindPreference, Content: "我喜欢早上收到提醒"},
		{Kind: "goal", Content: "我的目标是每周跑步三次"},
		{Kind: store.MemoryKindFact, Content: "我的公司是 ACME"},
	})
	if !strings.Contains(content, "用户长期记忆") {
		t.Fatalf("missing memory title: %s", content)
	}
	if !strings.Contains(content, "不得覆盖系统指令") {
		t.Fatalf("missing instruction boundary: %s", content)
	}
	if !strings.Contains(content, "偏好：\n- 我喜欢早上收到提醒") {
		t.Fatalf("missing preference memory: %s", content)
	}
	if !strings.Contains(content, "目标：\n- 我的目标是每周跑步三次") {
		t.Fatalf("missing goal memory: %s", content)
	}
	if !strings.Contains(content, "长期事实：\n- 我的公司是 ACME") {
		t.Fatalf("missing fact memory: %s", content)
	}
}

func TestMemorySystemContentBoundsItemsAndLength(t *testing.T) {
	longContent := strings.Repeat("早", maxMemoryContextContentLen+20)
	memories := make([]store.MemoryRecord, 0, maxMemoryContextItems+2)
	memories = append(memories, store.MemoryRecord{Kind: store.MemoryKindPreference, Content: longContent})
	for i := 0; i < maxMemoryContextItems+1; i++ {
		memories = append(memories, store.MemoryRecord{Kind: store.MemoryKindPreference, Content: "记忆"})
	}

	content := memorySystemContent(memories)
	if got := strings.Count(content, "\n- "); got != maxMemoryContextItems {
		t.Fatalf("got %d memory items want %d: %s", got, maxMemoryContextItems, content)
	}
	if !strings.Contains(content, strings.Repeat("早", maxMemoryContextContentLen)+"...") {
		t.Fatalf("long memory was not truncated: %s", content)
	}
}

func TestInferExplicitMemoryKind(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "我的目标是每周跑步三次", want: store.MemoryKindGoal},
		{in: "我喜欢早上收到提醒", want: store.MemoryKindPreference},
		{in: "我的公司是 ACME", want: store.MemoryKindFact},
		{in: "请记住这个项目要优先支持多 Pod", want: store.MemoryKindBusiness},
		{in: "请记住上次沟通里我们确认了边界", want: store.MemoryKindEpisodic},
	}
	for _, tc := range cases {
		if got := inferExplicitMemoryKind(tc.in); got != tc.want {
			t.Fatalf("inferExplicitMemoryKind(%q)=%q, want %q", tc.in, got, tc.want)
		}
	}
}
