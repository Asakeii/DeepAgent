import { MessageSquarePlus, PanelLeftClose, Search } from "lucide-react";
import type { SessionInfo } from "../types";
import { formatRelativeTime } from "../lib/format";

interface SessionSidebarProps {
  sessions: SessionInfo[];
  activeThreadId: string;
  isOpen: boolean;
  loading: boolean;
  onClose: () => void;
  onNew: () => void;
  onSelect: (threadId: string) => void;
}

export function SessionSidebar({
  sessions,
  activeThreadId,
  isOpen,
  loading,
  onClose,
  onNew,
  onSelect,
}: SessionSidebarProps) {
  return (
    <aside className={`session-sidebar ${isOpen ? "is-open" : ""}`} aria-label="历史会话">
      <div className="sidebar-header">
        <div>
          <span className="eyebrow">Sessions</span>
          <h2>历史会话</h2>
        </div>
        <button className="icon-button ghost mobile-only" type="button" onClick={onClose} aria-label="关闭历史">
          <PanelLeftClose size={18} />
        </button>
      </div>

      <button className="new-chat-button" type="button" onClick={onNew}>
        <MessageSquarePlus size={17} />
        新对话
      </button>

      <div className="sidebar-search" aria-hidden="true">
        <Search size={15} />
        <span>最近活动优先</span>
      </div>

      <div className="session-list">
        {loading ? (
          <>
            <div className="session-skeleton" />
            <div className="session-skeleton" />
            <div className="session-skeleton short" />
          </>
        ) : sessions.length === 0 ? (
          <div className="empty-panel">
            <strong>暂无历史</strong>
            <span>完成一次研究或打卡后会出现在这里。</span>
          </div>
        ) : (
          sessions.map((session) => (
            <button
              key={session.threadId}
              className={`session-item ${session.threadId === activeThreadId ? "active" : ""}`}
              type="button"
              onClick={() => onSelect(session.threadId)}
            >
              <span className="session-title">{session.firstMsg || "新对话"}</span>
              <span className="session-meta">
                {formatRelativeTime(session.lastAt)} · {session.msgCount} 条
              </span>
            </button>
          ))
        )}
      </div>
    </aside>
  );
}
