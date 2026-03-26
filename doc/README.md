# Documentation

## Demo Page 操作說明

Demo 頁面檔案：`doc/demo.html`

### 1. 啟動服務

在專案根目錄執行：

```bash
make up
```

預設會啟動：
- App Asset Service: `http://localhost:8080`
- MongoDB: `localhost:27017`
- MinIO API: `http://localhost:9000`
- MinIO Console: `http://localhost:9001`

### 2. 開啟 Demo 頁面

方式 A：直接打開 `doc/demo.html`

方式 B（建議）：在 `doc/` 目錄啟動靜態 server

```bash
cd doc
python -m http.server 5500
```

然後開啟：`http://localhost:5500/demo.html`

### 3. Upload Release（上傳版本）

在頁面左側填寫：
- Base URL: `http://localhost:8080`
- Bearer Token: `local-dev-token`
- app_name: 例如 `my-app`
- version: 例如 `1.2.3`
- artifact: 選擇 `.tar.gz` 檔案

按下 `Upload Release`。

成功時會看到：
- 狀態為 `Upload success`
- 回應 HTTP 狀態 `200`

若版本已存在（同 `app_name + version`）會回：
- HTTP `409 Conflict`

### 4. Fetch Asset（讀取資產）

在頁面右側填寫：
- app: 與上傳一致（例如 `my-app`）
- version: 與上傳一致（例如 `1.2.3`）
- path: 例如 `index.html` 或 `assets/app.js`

按下 `Fetch Asset`。

成功時會看到：
- HTTP `200`
- `index.html` 的 `Cache-Control: no-cache`
- 靜態資產（如 `app.js`）的 `Cache-Control: public, max-age=31536000, immutable`

### 5. CORS 說明

- CORS 只影響「瀏覽器跨網域請求」。
- Demo 頁面若不是和 API 同一個 origin，可能遇到 CORS。
- `curl`、CI、服務對服務呼叫通常不受 CORS 限制。
- 若正式環境前端與 API 是不同網域，需要在 API 端設定 CORS 白名單。



tar -czf ../ttt.tar.gz index.html
