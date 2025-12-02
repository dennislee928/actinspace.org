package ml

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sync"
	"time"
)

// CommandFeatures represents features extracted from a command for ML analysis
type CommandFeatures struct {
	Command       string
	Role          string
	HourOfDay     int
	DayOfWeek     int
	TimeSinceLast float64 // seconds
	CommandLength int
	HasParams     bool
}

// CommandHistory stores historical command data for training
type CommandHistory struct {
	Timestamp time.Time              `json:"timestamp"`
	Command   string                 `json:"command"`
	Role      string                 `json:"role"`
	Features  CommandFeatures        `json:"features"`
	Params    map[string]interface{} `json:"params,omitempty"`
}

// MLAnomalyDetector uses simple statistical methods for anomaly detection
// In production, this would integrate with actual ML frameworks (TensorFlow, PyTorch, etc.)
type MLAnomalyDetector struct {
	mu               sync.RWMutex
	history          []CommandHistory
	maxHistorySize   int
	commandBaselines map[string]*CommandBaseline
	roleBaselines    map[string]*RoleBaseline
	modelPath        string
}

// CommandBaseline stores statistical baseline for a command type
type CommandBaseline struct {
	Command         string
	Count           int
	AvgHourOfDay    float64
	StdHourOfDay    float64
	AvgTimeBetween  float64
	StdTimeBetween  float64
	TypicalRoles    map[string]int
	LastSeen        time.Time
}

// RoleBaseline stores statistical baseline for a role
type RoleBaseline struct {
	Role              string
	CommandsPerHour   float64
	TypicalCommands   map[string]int
	TypicalHours      map[int]int
	LastActivity      time.Time
}

// AnomalyScore represents the result of anomaly detection
type AnomalyScore struct {
	Score           float64
	IsAnomaly       bool
	Threshold       float64
	Reasons         []string
	Confidence      float64
	RecommendedAction string
}

// NewMLAnomalyDetector creates a new ML-based anomaly detector
func NewMLAnomalyDetector(modelPath string, maxHistory int) *MLAnomalyDetector {
	detector := &MLAnomalyDetector{
		history:          make([]CommandHistory, 0, maxHistory),
		maxHistorySize:   maxHistory,
		commandBaselines: make(map[string]*CommandBaseline),
		roleBaselines:    make(map[string]*RoleBaseline),
		modelPath:        modelPath,
	}

	// Load existing model/history if available
	detector.loadModel()

	return detector
}

// RecordCommand adds a command to the history for learning
func (d *MLAnomalyDetector) RecordCommand(cmd, role string, params map[string]interface{}) {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	features := d.extractFeatures(cmd, role, now, params)

	history := CommandHistory{
		Timestamp: now,
		Command:   cmd,
		Role:      role,
		Features:  features,
		Params:    params,
	}

	// Add to history with size limit
	d.history = append(d.history, history)
	if len(d.history) > d.maxHistorySize {
		d.history = d.history[1:]
	}

	// Update baselines
	d.updateBaselines(history)

	// Periodically save model
	if len(d.history)%100 == 0 {
		go d.saveModel()
	}
}

// DetectAnomaly analyzes a command and returns an anomaly score
func (d *MLAnomalyDetector) DetectAnomaly(cmd, role string, params map[string]interface{}) AnomalyScore {
	d.mu.RLock()
	defer d.mu.RUnlock()

	now := time.Now()
	features := d.extractFeatures(cmd, role, now, params)

	// Initialize score
	score := AnomalyScore{
		Score:     0.0,
		Threshold: 0.7, // Configurable threshold
		Reasons:   make([]string, 0),
	}

	// If insufficient history, return low confidence
	if len(d.history) < 10 {
		score.Confidence = 0.1
		score.IsAnomaly = false
		score.RecommendedAction = "collect_more_data"
		return score
	}

	// Compute anomaly scores from different perspectives
	commandScore := d.computeCommandAnomalyScore(features)
	roleScore := d.computeRoleAnomalyScore(features)
	temporalScore := d.computeTemporalAnomalyScore(features)
	frequencyScore := d.computeFrequencyAnomalyScore(features)

	// Weighted combination
	score.Score = 0.3*commandScore + 0.25*roleScore + 0.25*temporalScore + 0.2*frequencyScore
	score.IsAnomaly = score.Score > score.Threshold
	score.Confidence = d.computeConfidence()

	// Generate reasons
	if commandScore > 0.5 {
		score.Reasons = append(score.Reasons, fmt.Sprintf("unusual_command_pattern (score: %.2f)", commandScore))
	}
	if roleScore > 0.5 {
		score.Reasons = append(score.Reasons, fmt.Sprintf("unusual_role_behavior (score: %.2f)", roleScore))
	}
	if temporalScore > 0.5 {
		score.Reasons = append(score.Reasons, fmt.Sprintf("unusual_timing (score: %.2f)", temporalScore))
	}
	if frequencyScore > 0.5 {
		score.Reasons = append(score.Reasons, fmt.Sprintf("unusual_frequency (score: %.2f)", frequencyScore))
	}

	// Recommended action
	if score.IsAnomaly {
		if score.Score > 0.9 {
			score.RecommendedAction = "block_and_alert"
		} else if score.Score > 0.8 {
			score.RecommendedAction = "alert_and_log"
		} else {
			score.RecommendedAction = "log_for_review"
		}
	} else {
		score.RecommendedAction = "allow"
	}

	return score
}

