package agent

import "sync"

// CheckinThreads 是 Coordinator → handler 的跨包路由信号。
// Coordinator 路由到 hand_to_checkin 时将 threadID 写入此 map；
// handler 在 Stream 完成后 LoadAndDelete，判断是否需要切到 checkin agent。
var CheckinThreads sync.Map
