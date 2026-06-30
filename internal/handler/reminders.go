package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	"deepAgent/internal/scheduler"
)

func ListReminders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if infra.RDB == nil {
		http.Error(w, "reminders disabled", http.StatusServiceUnavailable)
		return
	}
	threadID := r.URL.Query().Get("thread_id")
	if threadID == "" {
		http.Error(w, "thread_id required", http.StatusBadRequest)
		return
	}

	reminders, err := scheduler.List(r.Context(), infra.RDB, threadID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]*model.ReminderResp, 0, len(reminders))
	for _, reminder := range reminders {
		out = append(out, &model.ReminderResp{
			ID:        reminder.ID,
			ThreadID:  reminder.ThreadID,
			Message:   reminder.Message,
			FireAt:    reminder.FireAt,
			Cron:      reminder.Cron,
			Recurring: reminder.Recurring,
			Status:    reminder.Status,
		})
	}
	limit := len(out)
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 && n < limit {
			limit = n
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(model.ListRemindersResponse{Reminders: out[:limit]})
}

func CancelReminder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if infra.RDB == nil {
		http.Error(w, "reminders disabled", http.StatusServiceUnavailable)
		return
	}

	var req model.CancelReminderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.ThreadID == "" || req.ReminderID == "" {
		http.Error(w, "thread_id and reminder_id required", http.StatusBadRequest)
		return
	}

	if err := scheduler.Cancel(r.Context(), infra.RDB, req.ThreadID, req.ReminderID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := model.CancelReminderResponse{
		Reminder: &model.ReminderResp{
			ID:       req.ReminderID,
			ThreadID: req.ThreadID,
			Status:   "cancelled",
		},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func ToggleReminder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if infra.RDB == nil {
		http.Error(w, "reminders disabled", http.StatusServiceUnavailable)
		return
	}

	var req model.ToggleReminderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.ThreadID == "" || req.ReminderID == "" {
		http.Error(w, "thread_id and reminder_id required", http.StatusBadRequest)
		return
	}

	reminder, err := scheduler.SetActive(r.Context(), infra.RDB, req.ThreadID, req.ReminderID, req.Active)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := model.ToggleReminderResponse{
		Reminder: &model.ReminderResp{
			ID:        reminder.ID,
			ThreadID:  reminder.ThreadID,
			Message:   reminder.Message,
			FireAt:    reminder.FireAt,
			Cron:      reminder.Cron,
			Recurring: reminder.Recurring,
			Status:    reminder.Status,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
