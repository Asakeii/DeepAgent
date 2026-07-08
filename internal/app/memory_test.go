package app

import "testing"

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
