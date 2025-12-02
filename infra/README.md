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
```

### 服務端口

- **Space-SOC Frontend**: http://localhost:3000
- **Space-SOC Backend**: http://localhost:8080
- **TT&C Gateway**: http://localhost:8081
- **Satellite Simulator**: http://localhost:8082

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
   netstat -ano | findstr :8080
   # 記下 PID，然後終止該程序
   taskkill /PID <PID> /F
   ```

3. **或者修改端口映射**：
   編輯 `infra/docker-compose.yaml`，將端口改為其他可用端口（例如 8080 → 18080）

4. **重新啟動**：
   ```bash
   docker compose -f infra/docker-compose.yaml up -d
   ```

#### 建置失敗

如果建置映像檔失敗：
1. 確認所有 Dockerfile 路徑正確
2. 檢查網路連線（需要下載基礎映像檔）
3. 查看詳細錯誤訊息：`docker compose -f infra/docker-compose.yaml build --no-cache`

### 開發模式

在開發時，可以使用 `docker compose` 的 `--build` 選項來重新建置映像檔：

```bash
docker compose -f infra/docker-compose.yaml up --build
```
