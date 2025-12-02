# Threat Library

本目錄包含 Space Cyber Resilience Platform 的威脅場景定義和重演腳本。

## 場景定義

所有場景定義在 `scenarios/` 目錄中，使用 YAML 格式。每個場景包含：

- **ID**: 唯一識別碼
- **Name**: 場景名稱
- **Description**: 詳細描述
- **Objectives**: 攻擊目標
- **Tactics**: 對應的 SPARTA / MITRE ATT&CK 戰術
- **Expected Observables**: 預期的可觀察指標
- **Playbook Steps**: 執行步驟
- **Mitigations**: 防禦措施
- **Severity**: 嚴重性等級

## 已定義場景

1. **malicious-ota-update** - 惡意 OTA 更新嘗試
2. **unauthorized-dangerous-command** - 未授權危險指令
3. **uplink-spoofing-flood** - Uplink 偽造/洪水攻擊
4. **ground-it-compromise** - 地面 IT 系統入侵並轉向 TT&C
5. **critical-phase-violation** - 關鍵任務階段違規

## 重演場景

使用 `scripts/replay-scenario.go` 來重演威脅場景：

```bash
# 編譯重演工具
go build -o replay-scenario ./threat-library/scripts/replay-scenario.go

# 重演未授權危險指令場景
./replay-scenario -scenario threat-library/scenarios/unauthorized-dangerous-command.yaml \
  -gateway http://localhost:8081 \
  -token operator-token

# 重演 uplink flood 場景
./replay-scenario -scenario threat-library/scenarios/uplink-spoofing-flood.yaml \
  -gateway http://localhost:8081 \
  -delay 1s
```

## 場景格式

每個場景 YAML 檔案遵循以下結構：

```yaml
id: scenario-id
name: Scenario Name
description: |
  Detailed description of the attack scenario
objectives:
  - Objective 1
  - Objective 2
assumed_attacker:
  type: Attacker Type
  capabilities:
    - Capability 1
tactics:
  - SPARTA: T0001
  - MITRE ATT&CK: T1078
expected_observables:
  - Observable 1
  - Observable 2
playbook_steps:
  - Step 1
  - Step 2
mitigations:
  - Mitigation 1
severity: high
```

## 與 Space-SOC 整合

當場景被重演時：

1. 相關事件會自動標記 `scenarioID`
2. Space-SOC 可以依場景 ID 查詢相關事件
3. 高嚴重性事件會自動創建 Incident
4. Frontend 可以顯示場景關聯的事件時間軸

## 擴展場景

要添加新場景：

1. 在 `scenarios/` 目錄創建新的 YAML 檔案
2. 在 `scripts/replay-scenario.go` 中添加對應的重演邏輯（如果需要自動化）
3. 更新本文檔的場景列表
