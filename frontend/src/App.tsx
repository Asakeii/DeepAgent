import { useCallback, useEffect, useRef, useState } from "react";
import { Composer } from "./components/Composer";
import { MessageList } from "./components/MessageList";
import { PlanReview } from "./components/PlanReview";
import { RemindersPanel } from "./components/RemindersPanel";
import { ResearchPanel } from "./components/ResearchPanel";
import { SessionSidebar } from "./components/SessionSidebar";
import { TopBar } from "./components/TopBar";
import { extractUrls } from "./lib/format";
import { listReminders, listSessions, loadMessages, newThreadId, streamChat, toggleReminder, type StreamEvent } from "./lib/api";
import type { AgentName, ChatMessage, Plan, ReminderInfo, SessionInfo, ToolActivity, TranscriptItem } from "./types";

const ASSISTANT_PLACEHOLDER = "";
const HIDDEN_ASSISTANT_CONTENT = new Set(["end", "processed"]);

function isAssistantSentinel(content?: string) {
  const normalized = (content ?? "").trim().toLowerCase();
  return HIDDEN_ASSISTANT_CONTENT.has(normalized);
}

function isHiddenAssistantContent(content?: string) {
  return (content ?? "").trim() === "" || isAssistantSentinel(content);
}

export function App() {
  const [threadId, setThreadId] = useState(() => newThreadId());
  const [sessions, setSessions] = useState<SessionInfo[]>([]);
  const [sessionsLoading, setSessionsLoading] = useState(false);
  const [sidebarOpen, setSidebarOpen] = useState(() => window.matchMedia("(min-width: 900px)").matches);
  const [items, setItems] = useState<TranscriptItem[]>([]);
  const [input, setInput] = useState("");
  const [image, setImage] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);
  const [activeAgent, setActiveAgent] = useState<AgentName | undefined>();
  const [plan, setPlan] = useState<Plan | undefined>();
  const [planReviewEnabled, setPlanReviewEnabled] = useState(true);
  const [reviewOpen, setReviewOpen] = useState(false);
  const [tools, setTools] = useState<ToolActivity[]>([]);
  const [reminders, setReminders] = useState<ReminderInfo[]>([]);
  const [remindersLoading, setRemindersLoading] = useState(false);
  const assistantIdRef = useRef<string | null>(null);
  const abortRef = useRef<AbortController | null>(null);
  const suppressNextReminderMessageRef = useRef(false);

  const refreshSessions = useCallback(async () => {
    setSessionsLoading(true);
    try {
      setSessions(await listSessions());
    } catch {
      setSessions([]);
    } finally {
      setSessionsLoading(false);
    }
  }, []);

  useEffect(() => {
    void refreshSessions();
  }, [refreshSessions]);

  const refreshReminders = useCallback(async () => {
    setRemindersLoading(true);
    try {
      setReminders(await listReminders(threadId));
    } catch {
      setReminders([]);
    } finally {
      setRemindersLoading(false);
    }
  }, [threadId]);

  useEffect(() => {
    void refreshReminders();
  }, [refreshReminders]);

  const appendItem = useCallback((item: TranscriptItem) => {
    setItems((current) => [...current, item]);
  }, []);

  const appendReminderCard = useCallback((payload: StreamEvent["data"], status: "scheduled" | "fired") => {
    const reminder = payload.reminder ?? {
      message: payload.content ?? "提醒",
      status,
    };
    appendItem({
      id: payload.id ?? `${reminder.id ?? crypto.randomUUID()}-${status}-${reminder.fire_at ?? Date.now()}`,
      role: "reminder",
      content: reminder.message || payload.content || "提醒",
      state: "complete",
      reminder: {
        ...reminder,
        status: reminder.status || status,
      },
    });
  }, [appendItem]);

  const removePendingAssistant = useCallback(() => {
    const id = assistantIdRef.current;
    if (!id) return;
    setItems((current) => current.filter((item) => item.id !== id || item.content.trim() !== ""));
    assistantIdRef.current = null;
  }, []);

  const updateAssistant = useCallback((patch: Partial<TranscriptItem>) => {
    const id = assistantIdRef.current;
    if (!id) return;
    setItems((current) => current.map((item) => (item.id === id ? { ...item, ...patch } : item)));
  }, []);

  const completeAssistant = useCallback((payload: Partial<TranscriptItem>, fallbackAgent: AgentName) => {
    const id = assistantIdRef.current;
    if (!id) return;
    const nextContent = payload.content;
    const shouldHideContent = isHiddenAssistantContent(nextContent);
    setItems((current) =>
      current.map((item) => {
        if (item.id !== id) return item;
        const hasVisibleContent = item.content !== ASSISTANT_PLACEHOLDER && !isHiddenAssistantContent(item.content);
        return {
          ...item,
          ...payload,
          content: shouldHideContent ? (hasVisibleContent ? item.content : "处理完成。") : nextContent ?? item.content,
          state: "complete",
          agent: payload.agent ?? item.agent ?? fallbackAgent,
        };
      }),
    );
  }, []);

  const appendAssistantContent = useCallback((chunk: string, agent?: AgentName) => {
    if (!chunk || isAssistantSentinel(chunk)) return;
    const id = assistantIdRef.current;
    if (!id) return;
    setItems((current) =>
      current.map((item) => {
        if (item.id !== id) return item;
        const currentContent = item.content === ASSISTANT_PLACEHOLDER ? "" : item.content;
        return {
          ...item,
          content: currentContent + chunk,
          agent,
          state: "streaming",
        };
      }),
    );
  }, []);

  const resetConversationState = useCallback(() => {
    setActiveAgent(undefined);
    setPlan(undefined);
    setReviewOpen(false);
    setTools([]);
    assistantIdRef.current = null;
  }, []);

  const handleStreamEvent = useCallback(
    (event: StreamEvent) => {
      const payload = event.data;

      switch (event.event) {
        case "agent":
          setActiveAgent(payload.agent);
          return;

        case "plan":
          if (payload.plan) {
            setPlan(payload.plan);
            setActiveAgent("planner");
          }
          return;

        case "interrupt":
          setActiveAgent("human_feedback");
          setReviewOpen(true);
          setBusy(false);
          updateAssistant({ content: "研究计划已准备好，请确认后继续。", state: "complete", agent: "human_feedback" });
          return;

        case "tool_calls":
          handleToolCall(payload.tool_calls?.[0]?.name, payload.tool_call_chunks?.[0]?.args);
          return;

        case "tool_call_chunks":
          handleToolCall(payload.tool_call_chunks?.[0]?.name, payload.tool_call_chunks?.[0]?.args);
          return;

        case "tool_call_result":
          handleToolResult(payload.content);
          return;

        case "message_chunk":
          if (payload.agent && payload.agent !== "reporter" && payload.agent !== "coordinator") return;
          appendAssistantContent(payload.content ?? "", payload.agent);
          return;

        case "final_message":
          completeAssistant({
            content: payload.content ?? "",
            agent: payload.agent ?? "reporter",
          }, "reporter");
          setBusy(false);
          setActiveAgent(payload.agent);
          void refreshSessions();
          return;

        case "reminder_scheduled":
          removePendingAssistant();
          appendReminderCard(payload, "scheduled");
          void refreshReminders();
          suppressNextReminderMessageRef.current = true;
          setBusy(false);
          return;

        case "reminder":
          appendReminderCard(payload, "fired");
          return;

        case "message":
          if (suppressNextReminderMessageRef.current && payload.content?.includes("提醒")) {
            suppressNextReminderMessageRef.current = false;
            setBusy(false);
            void refreshSessions();
            return;
          }
          suppressNextReminderMessageRef.current = false;
          completeAssistant({
            content: payload.content ?? "",
            agent: payload.agent ?? "checkin",
          }, "checkin");
          setBusy(false);
          void refreshSessions();
          return;

        case "error":
          updateAssistant({
            content: payload.content || "处理出错，请稍后重试。",
            state: "error",
          });
          setBusy(false);
          return;

        default:
          return;
      }
    },
    [appendAssistantContent, appendReminderCard, completeAssistant, refreshReminders, refreshSessions, removePendingAssistant, updateAssistant],
  );

  const handleToolCall = useCallback((name?: string, rawArgs?: string) => {
    if (!name) return;
    let query = "";
    try {
      const parsed = rawArgs ? JSON.parse(rawArgs) : {};
      query = typeof parsed.query === "string" ? parsed.query : "";
    } catch {
      query = "";
    }
    const id = `${name}-${query || Date.now()}`;
    setTools((current) => {
      if (current.some((tool) => tool.id === id)) return current;
      return [
        ...current,
        {
          id,
          name,
          query,
          urls: [],
          status: "running",
        },
      ];
    });
  }, []);

  const handleToolResult = useCallback((content?: string) => {
    if (!content) return;
    const urls = extractUrls(content);
    setTools((current) => {
      if (current.length === 0) return current;
      const next = [...current];
      const last = next[next.length - 1];
      next[next.length - 1] = {
        ...last,
        status: "done",
        urls: urls.length > 0 ? urls : last.urls,
      };
      return next;
    });
  }, []);

  const mergeReminderState = useCallback((updated: ReminderInfo) => {
    setReminders((current) =>
      current.map((reminder) => (reminder.id === updated.id ? { ...reminder, ...updated } : reminder)),
    );
    setItems((current) =>
      current.map((item) =>
        item.reminder?.id === updated.id
          ? {
              ...item,
              reminderAction: undefined,
              reminder: {
                ...item.reminder,
                ...updated,
              },
            }
          : item,
      ),
    );
  }, []);

  const handleToggleReminder = useCallback(async (itemId: string, reminderId: string, active: boolean) => {
    setItems((current) =>
      current.map((item) => (item.id === itemId ? { ...item, reminderAction: "toggling" } : item)),
    );
    try {
      const reminder = await toggleReminder(threadId, reminderId, active);
      mergeReminderState(reminder);
    } catch {
      setItems((current) =>
        current.map((item) => (item.id === itemId ? { ...item, reminderAction: "error" } : item)),
      );
    }
  }, [mergeReminderState, threadId]);

  const handleToggleReminderFromPanel = useCallback(async (reminderId: string, active: boolean) => {
    try {
      const reminder = await toggleReminder(threadId, reminderId, active);
      mergeReminderState(reminder);
    } catch {
      void refreshReminders();
    }
  }, [mergeReminderState, refreshReminders, threadId]);

  async function runStream(request: {
    messages: ChatMessage[];
    interrupt_feedback?: "accepted" | "edit_plan";
    image_base64?: string;
  }) {
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    setBusy(true);

    try {
      await streamChat(
        {
          messages: request.messages,
          thread_id: threadId,
          auto_accepted_plan: !planReviewEnabled,
          interrupt_feedback: request.interrupt_feedback,
          image_base64: request.image_base64,
        },
        handleStreamEvent,
        controller.signal,
      );
    } catch (error) {
      if (controller.signal.aborted) return;
      updateAssistant({
        content: error instanceof Error ? error.message : "连接中断，请稍后重试。",
        state: "error",
      });
      setBusy(false);
    }
  }

  const sendPrompt = useCallback(
    async (prompt?: string) => {
      const content = (prompt ?? input).trim();
      if (busy || (!content && !image)) return;

      const assistantId = crypto.randomUUID();
      assistantIdRef.current = assistantId;
      appendItem({
        id: crypto.randomUUID(),
        role: "user",
        content: content || "分析这张图片",
        image,
      });
      appendItem({
        id: assistantId,
        role: "assistant",
        content: ASSISTANT_PLACEHOLDER,
        state: "streaming",
      });

      setInput("");
      setImage(undefined);
      setPlan(undefined);
      setReviewOpen(false);
      setTools([]);
      setActiveAgent("coordinator");

      await runStream({
        messages: [{ role: "user", content: content || "分析这张图片" }],
        image_base64: image,
      });
    },
    [appendItem, busy, image, input, planReviewEnabled, threadId],
  );

  const acceptPlan = useCallback(async () => {
    setReviewOpen(false);
    const assistantId = crypto.randomUUID();
    assistantIdRef.current = assistantId;
    appendItem({
      id: assistantId,
      role: "assistant",
      content: "已确认计划，继续执行研究。",
      state: "streaming",
      agent: "research_team",
    });
    await runStream({ messages: [], interrupt_feedback: "accepted" });
  }, [appendItem, planReviewEnabled, threadId]);

  const editPlan = useCallback(
    async (feedback: string) => {
      const trimmed = feedback.trim();
      if (!trimmed) return;
      setReviewOpen(false);
      appendItem({
        id: crypto.randomUUID(),
        role: "user",
        content: trimmed,
      });
      const assistantId = crypto.randomUUID();
      assistantIdRef.current = assistantId;
      appendItem({
        id: assistantId,
        role: "assistant",
        content: "收到反馈，正在重新规划。",
        state: "streaming",
        agent: "planner",
      });
      await runStream({
        messages: [{ role: "user", content: trimmed }],
        interrupt_feedback: "edit_plan",
      });
    },
    [appendItem, planReviewEnabled, threadId],
  );

  const newChat = useCallback(() => {
    abortRef.current?.abort();
    setThreadId(newThreadId());
    setItems([]);
    setInput("");
    setImage(undefined);
    resetConversationState();
    setReminders([]);
  }, [resetConversationState]);

  const selectSession = useCallback(
    async (nextThreadId: string) => {
      abortRef.current?.abort();
      setThreadId(nextThreadId);
      if (window.matchMedia("(max-width: 899px)").matches) {
        setSidebarOpen(false);
      }
      resetConversationState();
      setItems([]);
      setReminders([]);
      try {
        const messages = await loadMessages(nextThreadId);
        setItems(
          messages.map((message) => ({
            id: crypto.randomUUID(),
            role: message.role === "user" ? "user" : "assistant",
            content: message.content,
            state: "complete",
            agent: message.role === "assistant" ? "reporter" : undefined,
          })),
        );
      } catch {
        setItems([
          {
            id: crypto.randomUUID(),
            role: "notice",
            content: "历史消息加载失败。",
            state: "error",
          },
        ]);
      }
    },
    [resetConversationState],
  );

  return (
    <div className="app-shell">
      <SessionSidebar
        sessions={sessions}
        activeThreadId={threadId}
        isOpen={sidebarOpen}
        loading={sessionsLoading}
        onClose={() => setSidebarOpen(false)}
        onNew={newChat}
        onSelect={selectSession}
      />

      {sidebarOpen ? <button className="scrim" type="button" onClick={() => setSidebarOpen(false)} aria-label="关闭历史" /> : null}

      <div className="workspace">
        <TopBar
          activeAgent={activeAgent}
          sidebarOpen={sidebarOpen}
          planReviewEnabled={planReviewEnabled}
          busy={busy}
          onToggleSidebar={() => setSidebarOpen((value) => !value)}
          onNew={newChat}
          onTogglePlanReview={setPlanReviewEnabled}
        />

        <div className="workspace-body">
          <section className="conversation-column">
            <PlanReview
              plan={plan}
              open={reviewOpen}
              busy={busy}
              onAccept={acceptPlan}
              onEdit={editPlan}
              onDismiss={() => setReviewOpen(false)}
            />
            <MessageList items={items} onPrompt={(value) => void sendPrompt(value)} onToggleReminder={handleToggleReminder} />
            <Composer
              value={input}
              image={image}
              busy={busy}
              onChange={setInput}
              onImageChange={setImage}
              onSend={() => void sendPrompt()}
            />
          </section>

          <aside className="research-panel" aria-label="工作台侧栏">
            <RemindersPanel
              reminders={reminders}
              loading={remindersLoading}
              onRefresh={() => void refreshReminders()}
              onToggle={handleToggleReminderFromPanel}
            />
            <ResearchPanel activeAgent={activeAgent} plan={plan} tools={tools} />
          </aside>
        </div>
      </div>
    </div>
  );
}
