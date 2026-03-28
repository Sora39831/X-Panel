# Partial Database Backup & Restore

## Context

The current backup system only supports full database export/import (raw SQLite `.db` file). Users need the ability to selectively export and import specific tables as JSON, so they can:
- Migrate only certain data between panels
- Backup only important tables without history/logs
- Restore specific tables without overwriting the entire database

## Scope

- SQLite path only (MongoDB already has provider-based partial operations)
- Frontend-only changes to register tab input type (no backend changes)

## Tables Available for Selection

| Table Name | Description | i18n Key |
|-----------|-------------|----------|
| `users` | Admin user accounts | `pages.index.tableUsers` |
| `inbounds` | Inbound configurations | `pages.index.tableInbounds` |
| `client_traffics` | Per-client traffic stats | `pages.index.tableClientTraffics` |
| `settings` | Panel settings | `pages.index.tableSettings` |
| `inbound_client_ips` | Client IP restrictions | `pages.index.tableInboundClientIps` |
| `outbound_traffics` | Outbound traffic logs | `pages.index.tableOutboundTraffics` |
| `history_of_seeders` | Migration history | `pages.index.tableHistoryOfSeeders` |
| `link_histories` | Subscription link history | `pages.index.tableLinkHistories` |
| `lottery_wins` | Lottery win records | `pages.index.tableLotteryWins` |

## JSON Export Format

```json
{
  "version": "1.0",
  "exported_at": "2026-03-27T12:00:00Z",
  "tables": {
    "users": [
      {"id": 1, "username": "admin", "password": "...", "role": "admin"}
    ],
    "inbounds": [
      {"id": 1, "up": 0, "down": 0, "total": 0, "remark": "...", "enable": true, ...}
    ]
  }
}
```

## Backend Changes

### 1. New endpoint: `GET /panel/api/server/getPartialDb`

**File:** `web/controller/server.go`

- Query param: `tables` — comma-separated table names (e.g., `?tables=inbounds,users`)
- Validate table names against an allowlist (prevent SQL injection)
- Call `ServerService.GetPartialDb(tables []string)`
- Return as JSON file download (`partial-backup.json`)

**File:** `web/service/server.go`

- New method `GetPartialDb(tables []string) ([]byte, error)`
- For each table: `database.GetDB().Table(tableName).Find(&results)` into `[]map[string]interface{}`
- Assemble into the JSON structure above
- Return `json.Marshal` result

### 2. New endpoint: `POST /panel/api/server/importPartialDb`

**File:** `web/controller/server.go`

- Accept multipart file upload (field name: `db`)
- Parse the uploaded JSON file
- Call `ServerService.ImportPartialDb(file multipart.File)`

**File:** `web/service/server.go`

- New method `ImportPartialDb(file multipart.File) error`
- Read and parse JSON, extract table names from `tables` key
- For each table in JSON:
  - `db.Exec("DELETE FROM " + tableName)` — clear existing data
  - Insert rows from JSON using `db.Table(tableName).Create(&rows)` (batch insert)
- After all tables processed: call `xrayService.SetToNeedRestart()`
- **No Xray stop/start needed** — this is a data-level operation, Xray config is generated from DB on next restart

### 3. Table allowlist

Define a constant slice in `web/service/server.go`:

```go
var allowedPartialTables = []string{
    "users", "inbounds", "client_traffics", "settings",
    "inbound_client_ips", "outbound_traffics",
    "history_of_seeders", "link_histories", "lottery_wins",
}
```

Validate all requested table names against this list before executing queries.

## Frontend Changes

### 1. Backup modal additions (`web/html/index.html`)

Add two new list items in the existing `#backup-modal`:

