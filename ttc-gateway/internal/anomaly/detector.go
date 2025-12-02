package anomaly

import (
	"fmt"
	"sync"
	"time"
)

// AnomalyType 定義異常類型。
type AnomalyType string

const (
	AnomalyTypeRateLimit    AnomalyType = "rate_limit"
	AnomalyTypeTimeOfDay    AnomalyType = "time_of_day"
	AnomalyTypeCommandBurst AnomalyType = "command_burst"
	AnomalyTypeUnusualRole  AnomalyType = "unusual_role"
)

// Anomaly 表示一個偵測到的異常。
type Anomaly struct {
	Type        AnomalyType
	Command     string
	OperatorRole string
	Message     string
	Severity    string // "low", "medium", "high", "critical"
	Timestamp   time.Time
	Metadata    map[string]interface{}
}

// Detector 是異常偵測器。
type Detector struct {
	mu sync.RWMutex

	// 指令計數器（按指令類型）
	commandCounts map[string][]time.Time

	// 操作者活動記錄
	operatorActivity map[string][]time.Time

	// 配置
	config Config
}

// Config 定義異常偵測的配置。
type Config struct {
	// 每種指令的最大頻率（每分鐘）
	MaxCommandsPerMinute map[string]int

	// 正常操作時間範圍（UTC）
	NormalHoursStart int // 小時 (0-23)
	NormalHoursEnd   int

	// 突發指令閾值（短時間內大量指令）
	BurstThreshold      int           // 指令數量
	BurstTimeWindow     time.Duration // 時間窗口
}

// NewDetector 創建新的異常偵測器。
func NewDetector(config Config) *Detector {
	if config.MaxCommandsPerMinute == nil {
		config.MaxCommandsPerMinute = map[string]int{
			"deorbit":       1,  // 每小時最多 1 次
			"orbit_change":  2,  // 每小時最多 2 次
			"payload_toggle": 10, // 每分鐘最多 10 次
			"default":       30, // 預設每分鐘最多 30 次
		}
	}
	if config.NormalHoursStart == 0 && config.NormalHoursEnd == 0 {
		config.NormalHoursStart = 8  // 08:00 UTC
		config.NormalHoursEnd = 20  // 20:00 UTC
	}
	if config.BurstThreshold == 0 {
		config.BurstThreshold = 10
		config.BurstTimeWindow = 10 * time.Second
	}

	return &Detector{
		commandCounts:    make(map[string][]time.Time),
		operatorActivity: make(map[string][]time.Time),
		config:           config,
	}
}

// CheckCommand 檢查指令是否異常。
func (d *Detector) CheckCommand(command string, operatorRole string, timestamp time.Time) []Anomaly {
	d.mu.Lock()
	defer d.mu.Unlock()

	var anomalies []Anomaly

	// 清理舊記錄（保留最近 5 分鐘）
	cutoff := timestamp.Add(-5 * time.Minute)
	d.cleanup(cutoff)

	// 檢查 1: 頻率限制
	if anomaly := d.checkRateLimit(command, timestamp); anomaly != nil {
		anomalies = append(anomalies, *anomaly)
	}

	// 檢查 2: 時間異常
	if anomaly := d.checkTimeOfDay(timestamp); anomaly != nil {
		anomalies = append(anomalies, *anomaly)
	}

	// 檢查 3: 指令突發
	if anomaly := d.checkCommandBurst(command, timestamp); anomaly != nil {
		anomalies = append(anomalies, *anomaly)
	}

	// 檢查 4: 異常角色活動
	if anomaly := d.checkUnusualRoleActivity(operatorRole, timestamp); anomaly != nil {
		anomalies = append(anomalies, *anomaly)
	}

	// 記錄此次指令
	d.recordCommand(command, operatorRole, timestamp)

	return anomalies
}

// checkRateLimit 檢查指令頻率是否超過限制。
func (d *Detector) checkRateLimit(command string, timestamp time.Time) *Anomaly {
	maxRate, exists := d.config.MaxCommandsPerMinute[command]
	if !exists {
		maxRate = d.config.MaxCommandsPerMinute["default"]
	}

	// 計算最近一分鐘內的指令數量
	oneMinuteAgo := timestamp.Add(-1 * time.Minute)
	count := 0
	for _, t := range d.commandCounts[command] {
		if t.After(oneMinuteAgo) {
			count++
		}
	}

	if count >= maxRate {
		return &Anomaly{
			Type:        AnomalyTypeRateLimit,
			Command:     command,
			Message:     fmt.Sprintf("command '%s' rate limit exceeded: %d commands in last minute (limit: %d)", command, count+1, maxRate),
			Severity:    "high",
			Timestamp:   timestamp,
			Metadata: map[string]interface{}{
				"count": count + 1,
				"limit": maxRate,
			},
		}
	}

	return nil
}

