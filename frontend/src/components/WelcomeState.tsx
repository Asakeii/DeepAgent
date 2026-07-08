import { Activity, Apple, BellRing, BookOpen, Dumbbell, Search, Sparkles, Target } from "lucide-react";
import disciplineVisual from "../assets/discipline-visual.webp";

interface WelcomeStateProps {
  onPrompt: (value: string) => void;
}

const disciplinePrompts = [
  { icon: Activity, label: "记录运动", text: "今天慢跑了五公里，帮我记一下", hint: "运动打卡" },
  { icon: Apple, label: "记录饮食", text: "午餐吃了一碗牛肉面，帮我记录", hint: "饮食打卡" },
  { icon: BookOpen, label: "学习回顾", text: "看看这周的学习和运动情况", hint: "周回顾" },
  { icon: BellRing, label: "设置提醒", text: "每天晚上 9 点提醒我拉伸 10 分钟", hint: "定时提醒" },
];

const researchPrompts = [
  {
    icon: Target,
    label: "定制训练计划",
    text: "帮我研究一下适合初学者的居家力量训练方案，制定一份 4 周计划",
    hint: "方案研究",
  },
  {
    icon: Search,
    label: "深度调研",
    text: "调研一下地中海饮食的科学依据和实操建议，帮我判断是否适合我",
    hint: "资料研究",
  },
];

function greeting() {
  const hour = new Date().getHours();
  if (hour < 6) return "夜深了，记得休息";
  if (hour < 11) return "早上好";
  if (hour < 14) return "中午好";
  if (hour < 18) return "下午好";
  return "晚上好";
}

export function WelcomeState({ onPrompt }: WelcomeStateProps) {
  return (
    <section className="welcome-state" aria-label="开始新对话">
      <div className="welcome-layout">
        <div className="welcome-hero">
          <span className="eyebrow">自律监督</span>
          <h1>
            {greeting()}，<br />
            <span className="welcome-accent">把今天稳稳推进。</span>
          </h1>
          <p>
            记录运动、饮食与学习，设置提醒保持节奏。需要定制计划或深入研究时，我会联网检索、分析资料，为你生成可执行的方案。
          </p>
          <div className="welcome-signals" aria-label="当前监督状态">
            <span>今日待记录</span>
            <span>提醒可追踪</span>
            <span>方案可确认</span>
          </div>
        </div>

        <div className="welcome-visual" aria-hidden="true">
          <div className="visual-stage">
            <img className="discipline-visual" src={disciplineVisual} alt="" />
            <span className="visual-orbit one" />
            <span className="visual-orbit two" />
            <div className="visual-status-card">
              <span>监督中</span>
              <strong>节奏稳定</strong>
              <div className="visual-meter">
                <span />
              </div>
            </div>
            <div className="visual-checkpoint">
              <Sparkles size={14} />
              <span>下一步已就绪</span>
            </div>
          </div>
        </div>
      </div>

      <div className="welcome-sections">
        <div className="welcome-section">
          <div className="welcome-section-head">
            <Dumbbell size={15} />
            <span>今日自律</span>
          </div>
          <div className="prompt-grid">
            {disciplinePrompts.map((prompt) => {
              const Icon = prompt.icon;
              return (
                <button key={prompt.label} className="prompt-card discipline" type="button" onClick={() => onPrompt(prompt.text)}>
                  <span className="prompt-card-icon">
                    <Icon size={18} />
                  </span>
                  <span className="prompt-card-body">
                    <strong>{prompt.label}</strong>
                    <small>{prompt.hint}</small>
                  </span>
                </button>
              );
            })}
          </div>
        </div>

        <div className="welcome-section">
          <div className="welcome-section-head research">
            <Sparkles size={15} />
            <span>深度研究</span>
            <small>为自律目标定制方案</small>
          </div>
          <div className="prompt-grid research">
            {researchPrompts.map((prompt) => {
              const Icon = prompt.icon;
              return (
                <button key={prompt.label} className="prompt-card research" type="button" onClick={() => onPrompt(prompt.text)}>
                  <span className="prompt-card-icon">
                    <Icon size={18} />
                  </span>
                  <span className="prompt-card-body">
                    <strong>{prompt.label}</strong>
                    <small>{prompt.hint}</small>
                  </span>
                </button>
              );
            })}
          </div>
        </div>
      </div>
    </section>
  );
}
