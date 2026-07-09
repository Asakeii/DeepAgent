import type {
  AdminOverviewInfo,
  ChatEventPayload,
  ChatMessage,
  ReminderInfo,
  SessionInfo,
  StreamRequest,
  TeamInfo,
  TeamSettingsInfo,
} from "../types";

export interface StreamEvent {
  event: string;
  data: ChatEventPayload;
}

export async function listSessions(query = "", teamId?: string): Promise<SessionInfo[]> {
  const params = new URLSearchParams({ limit: "50" });
  if (query.trim()) params.set("q", query.trim());
  if (teamId !== undefined) params.set("team_id", teamId);
  const response = await fetch(`/api/sessions?${params.toString()}`);
  if (!response.ok) {
    throw new Error(`Failed to load sessions (${response.status})`);
  }
  const raw = await response.json();
  return raw.map((item: Record<string, unknown>) => ({
    threadId: String(item.thread_id ?? item.ThreadID ?? ""),
    teamId: String(item.team_id ?? item.TeamID ?? "") || undefined,
    firstMsg: String(item.first_msg ?? item.FirstMsg ?? ""),
    lastAt: String(item.last_at ?? item.LastAt ?? ""),
    msgCount: Number(item.msg_count ?? item.MsgCount ?? 0),
  }));
}

function mapTeam(item: Record<string, unknown>): TeamInfo {
  return {
    id: String(item.id ?? item.ID ?? ""),
    name: String(item.name ?? item.Name ?? ""),
    role: String(item.role ?? item.Role ?? "member"),
    createdAt: String(item.created_at ?? item.CreatedAt ?? "") || undefined,
    updatedAt: String(item.updated_at ?? item.UpdatedAt ?? "") || undefined,
  };
}

export async function listTeams(): Promise<TeamInfo[]> {
  const response = await fetch("/api/teams");
  if (!response.ok) {
    throw new Error(`Failed to load teams (${response.status})`);
  }
  const payload = await response.json();
  return ((payload.teams ?? []) as Array<Record<string, unknown>>).map(mapTeam);
}

export async function createTeam(name: string): Promise<TeamInfo> {
  const response = await fetch("/api/teams", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name }),
  });
  if (!response.ok) {
    throw new Error(`Create team failed (${response.status})`);
  }
  const payload = await response.json();
  return mapTeam(payload.team ?? {});
}

function mapTeamSettings(item: Record<string, unknown>): TeamSettingsInfo {
  const rawBudget = item.daily_cost_budget_micros ?? item.DailyCostBudgetMicros;
  return {
    teamId: String(item.team_id ?? item.TeamID ?? ""),
    dailyCostBudgetMicros: rawBudget === undefined || rawBudget === null ? undefined : Number(rawBudget),
    updatedBy: String(item.updated_by ?? item.UpdatedBy ?? "") || undefined,
    updatedAt: String(item.updated_at ?? item.UpdatedAt ?? "") || undefined,
  };
}

export async function loadTeamSettings(teamId: string): Promise<TeamSettingsInfo> {
  const params = new URLSearchParams({ team_id: teamId });
  const response = await fetch(`/api/team-settings?${params.toString()}`);
  if (!response.ok) {
    throw new Error(`Load team settings failed (${response.status})`);
  }
  const payload = await response.json();
  return mapTeamSettings(payload.settings ?? {});
}

export async function updateTeamSettings(teamId: string, dailyCostBudgetMicros: number): Promise<TeamSettingsInfo> {
  const response = await fetch("/api/team-settings", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      team_id: teamId,
      daily_cost_budget_micros: dailyCostBudgetMicros,
    }),
  });
  if (!response.ok) {
    throw new Error(`Update team settings failed (${response.status})`);
  }
  const payload = await response.json();
  return mapTeamSettings(payload.settings ?? {});
}

export async function loadMessages(threadId: string): Promise<ChatMessage[]> {
  const response = await fetch(`/api/messages?thread_id=${encodeURIComponent(threadId)}&limit=100`);
  if (!response.ok) {
    throw new Error(`Failed to load messages (${response.status})`);
  }
  const raw = await response.json();
  return raw.map((item: Record<string, unknown>) => ({
    role: String(item.role ?? item.Role ?? "assistant").toLowerCase(),
    content: String(item.content ?? item.Content ?? ""),
  })) as ChatMessage[];
}

export async function listReminders(threadId: string): Promise<ReminderInfo[]> {
  const response = await fetch(`/api/reminders?thread_id=${encodeURIComponent(threadId)}&limit=100`);
  if (!response.ok) {
    throw new Error(`Failed to load reminders (${response.status})`);
  }
  const payload = await response.json();
  return (payload.reminders ?? []) as ReminderInfo[];
}

export async function streamChat(
  request: StreamRequest,
  onEvent: (event: StreamEvent) => void,
  signal?: AbortSignal,
): Promise<void> {
  const response = await fetch("/chat/stream", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
    signal,
  });

  if (!response.ok || !response.body) {
    throw new Error(`Stream failed (${response.status})`);
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  while (true) {
    const { value, done } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });

    let boundary = buffer.indexOf("\n\n");
    while (boundary >= 0) {
      const block = buffer.slice(0, boundary).replace(/\r/g, "");
      buffer = buffer.slice(boundary + 2);
      const event = parseEventBlock(block);
      if (event) onEvent(event);
      boundary = buffer.indexOf("\n\n");
    }
  }

  const trailing = buffer.trim();
  if (trailing) {
    const event = parseEventBlock(trailing);
    if (event) onEvent(event);
  }
}

export async function cancelReminder(threadId: string, reminderId: string): Promise<ReminderInfo> {
  const response = await fetch("/api/reminders/cancel", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      thread_id: threadId,
      reminder_id: reminderId,
    }),
  });

  if (!response.ok) {
    throw new Error(`Cancel reminder failed (${response.status})`);
  }

  const payload = await response.json();
  return payload.reminder as ReminderInfo;
}

export async function toggleReminder(threadId: string, reminderId: string, active: boolean): Promise<ReminderInfo> {
  const response = await fetch("/api/reminders/toggle", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      thread_id: threadId,
      reminder_id: reminderId,
      active,
    }),
  });

  if (!response.ok) {
    throw new Error(`Toggle reminder failed (${response.status})`);
  }

  const payload = await response.json();
  return payload.reminder as ReminderInfo;
}

export async function loadAdminOverview(adminKey: string, windowHours: number): Promise<AdminOverviewInfo> {
  const params = new URLSearchParams({ window_hours: String(windowHours) });
  const response = await fetch(`/api/admin/overview?${params.toString()}`, {
    headers: {
      Authorization: `Bearer ${adminKey}`,
    },
  });

  if (!response.ok) {
    throw new Error(`Load admin overview failed (${response.status})`);
  }

  return (await response.json()) as AdminOverviewInfo;
}

function parseEventBlock(block: string): StreamEvent | null {
  let eventName = "message";
  const dataLines: string[] = [];

  for (const line of block.split("\n")) {
    if (line.startsWith("event:")) {
      eventName = line.slice(6).trim();
    } else if (line.startsWith("data:")) {
      dataLines.push(line.slice(5).trimStart());
    }
  }

  if (dataLines.length === 0) return null;

  try {
    return {
      event: eventName,
      data: JSON.parse(dataLines.join("\n")) as ChatEventPayload,
    };
  } catch {
    return null;
  }
}

export function newThreadId(): string {
  return `w-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
}