// extractFeatures extracts features from a command for analysis
func (d *MLAnomalyDetector) extractFeatures(cmd, role string, timestamp time.Time, params map[string]interface{}) CommandFeatures {
	features := CommandFeatures{
		Command:       cmd,
		Role:          role,
		HourOfDay:     timestamp.Hour(),
		DayOfWeek:     int(timestamp.Weekday()),
		CommandLength: len(cmd),
		HasParams:     len(params) > 0,
	}

	// Find time since last command
	if len(d.history) > 0 {
		lastCmd := d.history[len(d.history)-1]
		features.TimeSinceLast = timestamp.Sub(lastCmd.Timestamp).Seconds()
	}

	return features
}

// computeCommandAnomalyScore checks if the command pattern is unusual
func (d *MLAnomalyDetector) computeCommandAnomalyScore(features CommandFeatures) float64 {
	baseline, exists := d.commandBaselines[features.Command]
	if !exists {
		// New command type - moderate anomaly
		return 0.6
	}

	score := 0.0

	// Check if role is typical for this command
	roleCount := baseline.TypicalRoles[features.Role]
	if roleCount == 0 {
		score += 0.4 // Unusual role for this command
	} else if float64(roleCount)/float64(baseline.Count) < 0.1 {
		score += 0.2 // Rare role for this command
	}

	// Check time-of-day deviation
	hourDiff := math.Abs(float64(features.HourOfDay) - baseline.AvgHourOfDay)
	if hourDiff > 24 {
		hourDiff = 48 - hourDiff // Handle wrap-around
	}
	if baseline.StdHourOfDay > 0 {
		zScore := hourDiff / baseline.StdHourOfDay
		if zScore > 2 {
			score += 0.3 // Unusual time
		} else if zScore > 1 {
			score += 0.15
		}
	}

	// Check time-between-commands deviation
	if features.TimeSinceLast > 0 && baseline.StdTimeBetween > 0 {
		zScore := math.Abs(features.TimeSinceLast-baseline.AvgTimeBetween) / baseline.StdTimeBetween
		if zScore > 2 {
			score += 0.3 // Unusual frequency
		}
	}

	return math.Min(score, 1.0)
}

// computeRoleAnomalyScore checks if the role's behavior is unusual
func (d *MLAnomalyDetector) computeRoleAnomalyScore(features CommandFeatures) float64 {
	baseline, exists := d.roleBaselines[features.Role]
	if !exists {
		// New role - moderate anomaly
		return 0.5
	}

	score := 0.0

	// Check if command is typical for this role
	cmdCount := baseline.TypicalCommands[features.Command]
	totalCmds := 0
	for _, count := range baseline.TypicalCommands {
		totalCmds += count
	}
	if cmdCount == 0 {
		score += 0.5 // Unusual command for this role
	} else if totalCmds > 0 && float64(cmdCount)/float64(totalCmds) < 0.05 {
		score += 0.25 // Rare command for this role
	}

	// Check if hour is typical for this role
	hourCount := baseline.TypicalHours[features.HourOfDay]
	if hourCount == 0 {
		score += 0.3 // Unusual hour for this role
	}

	return math.Min(score, 1.0)
}

// computeTemporalAnomalyScore checks for temporal anomalies
func (d *MLAnomalyDetector) computeTemporalAnomalyScore(features CommandFeatures) float64 {
	score := 0.0

	// Check for unusual hours (e.g., 2-5 AM)
	if features.HourOfDay >= 2 && features.HourOfDay <= 5 {
		score += 0.4
	}

	// Check for weekend activity (if typically weekday-only)
	if features.DayOfWeek == 0 || features.DayOfWeek == 6 {
		// Count weekend vs weekday commands
		weekendCount := 0
		weekdayCount := 0
		for _, h := range d.history {
			if h.Timestamp.Weekday() == 0 || h.Timestamp.Weekday() == 6 {
				weekendCount++
			} else {
				weekdayCount++
			}
		}
		if weekdayCount > 0 && float64(weekendCount)/float64(weekdayCount) < 0.1 {
			score += 0.3 // Unusual weekend activity
		}
	}

	return math.Min(score, 1.0)
}

