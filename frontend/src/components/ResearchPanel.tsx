import { CheckCircle2, Code2, ExternalLink, FileSearch, Loader2, Route } from "lucide-react";
import type { AgentName, Plan, ToolActivity } from "../types";
import { shortDomain } from "../lib/format";

const phases = [
  { key: "coordinator", label: "意图分析" },
  { key: "planner", label: "研究计划" },
  { key: "researcher", label: "资料检索" },
  { key: "coder", label: "计算处理" },
  { key: "reporter", label: "报告生成" },
];

interface ResearchPanelProps {
  activeAgent?: AgentName;
  plan?: Plan;
  tools: ToolActivity[];
}

export function ResearchPanel({ activeAgent, plan, tools }: ResearchPanelProps) {
  const activeIndex = Math.max(
    0,
    phases.findIndex((phase) => phase.key === activeAgent),
  );

  return (
    <>
      <div className="panel-section">
        <div className="panel-heading">
          <Route size={16} />
          <h2>研究进度</h2>
        </div>
        <ol className="phase-list">
          {phases.map((phase, index) => {
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
          <h2>计划步骤</h2>
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
          <p className="muted">研究计划生成后会显示在这里。</p>
        )}
      </div>

      <div className="panel-section">
        <div className="panel-heading">
          <ExternalLink size={16} />
          <h2>工具活动</h2>
        </div>
        {tools.length === 0 ? (
          <p className="muted">搜索或抓取页面时会记录来源。</p>
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
