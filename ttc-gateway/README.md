# TT&C Gateway

此服務將實作零信任的遙測、測控閘道，負責：

- 驗證操作員身分與權限
- 依據 policy 決策是否允許指令
- 產生完整稽核與異常事件，送往 Space-SOC

EO / ACRI-ST / S2GM / CLMS 資料由 Space-SOC 集中提供；設定 `SPACE_SOC_URL` 後可透過 Space-SOC API 使用（resources、jobs、assets 等）。