// checkTimeOfDay 檢查是否在異常時間執行指令。
func (d *Detector) checkTimeOfDay(timestamp time.Time) *Anomaly {
	hour := timestamp.UTC().Hour()
	
	// 檢查是否在正常時間範圍內
	inNormalHours := false
	if d.config.NormalHoursStart <= d.config.NormalHoursEnd {
		// 正常範圍在同一天（例如 8-20）
		inNormalHours = hour >= d.config.NormalHoursStart && hour < d.config.NormalHoursEnd
	} else {
		// 正常範圍跨日（例如 20-8）
		inNormalHours = hour >= d.config.NormalHoursStart || hour < d.config.NormalHoursEnd
	}

	if !inNormalHours {
		return &Anomaly{
			Type:      AnomalyTypeTimeOfDay,
			Message:   fmt.Sprintf("command executed outside normal hours (current: %02d:00 UTC, normal: %02d:00-%02d:00 UTC)", hour, d.config.NormalHoursStart, d.config.NormalHoursEnd),
			Severity:  "medium",
			Timestamp: timestamp,
			Metadata: map[string]interface{}{
				"hour": hour,
				"normalStart": d.config.NormalHoursStart,
				"normalEnd":   d.config.NormalHoursEnd,
			},
		}
	}

	return nil
}

// checkCommandBurst 檢查指令突發。
func (d *Detector) checkCommandBurst(command string, timestamp time.Time) *Anomaly {
	windowStart := timestamp.Add(-d.config.BurstTimeWindow)
	count := 0
	
	for cmd, times := range d.commandCounts {
		// 檢查所有指令類型（不僅是當前指令）
		for _, t := range times {
			if t.After(windowStart) {
				count++
			}
		}
	}

	if count >= d.config.BurstThreshold {
		return &Anomaly{
			Type:     AnomalyTypeCommandBurst,
			Command:  command,
			Message:  fmt.Sprintf("command burst detected: %d commands in last %v (threshold: %d)", count+1, d.config.BurstTimeWindow, d.config.BurstThreshold),
			Severity: "high",
			Timestamp: timestamp,
			Metadata: map[string]interface{}{
				"count":    count + 1,
				"threshold": d.config.BurstThreshold,
				"window":   d.config.BurstTimeWindow.String(),
			},
		}
	}

	return nil
}

// checkUnusualRoleActivity 檢查異常角色活動。
func (d *Detector) checkUnusualRoleActivity(operatorRole string, timestamp time.Time) *Anomaly {
	// 檢查該角色在短時間內是否有異常活動
	oneHourAgo := timestamp.Add(-1 * time.Hour)
	activityCount := 0
	
	for _, t := range d.operatorActivity[operatorRole] {
		if t.After(oneHourAgo) {
			activityCount++
		}
	}

	// 如果某個角色在非正常時間有大量活動，標記為異常
	hour := timestamp.UTC().Hour()
	if activityCount > 50 && (hour < 6 || hour > 22) {
		return &Anomaly{
			Type:        AnomalyTypeUnusualRole,
			OperatorRole: operatorRole,
			Message:     fmt.Sprintf("unusual activity for role '%s': %d commands in last hour during off-hours", operatorRole, activityCount),
			Severity:    "medium",
			Timestamp:   timestamp,
			Metadata: map[string]interface{}{
				"activityCount": activityCount,
				"hour":         hour,
			},
		}
	}

	return nil
}

// recordCommand 記錄指令執行。
func (d *Detector) recordCommand(command string, operatorRole string, timestamp time.Time) {
	d.commandCounts[command] = append(d.commandCounts[command], timestamp)
	d.operatorActivity[operatorRole] = append(d.operatorActivity[operatorRole], timestamp)
}

// cleanup 清理舊記錄。
func (d *Detector) cleanup(cutoff time.Time) {
	// 清理指令計數
	for cmd, times := range d.commandCounts {
		var filtered []time.Time
		for _, t := range times {
			if t.After(cutoff) {
				filtered = append(filtered, t)
			}
		}
		if len(filtered) == 0 {
			delete(d.commandCounts, cmd)
		} else {
			d.commandCounts[cmd] = filtered
		}
	}

	// 清理操作者活動記錄
	for role, times := range d.operatorActivity {
		var filtered []time.Time
		for _, t := range times {
			if t.After(cutoff) {
				filtered = append(filtered, t)
			}
		}
		if len(filtered) == 0 {
			delete(d.operatorActivity, role)
		} else {
			d.operatorActivity[role] = filtered
		}
	}
}

