# OTA Controller

OTA (Over-The-Air) Controller 管理衛星軟體更新的發布、批准和分發流程。

## 功能

- **版本管理**：註冊和追蹤軟體版本
- **批准工作流**：需要人工批准才能分發更新
- **任務政策**：根據任務階段（normal, critical, safe_mode）控制更新
- **安全驗證**：整合簽章驗證和 SBOM policy 檢查
- **事件記錄**：所有更新活動記錄到 Space-SOC

## API 端點

### 檢查更新

```bash
POST /api/v1/updates/check
Content-Type: application/json

{
  "component": "satellite-sim",
  "currentVersion": "v1.0.0",
  "satelliteId": "SAT-001"
}
```

### 註冊新版本

```bash
POST /api/v1/releases
Content-Type: application/json

{
  "component": "satellite-sim",
  "version": "v1.1.0",
  "imageDigest": "sha256:abc123...",
  "sbomUrl": "https://registry.example.com/sbom/satellite-sim-v1.1.0.json",
  "attestation": "{...}"
}
```

### 批准版本

```bash
POST /api/v1/releases/:id/approve
```

### 查詢所有版本

```bash
GET /api/v1/releases?component=satellite-sim&status=approved
```

## 環境變數

- `PORT`: 服務端口（預設: 8084）
- `DATABASE_PATH`: SQLite 資料庫路徑（預設: ota-controller.db）
- `MISSION_PHASE`: 任務階段（normal, critical, safe_mode）
- `SPACE_SOC_URL`: Space-SOC backend URL（用於事件記錄）

## 使用範例

### 1. 註冊新版本（由 CI pipeline 調用）

```bash
curl -X POST http://localhost:8084/api/v1/releases \
  -H "Content-Type: application/json" \
  -d '{
    "component": "satellite-sim",
    "version": "v1.1.0",
    "imageDigest": "sha256:abc123",
    "sbomUrl": "http://registry/sbom.json",
    "attestation": "{\"digest\":\"sha256:abc123\",\"signature\":\"...\"}"
  }'
```

### 2. 批准版本

```bash
curl -X POST http://localhost:8084/api/v1/releases/1/approve
```

### 3. 衛星檢查更新

```bash
curl -X POST http://localhost:8084/api/v1/updates/check \
  -H "Content-Type: application/json" \
  -d '{
    "component": "satellite-sim",
    "currentVersion": "v1.0.0"
  }'
```

## 安全考量

- 所有版本預設為 `pending` 狀態，需要人工批准
- 關鍵任務階段自動阻止更新
- 整合 SBOM policy 檢查（檢查已知漏洞、授權限制）
- 簽章驗證確保更新來源可信
- 所有操作記錄到 Space-SOC 供審計

## 與其他組件整合

- **CI Pipeline**: 建置完成後註冊新版本
- **Satellite-sim**: 週期性檢查更新並應用
- **Space-SOC**: 接收所有更新相關事件
- **Signing Service**: 提供簽章和 attestation

