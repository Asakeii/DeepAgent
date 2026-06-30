export type Role = "user" | "assistant" | "system" | "tool";

export interface ChatMessage {
  role: Role;
  content: string;
}

export interface SessionInfo {
  threadId: string;
  firstMsg: string;
  lastAt: string;
  msgCount: number;
}

export interface PlanStep {
  need_web_search: boolean;
  title: string;
  description: string;
  step_type: "research" | "processing";
  execution_res?: string;
}

export interface Plan {
  locale: string;
  has_enough_context: boolean;
  thought: string;
  title: string;
  steps: PlanStep[];
}

export interface ToolCall {
  id?: string;
  type?: string;
  name?: string;
  args?: Record<string, unknown>;
}

export interface ToolCallChunk {
  id?: string;
  type?: string;
  name?: string;
  args?: string;
}

export interface ChatEventPayload {
  thread_id?: string;
  agent?: string;
  id?: string;
  role?: Role;
  content?: string;
  reminder?: ReminderInfo;
  plan?: Plan;
  finish_reason?: string;
  options?: Array<Record<string, unknown>>;
  tool_call_id?: string;
  tool_calls?: ToolCall[];
  tool_call_chunks?: ToolCallChunk[];
}

export interface StreamRequest {
  messages: ChatMessage[];
  thread_id: string;
  auto_accepted_plan: boolean;
  interrupt_feedback?: "accepted" | "edit_plan";
  image_base64?: string;
}

export type AgentName =
  | "coordinator"
  | "planner"
  | "human_feedback"
  | "research_team"
  | "researcher"
  | "coder"
  | "reporter"
  | "background_investigator"
  | "checkin"
  | string;

export interface ToolActivity {
  id: string;
  name: string;
  query?: string;
  urls: string[];
  status: "running" | "done";
}

export interface ReminderInfo {
  id?: string;
  thread_id?: string;
  message: string;
  fire_at?: number;
  cron?: string;
  recurring?: boolean;
  status: "scheduled" | "paused" | "fired" | "cancelled" | string;
}

export interface TranscriptItem {
  id: string;
  role: "user" | "assistant" | "notice" | "reminder";
  content: string;
  image?: string;
  state?: "streaming" | "complete" | "error";
  agent?: AgentName;
  reminder?: ReminderInfo;
  reminderAction?: "toggling" | "cancelling" | "error";
}
