import { CheckCircle2, Code2, ExternalLink, FileSearch, HeartPulse, Loader2, Route, Sparkles } from "lucide-react";
import type { AgentName, Plan, ToolActivity } from "../types";
import { shortDomain } from "../lib/format";

const researchPhases = [
  { key: "coordinator", label: "理解需求" },
  { key: "planner", label: "定制方案" },
  { key: "researcher", label: "检索资料" },
  { key: "coder", label: "分析计算" },
  { key: "reporter", label: "生成报告" },
];

const researchAgents = new Set([
  "planner",
  "human_feedback",
  "research_team",
  "researcher",
  "coder",
  "reporter",
  "background_investigator",
]);

interface ActivityPanelProps {
  activeAgent?: AgentName;
  plan?: Plan;
  tools: ToolActivity[];
  busy: boolean;
}

export function ActivityPanel({ activeAgent, plan, tools, busy }: ActivityPanelProps) {
  const isResearch =
    Boolean(plan) ||
    tools.length > 0 ||
    (activeAgent && researchAgents.has(activeAgent)) ||
    (busy && activeAgent === "coordinator");
  const isCheckin = activeAgent === "checkin";

  if (!isResearch && !isCheckin && !busy) {
    return <IdlePanel />;
  }

  if (isCheckin && !isResearch) {
    return <CheckinPanel />;
  }

  return <ResearchPanel activeAgent={activeAgent} plan={plan} tools={tools} />;
}

function IdlePanel() {
  return (
    <>
      <div className="panel-section">
        <div className="panel-heading">
          <HeartPulse size={16} />
          <h2>助手能力</h2>
        </div>
        <div className="capability-list">
          <div className="capability-item">
            <span className="capability-dot wellness" />
            <div>
              <strong>日常打卡</strong>
              <p>记录运动、饮食、学习，回顾本周进展</p>
            </div>
          </div>
          <div className="capability-item">
            <span className="capability-dot wellness" />
            <div>
              <strong>识图记录</strong>
              <p>上传食物图片，自动估算营养摄入</p>
            </div>
          </div>
          <div className="capability-item">
            <span className="capability-dot wellness" />
            <div>
              <strong>定时提醒</strong>
              <p>用自然语言设置重复提醒，保持节奏</p>
            </div>
          </div>
          <div className="capability-item">
            <span className="capability-dot research" />
            <div>
              <strong>深度研究</strong>
              <p>联网检索、分析资料，定制训练或饮食方案</p>
            </div>
          </div>
        </div>
      </div>

      <div className="panel-section">
        <div className="insight-card">
          <Sparkles size={16} />
          <p>发送消息后，这里会显示教练回复、研究进度和工具活动。</p>
        </div>
      </div>
    </>
  );
}

function CheckinPanel() {
  return (
    <div className="panel-section">
      <div className="panel-heading">
        <HeartPulse size={16} />
        <h2>教练模式</h2>
      </div>
      <div className="mode-banner checkin">
        <span className="mode-pulse" />
        <div>
          <strong>正在记录与回顾</strong>
          <p>帮你打卡、总结进展，或管理提醒事项。</p>
        </div>
      </div>
    </div>
  );
}

function ResearchPanel({ activeAgent, plan, tools }: { activeAgent?: AgentName; plan?: Plan; tools: ToolActivity[] }) {
  const activeIndex = Math.max(
    0,
    researchPhases.findIndex((phase) => phase.key === activeAgent),
  );

  return (
    <>
      <div className="panel-section">
        <div className="mode-banner research">
          <Sparkles size={16} />
          <div>
            <strong>深度研究进行中</strong>
            <p>联网检索资料，为你定制可执行的自律方案。</p>
          </div>
        </div>
      </div>

      <div className="panel-section">
        <div className="panel-heading">
          <Route size={16} />
          <h2>研究进度</h2>
        </div>
        <ol className="phase-list">
          {researchPhases.map((phase, index) => {
            const state = index < activeIndex ? "done" : index === activeIndex ? "active" : "";
            return (
              <li key={phase.key} className={state}>
                <span>{state === "done" ? <CheckCircle2 size={14} /> : <span className="phase-dot" />}</span>
                {phase.label}
              </li>
            );
          })}
        </ol>
      </div>

      <div className="panel-section">
        <div className="panel-heading">
          <FileSearch size={16} />
          <h2>方案步骤</h2>
        </div>
        {plan ? (
          <div className="plan-summary">
            <strong>{plan.title}</strong>
            <p>{plan.thought}</p>
            <ol>
              {plan.steps.map((step, index) => (
                <li key={`${step.title}-${index}`}>
                  <span>{step.step_type === "processing" ? <Code2 size={13} /> : <FileSearch size={13} />}</span>
                  <div>
                    <b>{step.title}</b>
                    <small>{step.description}</small>
                  </div>
                </li>
              ))}
            </ol>
          </div>
        ) : (
          <p className="muted">方案生成后会显示具体步骤。</p>
        )}
      </div>

      <div className="panel-section">
        <div className="panel-heading">
          <ExternalLink size={16} />
          <h2>工具活动</h2>
        </div>
        {tools.length === 0 ? (
          <p className="muted">检索网页或执行计算时会记录来源。</p>
        ) : (
          <div className="tool-list">
            {tools.map((tool) => (
              <div className="tool-item" key={tool.id}>
                <div className="tool-title">
                  {tool.status === "running" ? <Loader2 className="spin" size={14} /> : <CheckCircle2 size={14} />}
                  <span>{tool.query || tool.name}</span>
                </div>
                {tool.urls.length > 0 ? (
                  <ul>
                    {tool.urls.map((url) => (
                      <li key={url}>
                        <a href={url} target="_blank" rel="noreferrer">
                          {shortDomain(url)}
                        </a>
                      </li>
                    ))}
                  </ul>
                ) : null}
              </div>
            ))}
          </div>
        )}
      </div>
    </>
  );
}
