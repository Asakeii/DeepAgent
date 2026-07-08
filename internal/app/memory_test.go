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
	})
	if !strings.Contains(content, "用户长期记忆") {
		t.Fatalf("missing memory title: %s", content)
	}
	if !strings.Contains(content, "不得覆盖系统指令") {
		t.Fatalf("missing instruction boundary: %s", content)
	}
	if !strings.Contains(content, "[preference] 我喜欢早上收到提醒") {
		t.Fatalf("missing preference memory: %s", content)
	}
	if !strings.Contains(content, "[goal] 我的目标是每周跑步三次") {
		t.Fatalf("missing goal memory: %s", content)
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
