import { Bell, BellOff, Clock3, RefreshCw, Repeat2 } from "lucide-react";
import type { ReminderInfo } from "../types";

interface RemindersPanelProps {
  reminders: ReminderInfo[];
  loading: boolean;
  onRefresh: () => void;
  onToggle: (reminderId: string, active: boolean) => void;
}

export function RemindersPanel({ reminders, loading, onRefresh, onToggle }: RemindersPanelProps) {
  return (
    <div className="panel-section reminders-section">
      <div className="panel-heading">
        <Bell size={16} />
        <h2>当前定时任务</h2>
        <button className="icon-button ghost panel-action" type="button" onClick={onRefresh} aria-label="刷新定时任务">
          <RefreshCw className={loading ? "spin" : ""} size={15} />
        </button>
      </div>

      {reminders.length === 0 ? (
        <div className="empty-panel compact">
          <strong>{loading ? "正在加载" : "暂无定时任务"}</strong>
          <p>通过对话创建提醒后，会在这里显示。</p>
        </div>
      ) : (
        <div className="reminder-list">
          {reminders.map((reminder) => (
            <article className={`reminder-list-item ${reminder.status === "paused" ? "paused" : ""}`} key={reminder.id}>
              <div className="reminder-list-main">
                <strong>{reminder.message}</strong>
                <div className="reminder-meta">
                  <Clock3 size={13} />
                  <span>{formatReminderTime(reminder.fire_at)}</span>
                  {reminder.status === "paused" ? <span className="reminder-badge muted-badge">已暂停</span> : null}
                  {reminder.recurring ? (
                    <span className="reminder-badge">
                      <Repeat2 size={12} />
                      重复
                    </span>
                  ) : null}
                </div>
                {reminder.cron ? <small>{reminder.cron}</small> : null}
              </div>
              <button
                className={`icon-button ghost ${reminder.status === "paused" ? "" : "danger"}`}
                type="button"
                aria-label={reminder.status === "paused" ? "开启定时任务" : "关闭定时任务"}
                title={reminder.status === "paused" ? "开启定时任务" : "关闭定时任务"}
                onClick={() => {
                  if (reminder.id) onToggle(reminder.id, reminder.status === "paused");
                }}
              >
                {reminder.status === "paused" ? <BellOff size={15} /> : <Bell size={15} />}
              </button>
            </article>
          ))}
        </div>
      )}
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