```html
<a-list-item class="ant-backup-list-item">
    <a-list-item-meta>
        <template #title>{{ i18n "pages.index.partialExport" }}</template>
        <template #description>{{ i18n "pages.index.partialExportDesc" }}</template>
    </a-list-item-meta>
    <a-button @click="openPartialExport()" type="primary" icon="download"/>
</a-list-item>
<a-list-item class="ant-backup-list-item">
    <a-list-item-meta>
        <template #title>{{ i18n "pages.index.partialImport" }}</template>
        <template #description>{{ i18n "pages.index.partialImportDesc" }}</template>
    </a-list-item-meta>
    <a-button @click="importPartialDatabase()" type="primary" icon="upload" />
</a-list-item>
```

### 2. Table selection modal (`web/html/index.html`)

New modal for partial export with checkboxes:

```html
<a-modal id="partial-export-modal"
    v-model="partialExportModal.visible"
    title='{{ i18n "pages.index.partialExportTitle" }}'
    @ok="exportPartialDatabase()"
    :ok-text='{{ i18n "pages.index.exportDatabase" }}'>
    <a-checkbox-group v-model="partialExportModal.selectedTables">
        <div style="margin-bottom: 8px;">
            <a-checkbox value="users" title='{{ i18n "pages.index.tableUsersTip" }}'>{{ i18n "pages.index.tableUsers" }}</a-checkbox>
        </div>
        <div style="margin-bottom: 8px;">
            <a-checkbox value="inbounds" title='{{ i18n "pages.index.tableInboundsTip" }}'>{{ i18n "pages.index.tableInbounds" }}</a-checkbox>
        </div>
        <div style="margin-bottom: 8px;">
            <a-checkbox value="client_traffics" title='{{ i18n "pages.index.tableClientTrafficsTip" }}'>{{ i18n "pages.index.tableClientTraffics" }}</a-checkbox>
        </div>
        <div style="margin-bottom: 8px;">
            <a-checkbox value="settings" title='{{ i18n "pages.index.tableSettingsTip" }}'>{{ i18n "pages.index.tableSettings" }}</a-checkbox>
        </div>
        <div style="margin-bottom: 8px;">
            <a-checkbox value="inbound_client_ips" title='{{ i18n "pages.index.tableInboundClientIpsTip" }}'>{{ i18n "pages.index.tableInboundClientIps" }}</a-checkbox>
        </div>
        <div style="margin-bottom: 8px;">
            <a-checkbox value="outbound_traffics" title='{{ i18n "pages.index.tableOutboundTrafficsTip" }}'>{{ i18n "pages.index.tableOutboundTraffics" }}</a-checkbox>
        </div>
        <div style="margin-bottom: 8px;">
            <a-checkbox value="history_of_seeders" title='{{ i18n "pages.index.tableHistoryOfSeedersTip" }}'>{{ i18n "pages.index.tableHistoryOfSeeders" }}</a-checkbox>
        </div>
        <div style="margin-bottom: 8px;">
            <a-checkbox value="link_histories" title='{{ i18n "pages.index.tableLinkHistoriesTip" }}'>{{ i18n "pages.index.tableLinkHistories" }}</a-checkbox>
        </div>
        <div style="margin-bottom: 8px;">
            <a-checkbox value="lottery_wins" title='{{ i18n "pages.index.tableLotteryWinsTip" }}'>{{ i18n "pages.index.tableLotteryWins" }}</a-checkbox>
        </div>
    </a-checkbox-group>
</a-modal>
```

### 3. Vue methods

**`openPartialExport()`** — Opens the table selection modal, resets selection.

**`exportPartialDatabase()`** — Builds query string from selected tables, triggers download:
```javascript
const tables = this.partialExportModal.selectedTables.join(',');
window.location.href = `/panel/api/server/getPartialDb?tables=${tables}`;
```

**`importPartialDatabase()`** — Opens file picker (accepts `.json`), uploads to `/panel/api/server/importPartialDb`:
```javascript
const fileInput = document.createElement('input');
fileInput.type = 'file';
fileInput.accept = '.json';
fileInput.addEventListener('change', async (event) => {
    const file = event.target.files[0];
    if (file) {
        const formData = new FormData();
        formData.append('db', file);
        const msg = await HttpUtil.post('/panel/api/server/importPartialDb', formData);
        // handle result
    }
});
fileInput.click();
```

