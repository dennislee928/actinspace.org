package policy

import (
	"fmt"
	"time"
)

// PolicyDecision 定義 policy 引擎的決策結果。
type PolicyDecision struct {
	Allowed   bool
	Reason    string
	RuleID    string
	Severity  string // "low", "medium", "high", "critical"
}

// CommandContext 包含評估 policy 所需的上下文。
type CommandContext struct {
	Command      string
	OperatorRole string
	SatelliteID  string
	MissionPhase string // "normal", "critical", "safe_mode", "maintenance"
	TimeOfDay    time.Time
}

// Engine 是 policy 引擎的主要結構。
type Engine struct {
	rules []Rule
}

// Rule 定義單一 policy 規則。
type Rule struct {
	ID          string
	Description string
	Condition   func(ctx CommandContext) bool
	Action      func(ctx CommandContext) PolicyDecision
}

// NewEngine 創建新的 policy 引擎。
func NewEngine() *Engine {
	engine := &Engine{
		rules: []Rule{},
	}
	engine.loadDefaultRules()
	return engine
}

// Evaluate 評估指令是否符合 policy。
func (e *Engine) Evaluate(ctx CommandContext) PolicyDecision {
	// 按順序評估所有規則
	for _, rule := range e.rules {
		if rule.Condition(ctx) {
			decision := rule.Action(ctx)
			decision.RuleID = rule.ID
			return decision
		}
	}

	// 預設允許
	return PolicyDecision{
		Allowed:  true,
		Reason:   "no matching policy rule, default allow",
		RuleID:   "default-allow",
		Severity: "low",
	}
}

// loadDefaultRules 載入預設的 policy 規則。
func (e *Engine) loadDefaultRules() {
	// 規則 1: 危險指令需要 admin 角色
	e.rules = append(e.rules, Rule{
		ID:          "dangerous-command-admin-only",
		Description: "危險指令僅允許 admin 角色執行",
		Condition: func(ctx CommandContext) bool {
			dangerousCommands := map[string]bool{
				"deorbit":       true,
				"disable_power": true,
				"format_memory": true,
				"orbit_change":  true,
			}
			return dangerousCommands[ctx.Command]
		},
		Action: func(ctx CommandContext) PolicyDecision {
			if ctx.OperatorRole != "admin" {
				return PolicyDecision{
					Allowed:  false,
					Reason:   fmt.Sprintf("command '%s' requires admin role, got '%s'", ctx.Command, ctx.OperatorRole),
					Severity: "high",
				}
			}
			return PolicyDecision{
				Allowed:  true,
				Reason:   fmt.Sprintf("admin role authorized for dangerous command '%s'", ctx.Command),
				Severity: "high",
			}
		},
	})

	// 規則 2: 關鍵任務階段限制
	e.rules = append(e.rules, Rule{
		ID:          "critical-phase-restrictions",
		Description: "關鍵任務階段限制非關鍵指令",
		Condition: func(ctx CommandContext) bool {
			return ctx.MissionPhase == "critical"
		},
		Action: func(ctx CommandContext) PolicyDecision {
			// 在關鍵階段，只允許關鍵指令
			criticalCommands := map[string]bool{
				"emergency_safe_mode": true,
				"health_check":        true,
			}
			if !criticalCommands[ctx.Command] && ctx.OperatorRole != "admin" {
				return PolicyDecision{
					Allowed:  false,
					Reason:   fmt.Sprintf("mission phase '%s' restricts non-critical commands", ctx.MissionPhase),
					Severity: "medium",
				}
			}
			return PolicyDecision{
				Allowed:  true,
				Reason:   "command allowed in critical phase",
				Severity: "medium",
			}
		},
	})

	// 規則 3: 安全模式限制
	e.rules = append(e.rules, Rule{
		ID:          "safe-mode-restrictions",
		Description: "安全模式僅允許基本操作",
		Condition: func(ctx CommandContext) bool {
			return ctx.MissionPhase == "safe_mode"
		},
		Action: func(ctx CommandContext) PolicyDecision {
			allowedInSafeMode := map[string]bool{
				"health_check":        true,
				"exit_safe_mode":     true,
				"emergency_safe_mode": true,
			}
			if !allowedInSafeMode[ctx.Command] {
				return PolicyDecision{
					Allowed:  false,
					Reason:   fmt.Sprintf("command '%s' not allowed in safe mode", ctx.Command),
					Severity: "high",
				}
			}
			return PolicyDecision{
				Allowed:  true,
				Reason:   "command allowed in safe mode",
				Severity: "medium",
			}
		},
	})

	// 規則 4: 工程師角色限制
	e.rules = append(e.rules, Rule{
		ID:          "engineer-role-restrictions",
		Description: "工程師角色僅允許維護相關指令",
		Condition: func(ctx CommandContext) bool {
			return ctx.OperatorRole == "engineer"
		},
		Action: func(ctx CommandContext) PolicyDecision {
			engineerCommands := map[string]bool{
				"health_check":     true,
				"diagnostics":      true,
				"system_status":    true,
				"payload_toggle":   true,
				"maintenance_mode": true,
			}
			if !engineerCommands[ctx.Command] {
				return PolicyDecision{
					Allowed:  false,
					Reason:   fmt.Sprintf("engineer role not authorized for command '%s'", ctx.Command),
					Severity: "medium",
				}
			}
			return PolicyDecision{
				Allowed:  true,
				Reason:   "engineer role authorized",
				Severity: "low",
			}
		},
	})
}

