#!/bin/bash

# Phase 3 測試腳本 - OTA 與 SBOM

set -e

echo "=== Phase 3 OTA 與 SBOM 測試 ==="
echo ""

# 檢查服務健康狀態
echo "1. 檢查服務健康狀態..."
curl -s http://localhost:8084/health | grep -q "ok" && echo "   ✅ OTA Controller 正常" || echo "   ❌ OTA Controller 異常"
curl -s http://localhost:8082/health | grep -q "ok" && echo "   ✅ Satellite Sim 正常" || echo "   ❌ Satellite Sim 異常"
curl -s http://localhost:8083/health | grep -q "ok" && echo "   ✅ Space-SOC Backend 正常" || echo "   ❌ Space-SOC Backend 異常"
echo ""

# 測試 SBOM policy 檢查
echo "2. 測試 SBOM Policy 檢查..."
if [ -f "supply-chain/sbom/examples/satellite-sim-v1.0.0.cdx.json" ]; then
    go run supply-chain/sbom/cmd/check-sbom/main.go \
        -sbom supply-chain/sbom/examples/satellite-sim-v1.0.0.cdx.json && \
        echo "   ✅ SBOM Policy 檢查通過" || echo "   ⚠️  SBOM Policy 有違規"
else
    echo "   ⚠️  SBOM 範例檔案不存在"
fi
echo ""

# 註冊新版本
echo "3. 註冊新版本到 OTA Controller..."
RELEASE_RESPONSE=$(curl -s -X POST http://localhost:8084/api/v1/releases \
    -H "Content-Type: application/json" \
    -d '{
        "component": "satellite-sim",
        "version": "v1.1.0",
        "imageDigest": "sha256:test123456",
        "sbomUrl": "http://example.com/sbom.json",
        "attestation": "{\"digest\":\"sha256:test123456\",\"signature\":\"test_sig\"}"
    }')

RELEASE_ID=$(echo $RELEASE_RESPONSE | jq -r '.id')
if [ "$RELEASE_ID" != "null" ] && [ "$RELEASE_ID" != "" ]; then
    echo "   ✅ 版本註冊成功 (ID: $RELEASE_ID)"
else
    echo "   ❌ 版本註冊失敗"
    echo "   回應: $RELEASE_RESPONSE"
fi
echo ""

# 批准版本
if [ "$RELEASE_ID" != "null" ] && [ "$RELEASE_ID" != "" ]; then
    echo "4. 批准版本..."
    APPROVE_RESPONSE=$(curl -s -X POST http://localhost:8084/api/v1/releases/$RELEASE_ID/approve)
    echo $APPROVE_RESPONSE | jq -r '.status' | grep -q "approved" && \
        echo "   ✅ 版本批准成功" || echo "   ❌ 版本批准失敗"
    echo ""
fi

# 查詢 releases
echo "5. 查詢所有 releases..."
RELEASES=$(curl -s http://localhost:8084/api/v1/releases)
RELEASE_COUNT=$(echo $RELEASES | jq -r '.count')
echo "   ✅ 找到 $RELEASE_COUNT 個 releases"
echo ""

# 檢查軟體姿態
echo "6. 檢查軟體姿態..."
sleep 2
POSTURE=$(curl -s http://localhost:8083/api/v1/posture)
POSTURE_COUNT=$(echo $POSTURE | jq -r '.count')
echo "   ✅ 追蹤 $POSTURE_COUNT 個組件"
if [ "$POSTURE_COUNT" -gt 0 ]; then
    echo "   組件列表:"
    echo $POSTURE | jq -r '.postures[] | "      - \(.component): \(.currentVersion)"'
fi
echo ""

# 檢查 OTA 相關事件
echo "7. 檢查 OTA 相關事件..."
EVENTS=$(curl -s "http://localhost:8083/api/v1/events?limit=50")
OTA_EVENTS=$(echo $EVENTS | jq -r '.events[] | select(.component == "ota-controller") | .eventType' | wc -l)
echo "   ✅ 找到 $OTA_EVENTS 個 OTA 相關事件"
echo ""

echo "=== Phase 3 測試完成 ==="
echo ""
echo "訪問以下 URL 查看結果:"
echo "  - Space-SOC 事件: http://localhost:3001"
echo "  - 軟體姿態: http://localhost:3001/posture"
echo "  - OTA Releases: curl http://localhost:8084/api/v1/releases"

