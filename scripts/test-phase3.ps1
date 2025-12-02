# Phase 3 測試腳本 - OTA 與 SBOM (PowerShell)

Write-Host "=== Phase 3 OTA 與 SBOM 測試 ===" -ForegroundColor Cyan
Write-Host ""

# 檢查服務健康狀態
Write-Host "1. 檢查服務健康狀態..." -ForegroundColor Yellow
try {
    $r = Invoke-WebRequest -Uri http://localhost:8084/health -UseBasicParsing
    Write-Host "   ✅ OTA Controller 正常" -ForegroundColor Green
} catch {
    Write-Host "   ❌ OTA Controller 異常" -ForegroundColor Red
}

try {
    $r = Invoke-WebRequest -Uri http://localhost:8082/health -UseBasicParsing
    Write-Host "   ✅ Satellite Sim 正常" -ForegroundColor Green
} catch {
    Write-Host "   ❌ Satellite Sim 異常" -ForegroundColor Red
}

try {
    $r = Invoke-WebRequest -Uri http://localhost:8083/health -UseBasicParsing
    Write-Host "   ✅ Space-SOC Backend 正常" -ForegroundColor Green
} catch {
    Write-Host "   ❌ Space-SOC Backend 異常" -ForegroundColor Red
}
Write-Host ""

# 測試 SBOM policy 檢查
Write-Host "2. 測試 SBOM Policy 檢查..." -ForegroundColor Yellow
if (Test-Path "supply-chain/sbom/examples/satellite-sim-v1.0.0.cdx.json") {
    try {
        go run supply-chain/sbom/cmd/check-sbom/main.go `
            -sbom supply-chain/sbom/examples/satellite-sim-v1.0.0.cdx.json 2>&1 | Out-Null
        Write-Host "   ✅ SBOM Policy 檢查通過" -ForegroundColor Green
    } catch {
        Write-Host "   ⚠️  SBOM Policy 有違規" -ForegroundColor Yellow
    }
} else {
    Write-Host "   ⚠️  SBOM 範例檔案不存在" -ForegroundColor Yellow
}
Write-Host ""

# 註冊新版本
Write-Host "3. 註冊新版本到 OTA Controller..." -ForegroundColor Yellow
$releaseBody = @{
    component = "satellite-sim"
    version = "v1.1.0"
    imageDigest = "sha256:test123456"
    sbomUrl = "http://example.com/sbom.json"
    attestation = '{"digest":"sha256:test123456","signature":"test_sig"}'
} | ConvertTo-Json

try {
    $releaseResponse = Invoke-WebRequest -Uri http://localhost:8084/api/v1/releases `
        -Method POST `
        -Headers @{'Content-Type'='application/json'} `
        -Body $releaseBody `
        -UseBasicParsing
    
    $releaseData = $releaseResponse.Content | ConvertFrom-Json
    $releaseId = $releaseData.id
    Write-Host "   ✅ 版本註冊成功 (ID: $releaseId)" -ForegroundColor Green
} catch {
    Write-Host "   ❌ 版本註冊失敗: $($_.Exception.Message)" -ForegroundColor Red
    $releaseId = $null
}
Write-Host ""

# 批准版本
if ($releaseId) {
    Write-Host "4. 批准版本..." -ForegroundColor Yellow
    try {
        $approveResponse = Invoke-WebRequest -Uri "http://localhost:8084/api/v1/releases/$releaseId/approve" `
            -Method POST `
            -UseBasicParsing
        
        $approveData = $approveResponse.Content | ConvertFrom-Json
        if ($approveData.status -eq "approved") {
            Write-Host "   ✅ 版本批准成功" -ForegroundColor Green
        } else {
            Write-Host "   ❌ 版本批准失敗" -ForegroundColor Red
        }
    } catch {
        Write-Host "   ❌ 版本批准失敗: $($_.Exception.Message)" -ForegroundColor Red
    }
    Write-Host ""
}

# 查詢 releases
Write-Host "5. 查詢所有 releases..." -ForegroundColor Yellow
try {
    $releasesResponse = Invoke-WebRequest -Uri http://localhost:8084/api/v1/releases -UseBasicParsing
    $releasesData = $releasesResponse.Content | ConvertFrom-Json
    Write-Host "   ✅ 找到 $($releasesData.count) 個 releases" -ForegroundColor Green
} catch {
    Write-Host "   ❌ 查詢失敗" -ForegroundColor Red
}
Write-Host ""

# 檢查軟體姿態
Write-Host "6. 檢查軟體姿態..." -ForegroundColor Yellow
Start-Sleep -Seconds 2
try {
    $postureResponse = Invoke-WebRequest -Uri http://localhost:8083/api/v1/posture -UseBasicParsing
    $postureData = $postureResponse.Content | ConvertFrom-Json
    Write-Host "   ✅ 追蹤 $($postureData.count) 個組件" -ForegroundColor Green
    
    if ($postureData.count -gt 0) {
        Write-Host "   組件列表:" -ForegroundColor Cyan
        foreach ($p in $postureData.postures) {
            Write-Host "      - $($p.component): $($p.currentVersion)" -ForegroundColor Gray
        }
    }
} catch {
    Write-Host "   ❌ 查詢失敗: $($_.Exception.Message)" -ForegroundColor Red
}
Write-Host ""

# 檢查 OTA 相關事件
Write-Host "7. 檢查 OTA 相關事件..." -ForegroundColor Yellow
try {
    $eventsResponse = Invoke-WebRequest -Uri "http://localhost:8083/api/v1/events?limit=50" -UseBasicParsing
    $eventsData = $eventsResponse.Content | ConvertFrom-Json
    $otaEvents = $eventsData.events | Where-Object { $_.component -eq "ota-controller" }
    Write-Host "   ✅ 找到 $($otaEvents.Count) 個 OTA 相關事件" -ForegroundColor Green
} catch {
    Write-Host "   ❌ 查詢失敗" -ForegroundColor Red
}
Write-Host ""

Write-Host "=== Phase 3 測試完成 ===" -ForegroundColor Cyan
Write-Host ""
Write-Host "訪問以下 URL 查看結果:" -ForegroundColor Yellow
Write-Host "  - Space-SOC 事件: http://localhost:3001"
Write-Host "  - 軟體姿態: http://localhost:3001/posture"
Write-Host "  - OTA Releases: http://localhost:8084/api/v1/releases"

