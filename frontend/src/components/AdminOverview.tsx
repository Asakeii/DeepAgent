import { Activity, BarChart3, Database, FileText, KeyRound, RefreshCw, Share2, Users } from "lucide-react";
import type { ReactNode } from "react";
import type { AdminOverviewInfo } from "../types";

interface AdminOverviewProps {
  adminKey: string;
  windowHours: number;
  overview?: AdminOverviewInfo;
  loading: boolean;
  error?: string;
  onAdminKeyChange: (value: string) => void;
  onWindowHoursChange: (value: number) => void;
  onRefresh: () => void;
}

export function AdminOverview({
  adminKey,
  windowHours,
  overview,
  loading,
  error,
  onAdminKeyChange,
  onWindowHoursChange,
  onRefresh,
}: AdminOverviewProps) {
  return (
    <section className="admin-page" aria-label="管理概览">
      <div className="admin-toolbar">
        <div>
          <h1>管理概览</h1>
          <p>跨用户运行、工具、模型与产物指标</p>
        </div>
        <div className="admin-controls">
          <label className="admin-key-field">
            <KeyRound size={15} />
            <input
              type="password"
              value={adminKey}
              placeholder="Admin API Key"
              onChange={(event) => onAdminKeyChange(event.target.value)}
            />
          </label>
          <select value={windowHours} onChange={(event) => onWindowHoursChange(Number(event.target.value))}>
            <option value={24}>24h</option>
            <option value={168}>7d</option>
            <option value={720}>30d</option>
          </select>
          <button className="secondary-button" type="button" onClick={onRefresh} disabled={loading || !adminKey.trim()}>
            <RefreshCw size={16} className={loading ? "spin" : ""} />
            刷新
          </button>
        </div>
      </div>

      {error ? <div className="admin-error">{error}</div> : null}

      {overview ? (
        <>
          <div className="admin-kpi-grid">
            <MetricCard icon={<Users size={18} />} label="用户" value={formatNumber(overview.users_total)} />
            <MetricCard icon={<Activity size={18} />} label="会话" value={formatNumber(overview.threads_total)} />
            <MetricCard icon={<FileText size={18} />} label="产物" value={formatNumber(overview.artifacts_total)} />
            <MetricCard icon={<Share2 size={18} />} label="分享" value={formatNumber(overview.artifact_shares)} />
          </div>

          <div className="admin-section-grid">
            <section className="admin-panel">
              <PanelTitle icon={<BarChart3 size={17} />} title={`Run · ${overview.window_hours}h`} />
              <MetricRows
                rows={[
                  ["总数", overview.runs_total],
                  ["成功", overview.runs_succeeded],
                  ["失败", overview.runs_failed],
                  ["运行中", overview.runs_running],
                  ["成功率", formatPercent(overview.run_success_rate)],
                ]}
              />
            </section>

            <section className="admin-panel">
              <PanelTitle icon={<Database size={17} />} title="Tool" />
              <MetricRows
                rows={[
                  ["总数", overview.tools_total],
                  ["失败", overview.tools_failed],
                  ["阻断", overview.tools_blocked],
                  ["错误率", formatPercent(overview.tool_error_rate)],
                ]}
              />
            </section>

            <section className="admin-panel wide">
              <PanelTitle icon={<Activity size={17} />} title="Token" />
              <div className="token-grid">
                <TokenStat label="Total" value={formatNumber(overview.total_tokens)} />
                <TokenStat label="Prompt" value={formatNumber(overview.prompt_tokens)} />
                <TokenStat label="Completion" value={formatNumber(overview.completion_tokens)} />
                <TokenStat label="Cached" value={formatNumber(overview.cached_tokens)} />
                <TokenStat label="Reasoning" value={formatNumber(overview.reasoning_tokens)} />
              </div>
            </section>
          </div>
        </>
      ) : (
        <div className="admin-empty">输入 Admin API Key 后刷新</div>
      )}
    </section>
  );
}

function MetricCard({
  icon,
  label,
  value,
}: {
  icon?: ReactNode;
  label: string;
  value: string;
}) {
  return (
    <div className="metric-card">
      <div className="metric-label">
        {icon}
        {label}
      </div>
      <strong>{value}</strong>
    </div>
  );
}

function TokenStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="token-stat">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function PanelTitle({ icon, title }: { icon: ReactNode; title: string }) {
  return (
    <div className="panel-heading">
      {icon}
      <h2>{title}</h2>
    </div>
  );
}

function MetricRows({ rows }: { rows: Array<[string, number | string]> }) {
  return (
    <div className="metric-rows">
      {rows.map(([label, value]) => (
        <div className="metric-row" key={label}>
          <span>{label}</span>
          <strong>{typeof value === "number" ? formatNumber(value) : value}</strong>
        </div>
      ))}
    </div>
  );
}

function formatNumber(value: number) {
  return new Intl.NumberFormat("zh-CN").format(value);
}

function formatPercent(value: number) {
  return `${Math.round(value * 1000) / 10}%`;
}
