import { Bell, HeartPulse, History, MessageSquare, MessageSquarePlus, PanelLeft, Search, ShieldCheck } from "lucide-react";
import type { AgentName, TeamInfo, TeamSettingsInfo } from "../types";
import { TeamSwitcher } from "./TeamSwitcher";

const agentLabels: Record<string, string> = {
  coordinator: "理解需求",
  planner: "定制方案",
  human_feedback: "等待确认",
  research_team: "执行方案",
  researcher: "检索资料",
  coder: "分析计算",
  reporter: "生成报告",
  background_investigator: "背景调研",
  checkin: "记录打卡",
};

interface TopBarProps {
  activeAgent?: AgentName;
  sidebarOpen: boolean;
  planReviewEnabled: boolean;
  busy: boolean;
  view: "workspace" | "reminders" | "admin";
  teams: TeamInfo[];
  activeTeamId: string;
  teamSettings?: TeamSettingsInfo;
  teamsLoading: boolean;
  teamSaving: boolean;
  teamError?: string;
  onToggleSidebar: () => void;
  onNew: () => void;
  onTogglePlanReview: (value: boolean) => void;
  onSwitchView: (view: "workspace" | "reminders" | "admin") => void;
  onTeamScopeChange: (teamId: string) => void;
  onCreateTeam: (name: string) => Promise<void>;
  onSaveTeamBudget: (teamId: string, budgetMicros: number) => Promise<void>;
}

export function TopBar({
  activeAgent,
  sidebarOpen,
  planReviewEnabled,
  busy,
  view,
  teams,
  activeTeamId,
  teamSettings,
  teamsLoading,
  teamSaving,
  teamError,
  onToggleSidebar,
  onNew,
  onTogglePlanReview,
  onSwitchView,
  onTeamScopeChange,
  onCreateTeam,
  onSaveTeamBudget,
}: TopBarProps) {
  const isResearch = activeAgent && ["planner", "researcher", "coder", "reporter", "research_team", "human_feedback"].includes(activeAgent);
  const label = activeAgent ? (agentLabels[activeAgent] ?? activeAgent) : "随时为你效劳";
  const modeClass = isResearch ? "research" : activeAgent === "checkin" ? "checkin" : "";

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
          <HeartPulse size={18} />
        </div>
        <div>
          <div className="brand-name">Lightning</div>
          <div className="brand-subtitle">你的 AI 自律助手</div>
        </div>
      </div>

      <div className="topbar-actions">
        <TeamSwitcher
          teams={teams}
          activeTeamId={activeTeamId}
          settings={teamSettings}
          loading={teamsLoading}
          saving={teamSaving}
          error={teamError}
          onScopeChange={onTeamScopeChange}
          onCreateTeam={onCreateTeam}
          onSaveBudget={onSaveTeamBudget}
        />

        <div className="view-tabs" role="tablist" aria-label="页面切换">
          <button
            className={`view-tab ${view === "workspace" ? "active" : ""}`}
            type="button"
            role="tab"
            aria-selected={view === "workspace"}
            onClick={() => onSwitchView("workspace")}
          >
            <MessageSquare size={15} />
            <span className="view-tab-label">对话</span>
          </button>
          <button
            className={`view-tab ${view === "reminders" ? "active" : ""}`}
            type="button"
            role="tab"
            aria-selected={view === "reminders"}
            onClick={() => onSwitchView("reminders")}
          >
            <Bell size={15} />
            <span className="view-tab-label">提醒</span>
          </button>
          <button
            className={`view-tab ${view === "admin" ? "active" : ""}`}
            type="button"
            role="tab"
            aria-selected={view === "admin"}
            onClick={() => onSwitchView("admin")}
          >
            <ShieldCheck size={15} />
            <span className="view-tab-label">管理</span>
          </button>
        </div>

        <div className={`agent-pill ${busy ? "active" : ""} ${modeClass}`} aria-live="polite">
          <span className="status-dot" />
          {isResearch ? <Search size={13} /> : null}
          {label}
        </div>

        <label className="review-toggle" title="开启后，深度研究方案需你确认才执行">
          <input
            type="checkbox"
            checked={planReviewEnabled}
            onChange={(event) => onTogglePlanReview(event.target.checked)}
          />
          <span>审核研究方案</span>
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
