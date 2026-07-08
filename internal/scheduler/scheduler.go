package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
)

const (
	zsetScheduled = "reminders:scheduled"
	zsetRecurring = "reminders:recurring"
	hashPrefix    = "reminder:" // reminder:{id}
	threadPrefix  = "reminders:thread:"
	lockKey       = "reminders:lock"
	pubsubChannel = "reminders:events"
	pendingPrefix = "pending:"
)

// Reminder is the in-memory / on-wire representation of a reminder.
type Reminder struct {
	ID        string `json:"id"`
	ThreadID  string `json:"thread_id"`
	Message   string `json:"message"`
	FireAt    int64  `json:"fire_at"` // unix seconds
	Cron      string `json:"cron,omitempty"`
	Recurring bool   `json:"recurring"`
	Status    string `json:"status,omitempty"`
}

// Schedule adds a one-shot or recurring reminder to Redis.
func Schedule(ctx context.Context, rdb *redis.Client, r Reminder) error {
	if r.Status == "" {
		r.Status = "scheduled"
	}
	data, _ := json.Marshal(r)

	pipe := rdb.Pipeline()
	pipe.HSet(ctx, hashPrefix+r.ID, "data", string(data))
	pipe.SAdd(ctx, threadPrefix+r.ThreadID, r.ID)

	if r.Status == "scheduled" {
		if r.Recurring && r.Cron != "" {
			pipe.ZAdd(ctx, zsetRecurring, redis.Z{Score: float64(r.FireAt), Member: r.ID})
		} else {
			pipe.ZAdd(ctx, zsetScheduled, redis.Z{Score: float64(r.FireAt), Member: r.ID})
		}
	}

	_, err := pipe.Exec(ctx)
	return err
}

// Cancel removes a pending reminder.
func Cancel(ctx context.Context, rdb *redis.Client, threadID, reminderID string) error {
	r, err := load(ctx, rdb, reminderID)
	if err != nil {
		return err
	}
	if r.ThreadID != threadID {
		return fmt.Errorf("reminder %s not found for thread", reminderID)
	}
	pipe := rdb.Pipeline()
	pipe.ZRem(ctx, zsetScheduled, reminderID)
	pipe.ZRem(ctx, zsetRecurring, reminderID)
	pipe.Del(ctx, hashPrefix+reminderID)
	pipe.SRem(ctx, threadPrefix+threadID, reminderID)
	_, err = pipe.Exec(ctx)
	return err
}

// SetActive pauses or resumes a pending reminder without deleting it.
func SetActive(ctx context.Context, rdb *redis.Client, threadID, reminderID string, active bool) (Reminder, error) {
	r, err := load(ctx, rdb, reminderID)
	if err != nil {
		return r, err
	}
	if r.ThreadID != threadID {
		return r, fmt.Errorf("reminder %s not found for thread", reminderID)
	}

	if active {
		if r.Recurring && r.Cron != "" && r.FireAt <= time.Now().Unix() {
			sched, err := CronParser.Parse(r.Cron)
			if err != nil {
				return r, fmt.Errorf("invalid cron %q: %w", r.Cron, err)
			}
			next := sched.Next(reminderNow())
			if next.IsZero() {
				return r, fmt.Errorf("cron %q has no future fire time", r.Cron)
			}
			r.FireAt = next.Unix()
		}
		if !r.Recurring && r.FireAt <= time.Now().Unix() {
			return r, fmt.Errorf("reminder fire time has passed")
		}
		r.Status = "scheduled"
	} else {
		r.Status = "paused"
	}

	data, _ := json.Marshal(r)
	pipe := rdb.Pipeline()
	pipe.HSet(ctx, hashPrefix+r.ID, "data", string(data))
	pipe.SAdd(ctx, threadPrefix+r.ThreadID, r.ID)
	pipe.ZRem(ctx, zsetScheduled, r.ID)
	pipe.ZRem(ctx, zsetRecurring, r.ID)
	if active {
		if r.Recurring && r.Cron != "" {
			pipe.ZAdd(ctx, zsetRecurring, redis.Z{Score: float64(r.FireAt), Member: r.ID})
		} else {
			pipe.ZAdd(ctx, zsetScheduled, redis.Z{Score: float64(r.FireAt), Member: r.ID})
		}
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return r, err
	}
	return r, nil
}