### 4. Vue data

Add to data:
```javascript
partialExportModal: {
    visible: false,
    selectedTables: [],
}
```

## Translation Keys

### English (`translate.en_US.toml`)

```toml
"partialExport" = "Partial Back Up"
"partialExportDesc" = "Select specific tables to export as JSON."
"partialExportTitle" = "Select Tables to Export"
"partialImport" = "Partial Restore"
"partialImportDesc" = "Import selected tables from a JSON backup file."
"partialImportSuccess" = "Partial database import completed successfully."
"partialImportError" = "An error occurred during partial database import."
"tableUsers" = "Users"
"tableInbounds" = "Inbounds"
"tableClientTraffics" = "Client Traffics"
"tableSettings" = "Settings"
"tableInboundClientIps" = "Inbound Client IPs"
"tableOutboundTraffics" = "Outbound Traffics"
"tableHistoryOfSeeders" = "History of Seeders"
"tableLinkHistories" = "Link Histories"
"tableLotteryWins" = "Lottery Wins"
"tableUsersTip" = "Admin user accounts"
"tableInboundsTip" = "Inbound proxy configurations"
"tableClientTrafficsTip" = "Per-client traffic statistics"
"tableSettingsTip" = "Panel settings and preferences"
"tableInboundClientIpsTip" = "Client IP restriction rules"
"tableOutboundTrafficsTip" = "Outbound traffic records"
"tableHistoryOfSeedersTip" = "Database migration history"
"tableLinkHistoriesTip" = "Subscription link history"
"tableLotteryWinsTip" = "Lottery game win records"
```

### Chinese (`translate.zh_CN.toml`)

```toml
"partialExport" = "部分备份"
"partialExportDesc" = "选择需要备份的表导出为 JSON 文件。"
"partialExportTitle" = "选择要导出的表"
"partialImport" = "部分恢复"
"partialImportDesc" = "从 JSON 备份文件导入选中的表。"
"partialImportSuccess" = "部分数据库导入成功。"
"partialImportError" = "部分数据库导入时发生错误。"
"tableUsers" = "用户"
"tableInbounds" = "入站"
"tableClientTraffics" = "客户端流量"
"tableSettings" = "设置"
"tableInboundClientIps" = "入站客户端IP"
"tableOutboundTraffics" = "出站流量"
"tableHistoryOfSeeders" = "迁移历史"
"tableLinkHistories" = "链接历史"
"tableLotteryWins" = "抽奖记录"
"tableUsersTip" = "管理员账号信息"
"tableInboundsTip" = "入站代理配置"
"tableClientTrafficsTip" = "各客户端的流量统计"
"tableSettingsTip" = "面板设置与偏好"
"tableInboundClientIpsTip" = "客户端IP限制规则"
"tableOutboundTrafficsTip" = "出站流量记录"
"tableHistoryOfSeedersTip" = "数据库迁移历史"
"tableLinkHistoriesTip" = "订阅链接历史"
"tableLotteryWinsTip" = "抽奖游戏获奖记录"
```

## Files to Modify

| File | Changes |
|------|---------|
| `web/controller/server.go` | Add 2 routes + 2 handler functions |
| `web/service/server.go` | Add `GetPartialDb()` + `ImportPartialDb()` methods |
| `web/html/index.html` | Add 2 list items + table selection modal + Vue data/methods |
| `web/translation/translate.en_US.toml` | Add ~11 translation keys |
| `web/translation/translate.zh_CN.toml` | Add ~11 translation keys |

## Security Considerations

- Table names validated against hardcoded allowlist — no SQL injection possible
- Import only accepts `.json` files (frontend filter)
- JSON structure validated before processing
- No password/credential exposure beyond what full backup already does
