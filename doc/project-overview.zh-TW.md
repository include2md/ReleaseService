# App Asset Service 專案說明（中文）

## 1. 主要目的

`App Asset Service` 是一個專門服務前端靜態資產發版與讀取的後端服務，核心目標如下：

- 讓 CI 可以上傳前端打包檔（tar.gz）並建立不可變版本。
- 將版本資產儲存到 MinIO，並以 `app/version` 作為儲存路徑。
- 提供資產讀取 API 給 CDN / 前端。
- 控制每個 `app` 僅保留最多 15 個可用版本（`success`）；超過則自動 rotation。
- rotation 時刪除 MinIO 舊版檔案，MongoDB 走軟刪除（`status = rotated_deleted`）。

---

## 2. 後端流程

### 2.1 Upload 流程（寫入）

1. Client/CI 呼叫 `POST /api/v1/releases`（需 Bearer Token）。
2. Handler 驗證必要欄位與 artifact（必須是 tar.gz）。
3. 服務層解壓後逐檔上傳到 MinIO：`/{app}/{version}/{file}`。
4. 寫入 MongoDB release 文件，狀態為 `success`。
5. 執行 rotation：
   - 查出同 `app` 且 `status=success` 的版本（新到舊）。
   - 保留最新 15 個，其他視為過期版本。
   - 對過期版本刪除 MinIO prefix。
   - 將 MongoDB 該版本 `status` 更新為 `rotated_deleted`。

### 2.2 Asset 讀取流程（讀取）

1. CDN/瀏覽器呼叫 `GET /assets/:app/:version/*path`。
2. 服務組出 MinIO key 並回傳檔案內容。
3. Cache-Control：
   - `index.html` -> `no-cache`
   - 其他靜態資產 -> `public, max-age=31536000, immutable`

### 2.3 版本列表流程

1. 前端呼叫 `GET /api/v1/apps/:app/versions`（不需 Token）。
2. 僅查詢 `status=success`。
3. 回傳可用版本清單（新到舊）。

---

## 3. API 說明

### 3.1 Upload Release

- Method: `POST`
- Path: `/api/v1/releases`
- Auth: `Authorization: Bearer <token>`
- Content-Type: `multipart/form-data`

Request fields:
- `app_name` (required)
- `version` (required)
- `artifact` (required, tar.gz)
- `commit_sha` (optional)
- `build_id` (optional)

Success response (`200`):

```json
{
  "success": true,
  "app_name": "my-app",
  "version": "1.2.3",
  "status": "success"
}
```

Common errors:
- `400` 缺欄位 / 非 tar.gz / 壞檔案
- `401` token 驗證失敗
- `409` 同 `app_name + version` 已存在
- `500` 伺服器處理失敗

### 3.2 Get Asset

- Method: `GET`
- Path: `/assets/:app/:version/*path`
- Auth: none

Example:
- `/assets/my-app/1.2.3/index.html`

Success response:
- `200` + binary/text body
- 含對應的 `Content-Type` 與 `Cache-Control`

Common errors:
- `404` 檔案不存在
- `500` 讀取失敗

### 3.3 List Available Versions

- Method: `GET`
- Path: `/api/v1/apps/:app/versions`
- Auth: none

Success response (`200`):

```json
{
  "success": true,
  "app_name": "my-app",
  "versions": ["2.0.0", "1.9.0", "1.8.5"]
}
```

Common errors:
- `400` 缺少 `app`
- `500` 查詢失敗

---

## 4. Schema

### 4.1 MongoDB

- DB: 預設 `app_asset_service`（可由 `MONGO_DB` 覆寫）
- Collection: 預設 `app_asset_releases`（可由 `MONGO_COLLECTION` 覆寫）

Document fields:
- `app_name` (string)
- `version` (string)
- `status` (string) - 目前值：`success`, `rotated_deleted`
- `storage_prefix` (string) - 例：`/my-app/1.2.3/`
- `commit_sha` (string, optional)
- `build_id` (string, optional)
- `created_at` (datetime, UTC)

Index:
- Unique: `(app_name, version)`

### 4.2 MinIO 路徑

- Prefix pattern: `/{app}/{version}/{file}`
- 例：`/my-app/1.2.3/index.html`

---

## 5. 未來可做的項目

- 版本列表增加分頁、limit、prefix 過濾。
- 增加 release 詳細查詢 API（含狀態、建立時間、commit SHA）。
- rotation 失敗補償機制（例如重試佇列、背景 job）。
- 新增審計欄位（`rotated_at`, `rotated_by`, `delete_reason`）。
- 建立管理 API：手動下架、回滾、重新標記 status。
- 支援 artifact checksum 驗證與簽章檢查。
- 強化安全性：token 管理、細粒度權限、審計 log。
- 增加可觀測性：Prometheus metrics、trace、結構化 logging。
- 增加更多整合測試（Mongo + MinIO + router e2e）。