// List returns all pending reminders for a thread.
func List(ctx context.Context, rdb *redis.Client, threadID string) ([]Reminder, error) {
	var out []Reminder
	seen := make(map[string]struct{})

	ids, err := rdb.SMembers(ctx, threadPrefix+threadID).Result()
	if err != nil {
		return nil, fmt.Errorf("smembers reminders: %w", err)
	}
	for _, id := range ids {
		r, err := load(ctx, rdb, id)
		if err != nil || r.ThreadID != threadID {
			continue
		}
		if r.Status == "" {
			r.Status = "scheduled"
		}
		out = append(out, r)
		seen[id] = struct{}{}
	}

	// Scan scheduled
	members, err := rdb.ZRangeByScore(ctx, zsetScheduled, &redis.ZRangeBy{
		Min: "-inf", Max: "+inf",
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("zrange scheduled: %w", err)
	}
	for _, id := range members {
		if _, ok := seen[id]; ok {
			continue
		}
		r, err := load(ctx, rdb, id)
		if err != nil || r.ThreadID != threadID {
			continue
		}
		if r.Status == "" {
			r.Status = "scheduled"
		}
		out = append(out, r)
		seen[id] = struct{}{}
	}

	// Scan recurring
	members, err = rdb.ZRangeByScore(ctx, zsetRecurring, &redis.ZRangeBy{
		Min: "-inf", Max: "+inf",
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("zrange recurring: %w", err)
	}
	for _, id := range members {
		if _, ok := seen[id]; ok {
			continue
		}
		r, err := load(ctx, rdb, id)
		if err != nil || r.ThreadID != threadID {
			continue
		}
		if r.Status == "" {
			r.Status = "scheduled"
		}
		out = append(out, r)
		seen[id] = struct{}{}
	}

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].FireAt < out[j].FireAt
	})
	return out, nil
}

func load(ctx context.Context, rdb *redis.Client, id string) (Reminder, error) {
	var r Reminder
	data, err := rdb.HGet(ctx, hashPrefix+id, "data").Result()
	if err != nil {
		return r, err
	}
	if err := json.Unmarshal([]byte(data), &r); err != nil {
		return r, err
	}
	return r, nil
}

// CronParser for validating cron expressions.
var CronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

func reminderLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.Local
	}
	return loc
}

func reminderNow() time.Time {
	return time.Now().In(reminderLocation())
}

// Worker polls Redis every second and fires due reminders.
type Worker struct {
	rdb      *redis.Client
	registry *ConnRegistry
	stopCh   chan struct{}
}

func NewWorker(rdb *redis.Client, reg *ConnRegistry) *Worker {
	return &Worker{rdb: rdb, registry: reg, stopCh: make(chan struct{})}
}

// Start creates and starts a Worker. Returns it for the caller to Stop() on shutdown.
func Start(ctx context.Context, rdb *redis.Client, reg *ConnRegistry) *Worker {
	w := NewWorker(rdb, reg)
	w.Start(ctx)
	log.Printf("[scheduler] worker started (redis zset, 1s tick)")
	return w
}

func (w *Worker) Start(ctx context.Context) {
	go w.loop(ctx)
	go w.listenBroadcast(ctx)
}

func (w *Worker) Stop() {
	close(w.stopCh)
}

