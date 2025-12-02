# Infrastructure

本目錄包含 Space Cyber Resilience Platform 的基礎設施配置。

## Docker Compose

使用 Docker Compose 來啟動整個平台的所有服務。

### 前置需求

1. **Docker Desktop** 必須正在運行
   - Windows: 啟動 Docker Desktop 應用程式
   - 確認 Docker 服務正在運行：`docker ps` 應該能正常執行

2. **Docker Compose** (通常已包含在 Docker Desktop 中)

### 啟動服務

```bash
# 從專案根目錄執行
docker compose -f infra/docker-compose.yaml up

# 或使用後台模式
docker compose -f infra/docker-compose.yaml up -d

# 查看日誌
docker compose -f infra/docker-compose.yaml logs -f

# 停止服務
docker compose -f infra/docker-compose.yaml down

# 停止並移除 volumes
docker compose -f infra/docker-compose.yaml down -v
```

### 服務端口

- **Space-SOC Frontend**: http://localhost:3001
- **Space-SOC Backend**: http://localhost:8083
- **TT&C Gateway**: http://localhost:8081
- **Satellite Simulator**: http://localhost:8082
- **OTA Controller**: http://localhost:8084

### 服務說明

#### Space-SOC (Security Operations Center)

- **Backend**: 事件接收、儲存、查詢，Incidents 管理，軟體姿態追蹤
- **Frontend**: 事件儀表板、Incidents 視圖、軟體姿態面板

#### TT&C Gateway

- Zero-Trust 指令閘道
- Policy-as-code 引擎
- 規則型異常偵測
- 完整審計日誌

#### Satellite Simulator

- 模擬衛星節點
- 接收並執行指令
- OTA 客戶端（自動更新）
- 事件記錄

#### OTA Controller

- 軟體版本管理
- 批准工作流
- 任務政策執行
- SBOM policy 檢查整合

### 故障排除

#### Docker Desktop 未運行

如果看到錯誤訊息：
```
error during connect: Get "http://%2F%2F.%2Fpipe%2FdockerDesktopLinuxEngine/...": 
open //./pipe/dockerDesktopLinuxEngine: The system cannot find the file specified.
```

**解決方法**：
1. 啟動 Docker Desktop 應用程式
2. 等待 Docker Desktop 完全啟動（系統托盤圖示不再顯示「正在啟動」）
3. 再次執行 `docker compose` 命令

#### 端口已被佔用

如果看到錯誤訊息：
```
failed to bind host port for 0.0.0.0:8080: address already in use
```

**解決方法**：

1. **停止所有相關容器**：
   ```bash
   docker compose -f infra/docker-compose.yaml down
   ```

2. **檢查並停止佔用端口的程序**（Windows）：
   ```bash
   netstat -ano | findstr :8083
   # 記下 PID，然後終止該程序
   taskkill /PID <PID> /F
   ```

3. **或者修改端口映射**：
   編輯 `infra/docker-compose.yaml`，將端口改為其他可用端口

4. **重新啟動**：
   ```bash
   docker compose -f infra/docker-compose.yaml up -d
   ```

#### 建置失敗

如果建置映像檔失敗：
1. 確認所有 Dockerfile 路徑正確
2. 檢查網路連線（需要下載基礎映像檔）
3. 清除快取重新建置：
   ```bash
   docker compose -f infra/docker-compose.yaml build --no-cache
   ```
4. 查看詳細錯誤訊息

#### 服務無法啟動

1. 檢查服務狀態：
   ```bash
   docker compose -f infra/docker-compose.yaml ps
   ```

2. 查看特定服務的日誌：
   ```bash
   docker compose -f infra/docker-compose.yaml logs <service-name>
   ```

3. 檢查健康檢查狀態：
   ```bash
   docker inspect <container-name> | grep -A 10 Health
   ```

### 開發模式

在開發時，可以使用 `docker compose` 的 `--build` 選項來重新建置映像檔：

```bash
docker compose -f infra/docker-compose.yaml up --build
```

只重建特定服務：

```bash
docker compose -f infra/docker-compose.yaml up -d --build space-soc-backend
```

### 清理

清理所有容器、網路和 volumes：

```bash
# 停止並移除所有資源
docker compose -f infra/docker-compose.yaml down -v

# 清理未使用的映像檔
docker image prune -a

# 清理所有未使用的資源
docker system prune -a --volumes
```

## 測試指南

### 基本健康檢查

```bash
# 檢查所有服務
curl http://localhost:8083/health  # Space-SOC Backend
curl http://localhost:8081/health  # TT&C Gateway
curl http://localhost:8082/health  # Satellite Sim
curl http://localhost:8084/health  # OTA Controller
```

### 測試 TT&C Gateway

```bash
# 使用 operator 角色發送危險指令（應被拒絕）
curl -X POST http://localhost:8081/command \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer operator-token" \
  -d '{"command":"deorbit"}'

# 使用 admin 角色（應被允許）
curl -X POST http://localhost:8081/command \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer admin-token" \
  -d '{"command":"deorbit"}'
```

### 測試 OTA 流程

請參考 `docs/PHASE3_OTA_GUIDE.md`

### 查看 Space-SOC

1. 開啟瀏覽器訪問 http://localhost:3001
2. 查看事件列表
3. 切換到 Incidents 標籤
4. 訪問軟體姿態頁面：http://localhost:3001/posture
