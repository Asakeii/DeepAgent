import { History, MessageSquarePlus, PanelLeft, Sparkles } from "lucide-react";
import type { AgentName } from "../types";

const agentLabels: Record<string, string> = {
  coordinator: "分析意图",
  planner: "制定计划",
  human_feedback: "等待确认",
  research_team: "调度步骤",
  researcher: "检索资料",
  coder: "执行处理",
  reporter: "撰写报告",
  background_investigator: "背景调查",
  checkin: "记录打卡",
};

interface TopBarProps {
  activeAgent?: AgentName;
  sidebarOpen: boolean;
  planReviewEnabled: boolean;
  busy: boolean;
  onToggleSidebar: () => void;
  onNew: () => void;
  onTogglePlanReview: (value: boolean) => void;
}

export function TopBar({
  activeAgent,
  sidebarOpen,
  planReviewEnabled,
  busy,
  onToggleSidebar,
  onNew,
  onTogglePlanReview,
}: TopBarProps) {
  const label = activeAgent ? agentLabels[activeAgent] ?? activeAgent : "准备就绪";

  return (
    <header className="topbar">
      <div className="brand-block">
        <button
          className="icon-button ghost desktop-hidden"
          type="button"
          onClick={onToggleSidebar}
          aria-label={sidebarOpen ? "关闭历史" : "打开历史"}
        >
          <PanelLeft size={18} />
        </button>
        <div className="brand-mark" aria-hidden="true">
          <Sparkles size={18} />
        </div>
        <div>
          <div className="brand-name">Lightning</div>
          <div className="brand-subtitle">多 Agent 研究与自律打卡工作台</div>
        </div>
      </div>

      <div className="topbar-actions">
        <div className={`agent-pill ${busy ? "active" : ""}`} aria-live="polite">
          <span className="status-dot" />
          {label}
        </div>

        <label className="review-toggle">
          <input
            type="checkbox"
            checked={planReviewEnabled}
            onChange={(event) => onTogglePlanReview(event.target.checked)}
          />
          <span>审核研究计划</span>
        </label>

        <button className="icon-button ghost desktop-hidden" type="button" onClick={onToggleSidebar} aria-label="历史">
          <History size={18} />
        </button>
        <button className="secondary-button" type="button" onClick={onNew}>
          <MessageSquarePlus size={16} />
          新对话
        </button>
      </div>
    </header>
  );
}
