# バックアップ・復旧 (DR) & データ仕様書 (`backup_and_recovery.md`)

本書では、Host Credential Manager (HCM) におけるデータのバックアップ手順、ディザスタリカバリ（復旧・移行手順）、CSV 入出力の重要仕様および制限事項について解説します。

---

## 1. バックアップ対象ファイル一覧

HCM はステートレスなバイナリと、ローカルファイル（TOML および PEM）のみで完全動作する設計になっています。そのため、以下の 2 つのディレクトリを退避するだけで、完全なバックアップが可能です。

```text
host-credential-manager-go/
├── data/                             # 【必須】データベース・設定ディレクトリ
│   ├── config.toml                   # 許可IP、システムパスワード、マスターパスワード
│   ├── hostlist.toml                 # ホスト一覧、IP、ポート、OS、タグ、アクセスリスト
│   └── host_credentials.toml        # 各ホストのログインユーザー名、パスワード
└── cert/                             # 【必須】TLS / mTLS 証明書ディレクトリ
    ├── cacert.pem / cakey.pem        # Root CA 証明書および秘密鍵
    ├── cert.pem / key.pem            # サーバー証明書および秘密鍵
    ├── client_cert.pem / client_key.pem # クライアント証明書および秘密鍵
    └── crl.pem                       # 証明書失効リスト (CRL)
```

> [!CAUTION]
> `data/` だけでなく `cert/` も必ず一緒にバックアップしてください。
> `cert/` を失うと Root CA の秘密鍵が失われ、既存の `hcm-client` との mTLS 通信や CRL による失効管理が継続できなくなります。

---

## 2. 定期バックアップ・リストア手順

### 2.1 バックアップの作成（アーカイブ化）

サーバーホスト上で以下のコマンドを実行し、タイムスタンプ付きの圧縮アーカイブを作成します：

```bash
# プロジェクトルートで実行
BACKUP_NAME="hcm-backup-$(date +%Y%m%d_%H%M%S).tar.gz"
tar -czvf "${BACKUP_NAME}" data/ cert/

# バックアップファイルの安全なパーミッション設定
chmod 600 "${BACKUP_NAME}"
```

生成されたアーカイブには平文のパスワードや秘密鍵が含まれるため、セキュアなバックアップストレージ（暗号化された S3 バケットやセキュア NAS）へ暗号化転送してください。

### 2.2 リストア（復旧）および別サーバーへの移行

新サーバーへの移行や障害復旧時は、以下の手順で即座にサービスを再開できます：

1. 新サーバーに Go 実行バイナリ（またはリポジトリ一式）を配置します。
2. バックアップアーカイブを展開します：
   ```bash
   tar -xzvf hcm-backup-YYYYMMDD_HHMMSS.tar.gz -C ./
   ```
3. パーミッションを厳格化します：
   ```bash
   chmod 700 data cert
   chmod 600 data/*.toml cert/*.pem
   ```
4. サーバーを起動します：
   ```bash
   ./host-credential-manager
   ```
5. これにより、ホスト情報、パスワード情報、mTLS 証明書チェーン、CRL 失効情報が以前と全く同一の状態で復旧します。既存の `hcm-client` もそのまま接続を継続できます。

---

## 3. CSV インポート/エクスポートの仕様と重要制限

Web UI の管理者メニューには「CSVエクスポート」「CSVインポート」機能が用意されています。

### 3.1 【最重要警告】CSV にはパスワード情報が含まれません
> [!WARNING]
> **CSV をバックアップとして使用しないでください**:
> セキュリティおよびフォーマットの観点から、**CSV エクスポート機能はホストのメタデータ（ホスト名、IP、OS、タグ等）のみを出力します。ユーザー名およびパスワード（`userlist`）は CSV に出力されず、CSV インポートでも復元されません**。
> パスワードも含めた完全なバックアップを行うには、必ず前述の `data/host_credentials.toml` を退避してください。

### 3.2 CSV のフォーマット仕様
CSV のヘッダー行および各列の定義は以下の通りです：

```csv
id,hostname,ip,platform,os,port,tags,description,updatedAt
srv01,web-frontend-01,10.0.1.10,Linux,Ubuntu 22.04,22,"web,production","Web proxy node",2026-01-01T00:00:00.000Z
sw01,core-switch-01,192.168.1.1,Cisco,IOS-XE,23,"network,core","Core switch",2026-01-01T00:00:00.000Z
```

- 文字コード: `UTF-8`
- 改行コード: `LF` または `CRLF`
- カンマ区切り（カンマや改行を含む項目はダブルクォート `"` で囲む）
- インポート時のマージオプション:
  - `merge=false`（全置換）: 既存の `hostlist.toml` をインポートした CSV の内容で完全に上書きします。
  - `merge=true`（マージ）: 既存ホストと同一 ID のものを更新し、新しい ID のものを追加します。

---

## 4. `hostlist.toml` と `host_credentials.toml` の結合仕様

HCM は内部でホスト情報と認証情報を 2 つの独立したファイルで管理しています：
- `data/hostlist.toml`: ホストの基本情報（IP、プラットフォーム、OS、タグ、アクセスリスト）
- `data/host_credentials.toml`: ホストに紐付くログインユーザー一覧と各パスワード

### 4.1 結合キー（`hostname`）
- 両ファイルは **`hostname`（ホスト名文字列）を完全一致キー** としてメモリ上で結合されます。
- そのため、ホスト名はシステム全体で一意（ユニーク）でなければなりません。

### 4.2 ホスト名変更時の注意点
Web UI 上でホスト名を編集・変更した場合：
1. `hostlist.toml` の `hostname` は新しい名前に書き換わります。
2. しかし、現行バージョンでは `host_credentials.toml` 側のキーが自動更新されず、**パスワードとの紐付きが切断される現象** が発生する場合があります。
3. **対処法**: ホスト名を変更した後は、該当ホストのクレデンシャル設定画面を開いてユーザー名・パスワードが正しく紐付いているか確認し、必要に応じて再登録を行ってください。
