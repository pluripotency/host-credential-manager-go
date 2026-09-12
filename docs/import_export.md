# インポート / エクスポート仕様書 (`docs/import_export.md`)

本書では、Host Credential Manager (HCM) におけるホスト情報および認証情報の **インポート / エクスポート機能** のアーキテクチャ、API 仕様、セキュリティ設計、およびフロントエンド UI の挙動について詳しく解説します。

---

## 1. 概要 (Overview)

HCM では、データの移行、バックアップ、共有を容易にするために以下の **2種類のインポート / エクスポート方式** を提供しています。

| 方式 | フォーマット | 認証情報 (パスワード) | 主な用途 |
| :--- | :--- | :---: | :--- |
| **タブパッケージ** | `.tgz` / `.tar.gz` | **含む (完全)** | タブ単位の完全な環境移行、バックアップ、他環境との共有 |
| **CSV 入出力** | `.csv` | **含まない (除外)** | スプレッドシート連携、インフラ台帳からの一括登録 |

> [!IMPORTANT]
> 認証情報（ユーザー名・パスワード）を含む完全なバックアップ・移行を行いたい場合は、必ず **タブパッケージ (`.tgz`)** を使用してください。CSV にはセキュリティ上の仕様によりパスワード情報は含まれません。

---

## 2. タブパッケージ (`.tgz`) エクスポート機能

任意のタブディレクトリに含まれる設定および認証情報を `.tgz` (tar.gz) 形式でアーカイブ化してダウンロードします。

### 2.1 API 仕様

- **エンドポイント**: `GET /api/tabs/:name/export` (または `GET /api/tabs/export?name=:name`)
- **認証 / 権限**: `Admin` ロール限定 (`session_token` Cookie 必須)
- **Content-Type**: `application/gzip`
- **Content-Disposition**: `attachment; filename="<タブ名>.tgz"`

### 2.2 処理フローとメタデータ生成

1. `tab_config.toml` から指定されたタブ名（`:name`）の定義を検索します。
2. タブの対象ディレクトリ（`config/[tab.dirpath]`）を特定します。
3. **メタデータファイル (`meta.toml` / `meta.md`) の自動生成**:
   - リストファイル名が標準（`hostlist.toml`）以外、または資格情報ファイル名が標準（`hostcredentials.toml`）以外の場合、インポート側で正しくファイル名を特定できるように、ディレクトリ直下に `meta.toml` および `meta.md` を自動生成してアーカイブに同梱します。
   ```toml
   # meta.toml の内容例
   list_filename = "hostlist.toml"
   cred_filename = "host_credentials.toml"
   ```
4. 対象ディレクトリ配下の全ファイルを tar.gz 形式で圧縮ストリーム化し、ダウンロードレスポンスとして返却します。
   - アーカイブのルートディレクトリ名は **タブ名** と同一になります（例: `example1/hostlist.toml`, `example1/host_credentials.toml`）。

---

## 3. タブパッケージ (`.tgz`) インポート機能

外部からアップロードされたタブパッケージアーカイブを展開し、バリデーションと競合確認を行った上で安全にデータベースに登録します。

### 3.1 API 仕様

- **エンドポイント**: `POST /api/tabs/import`
- **認証 / 権限**: `Admin` ロール限定
- **リクエスト形式**: `multipart/form-data`
  - `file`: アップロードするアーカイブファイル（`.tgz` または `.tar.gz`）
  - `overwrite`: (任意クエリまたはフォーム値) `true` の場合、同名タブが存在しても確認なしで上書き実行
- **レスポンス**:
  - 成功時 (`200 OK`): `{"success": true, "tab": { "name": "...", "dirpath": "...", ... }}`
  - 同名競合時 (`409 Conflict`):
    ```json
    {
      "conflict": true,
      "name": "example1",
      "dirpath": "./example1",
      "message": "Tab directory './example1' already exists. Overwrite?"
    }
    ```
  - バリデーションエラー (`400 Bad Request`): `{"error": "詳細エラーメッセージ"}`

### 3.2 安全性を確保するインポートパイプライン

アップロードされたファイルが本番環境のデータを破損させないよう、以下の堅牢なステージング・検証フローを実行します：

```
[アップロード受付 (.tgz)]
       │
       ▼
[1. 一時ステージング (/tmp/hcm/)] ──(パストラバーサル防止 & 展開)
       │
       ▼
[2. TOML ロード可能性検証] ──(hostlist / credentials が存在しパース可能か？)
       │  └─ 不正/破損 ➔ 400 Bad Request で安全に中断
       ▼
[3. tab_config.toml との重複検査]
       ├─ 既存タブと名前重複 & overwrite=false ➔ 409 Conflict 返却 (フロント側で確認ダイアログ表示)
       └─ 重複なし OR overwrite=true
              │
              ▼
[4. 本番 config/ 配下へのコピー] (上書き時は既存ディレクトリ削除)
              │
              ▼
[5. tab_config.toml の更新] ➔ 200 OK 完了
```

#### ① 一時ディレクトリ (`/tmp/hcm`) へのステージング展開
- アーカイブはいきなり本番の `config/` ディレクトリには展開されず、必ず一時ディレクトリ `/tmp/hcm/<展開ディレクトリ名>` に展開されます。
- `../` や絶対パスを含む悪意あるアーカイブ（Zip Slip 攻撃）をブロックするパスサニタイズを実施しています。

