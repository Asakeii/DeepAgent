import { AlertTriangle, Bell, BellOff, Bot, Clock3, Repeat2, User } from "lucide-react";
import type { TranscriptItem } from "../types";
import { MarkdownReport } from "./MarkdownReport";
import { WelcomeState } from "./WelcomeState";

interface MessageListProps {
  items: TranscriptItem[];
  onPrompt: (value: string) => void;
  onToggleReminder: (itemId: string, reminderId: string, active: boolean) => void;
}

export function MessageList({ items, onPrompt, onToggleReminder }: MessageListProps) {
  return (
    <main className="message-pane" aria-label="对话内容">
      {items.length === 0 ? (
        <WelcomeState onPrompt={onPrompt} />
      ) : (
        <div className="message-list">
          {items.map((item) => (
            <article key={item.id} className={`message-row ${item.role}`}>
              <div className="message-avatar" aria-hidden="true">
                {item.role === "user" ? (
                  <User size={16} />
                ) : item.role === "notice" ? (
                  <AlertTriangle size={16} />
                ) : item.role === "reminder" ? (
                  <Bell size={16} />
                ) : (
                  <Bot size={16} />
                )}
              </div>
              <div className={`message-bubble ${item.state ?? ""}`}>
                {item.image ? <img className="message-image" src={item.image} alt="用户上传的图片" /> : null}
                {item.role === "reminder" && item.reminder ? (
                  <ReminderCard item={item} onToggleReminder={onToggleReminder} />
                ) : item.role === "assistant" && item.state === "streaming" && item.content.trim() === "" ? (
                  <ThinkingState />
                ) : item.role === "assistant" && item.agent === "reporter" ? (
                  <MarkdownReport content={item.content} />
                ) : (
                  <p>{item.content}</p>
                )}
                {item.state === "streaming" && item.content.trim() !== "" ? <span className="typing-indicator" aria-label="正在生成" /> : null}
              </div>
            </article>
          ))}
        </div>
      )}
    </main>
  );
}

function ReminderCard({ item, onToggleReminder }: { item: TranscriptItem; onToggleReminder: (itemId: string, reminderId: string, active: boolean) => void }) {
  const reminder = item.reminder;
  const status =
    reminder?.status === "fired"
      ? "fired"
      : reminder?.status === "cancelled"
        ? "cancelled"
        : reminder?.status === "paused"
          ? "paused"
          : "scheduled";
  const title =
    status === "fired"
      ? "提醒时间到"
      : status === "cancelled"
        ? "定时任务已取消"
        : status === "paused"
          ? "定时任务已暂停"
          : "定时任务已创建";
  const fireTime = formatReminderTime(reminder?.fire_at);
  const canToggle = (status === "scheduled" || status === "paused") && Boolean(reminder?.id) && item.reminderAction !== "toggling";
  const nextActive = status === "paused";
  const actionLabel = nextActive ? "开启定时任务" : "关闭定时任务";

  return (
    <div className={`reminder-card ${status}`}>
      <button
        className="reminder-orbit"
        type="button"
        disabled={!canToggle}
        aria-label={canToggle ? actionLabel : title}
        title={canToggle ? actionLabel : title}
        onClick={() => {
          if (reminder?.id) onToggleReminder(item.id, reminder.id, nextActive);
        }}
      >
        {status === "paused" ? <BellOff size={18} /> : <Bell size={18} />}
      </button>
      <div className="reminder-main">
        <div className="reminder-heading">
          <span>{title}</span>
          {reminder?.recurring ? (
            <span className="reminder-badge">
              <Repeat2 size={12} />
              重复
            </span>
          ) : null}
        </div>
        <p>{reminder?.message || item.content}</p>
        <div className="reminder-meta">
          <Clock3 size={13} />
          <span>{item.reminderAction === "toggling" ? "正在更新" : item.reminderAction === "error" ? "更新失败" : fireTime}</span>
          {reminder?.cron ? <span>{reminder.cron}</span> : null}
        </div>
      </div>
    </div>
  );
}

function ThinkingState() {
  return (
    <div className="thinking-state" role="status" aria-live="polite" aria-label="思考中">
      <span className="thinking-label">思考中</span>
      <span className="thinking-dots" aria-hidden="true">
        <span />
        <span />
        <span />
      </span>
    </div>
  );
}

function formatReminderTime(unixSeconds?: number) {
  if (!unixSeconds) return "等待调度";
  const date = new Date(unixSeconds * 1000);
  if (Number.isNaN(date.getTime())) return "等待调度";
  return new Intl.DateTimeFormat("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}
