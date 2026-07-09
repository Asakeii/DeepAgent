import { Building2, Check, CircleDollarSign, Loader2, Settings, UsersRound } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import type { TeamInfo, TeamSettingsInfo } from "../types";

interface TeamSwitcherProps {
  teams: TeamInfo[];
  activeTeamId: string;
  settings?: TeamSettingsInfo;
  loading: boolean;
  saving: boolean;
  error?: string;
  onScopeChange: (teamId: string) => void;
  onCreateTeam: (name: string) => Promise<void>;
  onSaveBudget: (teamId: string, budgetMicros: number) => Promise<void>;
}

function budgetToInput(settings?: TeamSettingsInfo) {
  if (!settings?.dailyCostBudgetMicros) return "";
  return (settings.dailyCostBudgetMicros / 1_000_000).toFixed(2);
}

export function TeamSwitcher({
  teams,
  activeTeamId,
  settings,
  loading,
  saving,
  error,
  onScopeChange,
  onCreateTeam,
  onSaveBudget,
}: TeamSwitcherProps) {
  const [open, setOpen] = useState(false);
  const [teamName, setTeamName] = useState("");
  const [budgetValue, setBudgetValue] = useState(() => budgetToInput(settings));
  const [localError, setLocalError] = useState<string | undefined>();

  const activeTeam = useMemo(() => teams.find((team) => team.id === activeTeamId), [activeTeamId, teams]);
  const canManage = activeTeam?.role === "owner" || activeTeam?.role === "admin";

  useEffect(() => {
    setBudgetValue(budgetToInput(settings));
  }, [settings]);

  async function submitTeam(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const name = teamName.trim();
    if (!name) return;
    setLocalError(undefined);
    try {
      await onCreateTeam(name);
      setTeamName("");
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : "创建团队失败");
    }
  }

  async function submitBudget(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!activeTeam) return;
    const trimmed = budgetValue.trim();
    const amount = trimmed ? Number(trimmed) : 0;
    if (!Number.isFinite(amount) || amount < 0) {
      setLocalError("预算金额无效");
      return;
    }
    setLocalError(undefined);
    try {
      await onSaveBudget(activeTeam.id, Math.round(amount * 1_000_000));
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : "预算保存失败");
    }
  }

  return (
    <div className="team-switcher">
      <label className="team-select-wrap">
        <UsersRound size={15} />
        <select value={activeTeamId} onChange={(event) => onScopeChange(event.target.value)} aria-label="空间">
          <option value="">个人空间</option>
          {teams.map((team) => (
            <option key={team.id} value={team.id}>
              {team.name}
            </option>
          ))}
        </select>
      </label>

      <button
        className={`icon-button ghost ${open ? "active" : ""}`}
        type="button"
        onClick={() => setOpen((value) => !value)}
        aria-label="团队设置"
        title="团队设置"
      >
        {loading ? <Loader2 className="spin" size={16} /> : <Settings size={16} />}
      </button>

      {open ? (
        <div className="team-popover">
          <div className="team-popover-head">
            <Building2 size={16} />
            <strong>{activeTeam ? activeTeam.name : "个人空间"}</strong>
            {activeTeam ? <span>{activeTeam.role}</span> : null}
          </div>

          {activeTeam ? (
            <form className="team-budget-form" onSubmit={submitBudget}>
              <label>
                <span>每日预算 USD</span>
                <div className="budget-input-row">
                  <CircleDollarSign size={15} />
                  <input
                    value={budgetValue}
                    inputMode="decimal"
                    placeholder="0.00"
                    disabled={!canManage || saving}
                    onChange={(event) => setBudgetValue(event.target.value)}
                    aria-label="团队每日预算 USD"
                  />
                </div>
              </label>
              <button className="primary-button compact" type="submit" disabled={!canManage || saving}>
                {saving ? <Loader2 className="spin" size={15} /> : <Check size={15} />}
                保存
              </button>
            </form>
          ) : null}

          <form className="team-create-form" onSubmit={submitTeam}>
            <input
              value={teamName}
              placeholder="新团队名称"
              maxLength={128}
              disabled={saving}
              onChange={(event) => setTeamName(event.target.value)}
              aria-label="新团队名称"
            />
            <button className="secondary-button compact" type="submit" disabled={saving || !teamName.trim()}>
              创建
            </button>
          </form>

          {error || localError ? <div className="team-error">{localError ?? error}</div> : null}
        </div>
      ) : null}
    </div>
  );
}
