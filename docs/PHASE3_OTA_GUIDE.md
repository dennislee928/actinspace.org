# Phase 3 - OTA 與 SBOM 實作指南

本文檔說明 Phase 3 實作的 OTA（Over-The-Air）更新流程與 SBOM policy 檢查。

## 架構概覽

```
CI Pipeline → OTA Controller → Satellite-sim
                ↓                    ↓
           Space-SOC ← ─ ─ ─ ─ ─ ─ ─ ┘
```

## 組件說明

### 1. OTA Controller (`supply-chain/ota-controller/`)

**職責**：
- 管理軟體版本發布
- 批准工作流（pending → approved）
- 任務政策執行（關鍵階段禁止更新）
- 整合 SBOM policy 檢查

**API**：
- `POST /api/v1/releases` - 註冊新版本
- `POST /api/v1/releases/:id/approve` - 批准版本
- `POST /api/v1/updates/check` - 檢查可用更新
- `GET /api/v1/releases` - 查詢版本列表

### 2. OTA Client (`satellite-sim/internal/ota/`)

**職責**：
- 週期性檢查更新（預設 30 秒）
- 下載並驗證更新
- 簽章驗證
- 應用更新（模擬）

**流程**：
1. 向 OTA Controller 查詢更新
2. 檢查是否有新版本
3. 驗證簽章和 attestation
4. 下載並應用更新
5. 記錄事件到 Space-SOC

### 3. SBOM Parser (`supply-chain/sbom/`)

**職責**：
- 解析 CycloneDX SBOM
- 執行 policy 檢查
- 偵測已知漏洞
- 檢查授權限制

**Policy 規則**：
1. **已知漏洞檢查**：拒絕包含已知 CVE 的套件
2. **授權限制**：拒絕特定授權（如 AGPL-3.0）
3. **依賴數量**：警告異常大量依賴（> 500）

### 4. Software Posture (`space-soc/backend/`)

**職責**：
- 追蹤所有組件的當前版本
- 記錄已知漏洞數量
- 顯示更新可用性
- 提供軟體姿態儀表板

**資料模型**：
- Component: 組件名稱
- CurrentVersion: 當前版本
- VulnCount: 已知漏洞數量
- LastScanTime: 最後掃描時間
- UpdateAvailable: 是否有可用更新

## 完整更新流程

### 正常更新流程

1. **CI Pipeline 建置**
   ```bash
   # 建置映像檔
   docker build -t satellite-sim:v1.1.0 .
   
   # 產生 SBOM
   syft satellite-sim:v1.1.0 -o cyclonedx-json > sbom.json
   
   # 簽章
   ./sign-artifact satellite-sim:v1.1.0 > attestation.json
   ```

2. **註冊到 OTA Controller**
   ```bash
   curl -X POST http://ota-controller:8084/api/v1/releases \
     -d '{
       "component": "satellite-sim",
       "version": "v1.1.0",
       "imageDigest": "sha256:...",
       "sbomUrl": "...",
       "attestation": "..."
     }'
   ```

3. **人工批准**
   ```bash
   curl -X POST http://ota-controller:8084/api/v1/releases/1/approve
   ```

4. **衛星自動更新**
   - Satellite-sim 每 30 秒檢查一次
   - 發現新版本後自動下載
   - 驗證簽章
   - 應用更新
   - 記錄到 Space-SOC

### 惡意更新防禦

**場景**：攻擊者嘗試注入惡意更新

1. **簽章驗證失敗**
   - OTA client 驗證簽章
   - 發現不匹配，拒絕更新
   - 事件記錄到 Space-SOC

2. **SBOM Policy 違規**
   - 檢測到已知漏洞套件
   - Policy 檢查失敗
   - 更新被阻止

3. **任務階段限制**
   - 關鍵任務階段禁止更新
   - OTA Controller 拒絕分發
   - 記錄拒絕原因

## 測試

### 測試 SBOM Policy

```bash
# 檢查 SBOM
go run supply-chain/sbom/cmd/check-sbom/main.go \
  -sbom supply-chain/sbom/examples/satellite-sim-v1.0.0.cdx.json

# JSON 輸出
go run supply-chain/sbom/cmd/check-sbom/main.go \
  -sbom supply-chain/sbom/examples/satellite-sim-v1.0.0.cdx.json \
  -json
```

### 測試 OTA 流程

1. 啟動服務
   ```bash
   docker compose -f infra/docker-compose.yaml up -d
   ```

2. 註冊新版本
   ```bash
   curl -X POST http://localhost:8084/api/v1/releases \
     -H "Content-Type: application/json" \
     -d '{
       "component": "satellite-sim",
       "version": "v1.1.0",
       "imageDigest": "sha256:test123",
       "attestation": "{\"digest\":\"sha256:test123\",\"signature\":\"...\"}"
     }'
   ```

3. 批准版本
   ```bash
   curl -X POST http://localhost:8084/api/v1/releases/1/approve
   ```

4. 觀察 satellite-sim 日誌
   ```bash
   docker compose -f infra/docker-compose.yaml logs -f satellite-sim
   ```

5. 檢查 Space-SOC
   - 訪問 http://localhost:3001
   - 查看更新相關事件
   - 檢查軟體姿態頁面

## Space-SOC 整合

### 軟體姿態儀表板

訪問 http://localhost:3001/posture 查看：

- 所有組件的當前版本
- 已知漏洞數量
- 可用更新
- 最後掃描/更新時間

### 事件類型

- `release_registered`: 新版本註冊
- `release_approved`: 版本批准
- `update_check`: 衛星檢查更新
- `update_applied`: 更新應用成功
- `update_denied`: 更新被拒絕
- `signature_verification_failed`: 簽章驗證失敗

## 安全最佳實踐

1. **簽章驗證**：所有更新必須有有效簽章
2. **SBOM Policy**：自動檢查已知漏洞和授權
3. **批准工作流**：人工審查關鍵更新
4. **任務政策**：關鍵階段自動阻止更新
5. **審計追蹤**：所有操作記錄到 Space-SOC
6. **回滾機制**：保留舊版本以便回滾（未實作）

## 未來改進

- 自動 SBOM 產生整合
- 與漏洞資料庫（如 NVD）整合
- 多階段部署（canary, blue-green）
- 自動回滾機制
- 更新排程和維護窗口

