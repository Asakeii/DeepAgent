import { Activity, BookOpenCheck, Search, Utensils } from "lucide-react";

interface WelcomeStateProps {
  onPrompt: (value: string) => void;
}

const prompts = [
  { icon: Activity, label: "晨跑归来", text: "今天慢跑了五公里" },
  { icon: Utensils, label: "记录午餐", text: "记录一下午餐" },
  { icon: BookOpenCheck, label: "本周回顾", text: "看看这周运动情况" },
  { icon: Search, label: "研究探索", text: "帮我分析一下最近有哪些新出的 AI 编程工具" },
];

export function WelcomeState({ onPrompt }: WelcomeStateProps) {
  return (
    <section className="welcome-state" aria-label="开始新对话">
      <div className="welcome-copy">
        <span className="eyebrow">Lightning Workspace</span>
        <h1>把研究和日常记录放在同一张工作台上。</h1>
        <p>先确认计划，再看检索、工具调用和最终报告；也可以轻量记录运动、饮食和学习。</p>
      </div>
      <div className="prompt-row">
        {prompts.map((prompt) => {
          const Icon = prompt.icon;
          return (
            <button key={prompt.label} className="prompt-chip" type="button" onClick={() => onPrompt(prompt.text)}>
              <Icon size={16} />
              {prompt.label}
            </button>
          );
        })}
      </div>
    </section>
  );
}