func (w *Worker) loop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.fireDue(ctx)
		case <-w.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (w *Worker) fireDue(ctx context.Context) {
	now := float64(time.Now().Unix())

	// Grab and claim due scheduled reminders atomically
	ids, err := w.rdb.ZRangeByScore(ctx, zsetScheduled, &redis.ZRangeBy{
		Min: "0", Max: fmt.Sprintf("%.0f", now),
	}).Result()
	if err != nil || len(ids) == 0 {
		goto recurring
	}
	for _, id := range ids {
		if n, _ := w.rdb.ZRem(ctx, zsetScheduled, id).Result(); n == 0 {
			continue // claimed by another replica
		}
		r, err := load(ctx, w.rdb, id)
		if err != nil {
			continue
		}
		w.deliver(ctx, r)
		w.rdb.Del(ctx, hashPrefix+id)
	}

recurring:
	// Fire recurring reminders
	ids, err = w.rdb.ZRangeByScore(ctx, zsetRecurring, &redis.ZRangeBy{
		Min: "0", Max: fmt.Sprintf("%.0f", now),
	}).Result()
	if err != nil || len(ids) == 0 {
		return
	}
	for _, id := range ids {
		if n, _ := w.rdb.ZRem(ctx, zsetRecurring, id).Result(); n == 0 {
			continue
		}
		r, err := load(ctx, w.rdb, id)
		if err != nil {
			continue
		}
		w.deliver(ctx, r)

		// Re-schedule next fire
		if r.Cron != "" {
			sched, err := CronParser.Parse(r.Cron)
			if err != nil {
				w.rdb.Del(ctx, hashPrefix+id)
				continue
			}
			next := sched.Next(reminderNow())
			if next.IsZero() {
				w.rdb.Del(ctx, hashPrefix+id)
				continue
			}
			r.FireAt = next.Unix()
			data, _ := json.Marshal(r)
			w.rdb.HSet(ctx, hashPrefix+r.ID, "data", string(data))
			w.rdb.ZAdd(ctx, zsetRecurring, redis.Z{Score: float64(r.FireAt), Member: r.ID})
		} else {
			w.rdb.Del(ctx, hashPrefix+id)
		}
	}
}

func (w *Worker) deliver(ctx context.Context, r Reminder) {
	log.Printf("[scheduler] FIRE | thread=%s | %s", r.ThreadID, r.Message)

	event := eventFromReminder(r, "fired")
	data, err := pendingPayload(event)
	if err != nil {
		log.Printf("[scheduler] encode event: %v", err)
		return
	}
	if err := enqueuePending(ctx, w.rdb, event, data); err != nil {
		log.Printf("[scheduler] pending enqueue thread=%s: %v", r.ThreadID, err)
	}
	if err := w.rdb.Publish(ctx, pubsubChannel, data).Err(); err != nil {
		log.Printf("[scheduler] publish thread=%s: %v", r.ThreadID, err)
	}
}

func (w *Worker) listenBroadcast(ctx context.Context) {
	sub := w.rdb.Subscribe(ctx, pubsubChannel)
	defer sub.Close()

	ch := sub.Channel()
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			var event ReminderEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				log.Printf("[scheduler] pubsub decode: %v", err)
				continue
			}
			if w.registry.Push(event.ThreadID, event) {
				if err := ackPending(ctx, w.rdb, event, msg.Payload); err != nil {
					log.Printf("[scheduler] pending ack thread=%s: %v", event.ThreadID, err)
				}
			}
		case <-w.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// DrainPending delivers any pending notifications for a newly connected thread.
func DrainPending(ctx context.Context, rdb *redis.Client, threadID string, ch chan ReminderEvent) {
	key := pendingPrefix + threadID
	for {
		msg, err := rdb.RPop(ctx, key).Result()
		if err != nil {
			break
		}
		var event ReminderEvent
		if err := json.Unmarshal([]byte(msg), &event); err != nil {
			event = ReminderEvent{
				ThreadID: threadID,
				Message:  msg,
				Status:   "fired",
				FireAt:   time.Now().Unix(),
			}
		}
		select {
		case ch <- event:
		default:
		}
	}
}

func pendingPayload(event ReminderEvent) (string, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func enqueuePending(ctx context.Context, rdb *redis.Client, event ReminderEvent, payload string) error {
	key := pendingPrefix + event.ThreadID
	pipe := rdb.Pipeline()
	pipe.LPush(ctx, key, payload)
	pipe.Expire(ctx, key, 24*time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}

func ackPending(ctx context.Context, rdb *redis.Client, event ReminderEvent, payload string) error {
	return rdb.LRem(ctx, pendingPrefix+event.ThreadID, 1, payload).Err()
}