#### ② ファイルの存在と TOML 構文検証 (Loadability Validation)
- 展開ディレクトリ直下に、ホスト一覧（`hostlist.toml`）と資格情報（`host_credentials.toml` / `hostcredentials.toml` / `host_credentails.toml`）が存在するか確認します。
- `meta.toml` または `meta.md` が同梱されている場合は、そこに明記されたファイル名定義を優先します。
- **両ファイルが有効な TOML 構文としてパース可能（Loadable）であるかを実際にデコードして検証**します。構文エラーやファイル欠損がある場合は `400 Bad Request` となり、本番環境の設定は一切変更されません。

#### ③ 同名タブの競合検知と確認ダイアログ (`409 Conflict`)
- 展開されたディレクトリ名が `tab_config.toml` 内の `tab.name` に既に存在する場合、直ちに上書きすることはせず、`409 Conflict` を返却します。
- フロントエンド側はこのステータスを受け取ると「上書き確認モーダル」を表示します。ユーザーが明示的に「Yes (上書きする)」を選択した場合のみ、`?overwrite=true` を伴って再リクエストが送信されます。

#### ④ 本番配置と設定反映
- 検証通過後、`/tmp/hcm/<展開ディレクトリ名>` から `config/<展開ディレクトリ名>` へ全ファイルがコピーされます（上書き時は旧ディレクトリを削除）。
- `config/tab_config.toml` に新しいタブ項目（`name`, `dirpath`, `list_filename`, `cred_filename`）が自動追記（または更新）され、即座にシステム全体で利用可能になります。

---

## 4. フロントエンド UI の実装仕様

Web インターフェース（[`front/src/manager.tsx`](file:///home/worker/Documents/antigravity/host-credential-manager-go/front/src/manager.tsx)）におけるインポート / エクスポート操作の UX 設計です。

### 4.1 ボタンの表示条件

```tsx
{role === "admin" && selectedTabs.length === 1 && (
  // Export ボタン & Import ボタンの表示
)}
```

- **管理者権限 (Admin)**: 一般ユーザー (`user`) にはボタンが表示されません。
- **タブ選択条件**: **「タブが1つだけ選択されている時」** にのみヘッダー右上にボタンが表示されます。
  - 複数タブ選択時や、タブ未選択時は表示されません。これにより「どのタブをエクスポートするのか」「どのコンテキストで操作しているか」の曖昧さを排除しています。
  - Export ボタンには現在の選択タブ名が表示されます（例: `Export (example1)`）。

### 4.2 インポートモーダルダイアログと状態遷移

1. **ドラッグ＆ドロップ対応**:
   - `.tgz` または `.tar.gz` ファイルを点線エリアにドラッグ＆ドロップ、もしくはクリックしてファイル選択が可能です。
2. **上書き確認フロー**:
   - バックエンドから `409 Conflict` が返された場合、モーダル内の表示が自動的に「タブ上書きの確認」に切り替わります。
   - 警告メッセージとして重複したタブ名とターゲットディレクトリパスが明示されます。
   - 「No (上書きしない)」で安全にキャンセル、「Yes (上書きする)」で上書きインポートを実行します。
3. **自動リフレッシュ**:
   - インポート成功後、タブ設定とホスト一覧が自動的に `fetchHostList()` で再取得され、画面が最新状態に更新されます。

---

## 5. CSV インポート / エクスポート仕様

表計算ソフト（Excel や Google スプレッドシート）や他システムとの連携用のフラット形式データ入出力です。

### 5.1 API 仕様

- **エクスポート**: `GET /api/hostlist/export`
  - 登録されているホスト一覧を CSV 形式で一括ダウンロードします。
- **インポート**: `POST /api/hostlist/import`
  - `multipart/form-data` 形式で CSV ファイルをアップロードします。
  - オプション: `?merge=true`（既存とマージ）、`?merge=false`（全置換）。

### 5.2 CSV フォーマット

```csv
hostname,ip,platform,os,tags,description,updatedAt,tab,accesslist
web-prod-01,10.0.1.10,Linux,Ubuntu 24.04,"web,prod","Production web server",2026-09-12T00:00:00.000Z,example1,"[{""protocol"":""ssh"",""port"":""22""}]"
db-master-01,10.0.2.10,Linux,Debian 12,"db,cluster","Primary DB node",2026-09-12T00:00:00.000Z,example1,"[{""protocol"":""mysql"",""port"":""3306""}]"
```

- `id` 列は出力されません（TOML ファイル内で連番自動採番されるため）。
- `accesslist` は JSON 配列形式でシリアライズされます。

---

## 6. セキュリティと運用上の注意点

1. **アーカイブ内のパスワード平文性**:
   - タブパッケージ（`.tgz`）内の `host_credentials.toml` にはパスワード情報が平文で保存されています。
   - エクスポートした `.tgz` ファイルの保管場所、転送経路（暗号化通信の利用）、およびアクセス権限には十分注意してください。
2. **パストラバーサル検証**:
   - `ImportTabArchive` では、アーカイブヘッダーの各ファイルパスに対して `filepath.Clean` および先頭文字の境界チェックを実施し、ディレクトリ脱出（Directory Traversal）を防いでいます。
3. **ホットリロード (`air`) との競合防止**:
   - 開発サーバーにおいて、インポートやエクスポートによって `config/` ディレクトリ内のファイルが書き換わった際にサーバーが意図せず再起動しないよう、[`.air.toml`](file:///home/worker/Documents/antigravity/host-credential-manager-go/.air.toml) で `config` ディレクトリが監視除外（`exclude_dir`）されています。