// computeFrequencyAnomalyScore checks for unusual command frequency
func (d *MLAnomalyDetector) computeFrequencyAnomalyScore(features CommandFeatures) float64 {
	score := 0.0

	// Count recent commands (last 5 minutes)
	recentCount := 0
	fiveMinAgo := time.Now().Add(-5 * time.Minute)
	for i := len(d.history) - 1; i >= 0; i-- {
		if d.history[i].Timestamp.Before(fiveMinAgo) {
			break
		}
		recentCount++
	}

	// Check for burst
	if recentCount > 20 {
		score += 0.8
	} else if recentCount > 10 {
		score += 0.5
	} else if recentCount > 5 {
		score += 0.3
	}

	return math.Min(score, 1.0)
}

// updateBaselines updates statistical baselines with new data
func (d *MLAnomalyDetector) updateBaselines(history CommandHistory) {
	// Update command baseline
	baseline, exists := d.commandBaselines[history.Command]
	if !exists {
		baseline = &CommandBaseline{
			Command:      history.Command,
			TypicalRoles: make(map[string]int),
		}
		d.commandBaselines[history.Command] = baseline
	}

	baseline.Count++
	baseline.TypicalRoles[history.Role]++
	baseline.LastSeen = history.Timestamp

	// Update running average for hour of day
	baseline.AvgHourOfDay = (baseline.AvgHourOfDay*float64(baseline.Count-1) + float64(history.Features.HourOfDay)) / float64(baseline.Count)

	// Update role baseline
	roleBaseline, exists := d.roleBaselines[history.Role]
	if !exists {
		roleBaseline = &RoleBaseline{
			Role:            history.Role,
			TypicalCommands: make(map[string]int),
			TypicalHours:    make(map[int]int),
		}
		d.roleBaselines[history.Role] = roleBaseline
	}

	roleBaseline.TypicalCommands[history.Command]++
	roleBaseline.TypicalHours[history.Features.HourOfDay]++
	roleBaseline.LastActivity = history.Timestamp
}

// computeConfidence returns confidence in the anomaly detection
func (d *MLAnomalyDetector) computeConfidence() float64 {
	historySize := len(d.history)
	if historySize < 10 {
		return 0.1
	} else if historySize < 50 {
		return 0.5
	} else if historySize < 200 {
		return 0.7
	}
	return 0.9
}

// saveModel saves the current model to disk
func (d *MLAnomalyDetector) saveModel() error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.modelPath == "" {
		return nil // No model path configured
	}

	data := struct {
		History          []CommandHistory            `json:"history"`
		CommandBaselines map[string]*CommandBaseline `json:"command_baselines"`
		RoleBaselines    map[string]*RoleBaseline    `json:"role_baselines"`
	}{
		History:          d.history,
		CommandBaselines: d.commandBaselines,
		RoleBaselines:    d.roleBaselines,
	}

	file, err := os.Create(d.modelPath)
	if err != nil {
		return fmt.Errorf("failed to create model file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode model: %w", err)
	}

	return nil
}

// loadModel loads a saved model from disk
func (d *MLAnomalyDetector) loadModel() error {
	if d.modelPath == "" {
		return nil // No model path configured
	}

	file, err := os.Open(d.modelPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No existing model, start fresh
		}
		return fmt.Errorf("failed to open model file: %w", err)
	}
	defer file.Close()

	var data struct {
		History          []CommandHistory            `json:"history"`
		CommandBaselines map[string]*CommandBaseline `json:"command_baselines"`
		RoleBaselines    map[string]*RoleBaseline    `json:"role_baselines"`
	}

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return fmt.Errorf("failed to decode model: %w", err)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.history = data.History
	d.commandBaselines = data.CommandBaselines
	d.roleBaselines = data.RoleBaselines

	return nil
}

// GetStatistics returns current model statistics
func (d *MLAnomalyDetector) GetStatistics() map[string]interface{} {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return map[string]interface{}{
		"history_size":         len(d.history),
		"command_baselines":    len(d.commandBaselines),
		"role_baselines":       len(d.roleBaselines),
		"confidence":           d.computeConfidence(),
		"model_path":           d.modelPath,
	}
}

