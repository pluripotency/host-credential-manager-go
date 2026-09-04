# Host Credential Manager (ホスト接続情報管理システム)

hostname, IPアドレス, プラットフォーム, OS, アクセスプロトコル/ポート, ユーザー認証情報などを高密度かつ高速に検索・管理できる、セキュアなフルスタック接続情報管理システムです。

---

## 🌟 主な機能・特徴

### 1. 高密度・省スペースのテーブル設計 (High-Density View)
- 大量のホスト情報を1画面に集約して俯瞰・操作可能。
- 表示密度を3段階（`normal` / `dense` / `super-dense`）でリアルタイムに切り替え可能。
- **カラム構成**:
  - `Hostname`: ホスト名（クリックでコピー可能）
  - `IP`: IPアドレス（クリックでコピー可能）
  - `Platform`: プラットフォーム（Linux, Windows, Cisco, MySQL, PostgreSQL 等をスマートなアイコン・バッジで識別）
  - `OS`: OS種別（Ubuntu 24.04, Windows Server 2022, Debian 等。未設定の場合は `---` 表示）
  - `Access`: プロトコル・ポートおよびダイレクトアクセスリンク
  - `User Credentials`: 登録ユーザー・パスワード一覧（マスキング/開示切替、コピー機能）
  - `Tags`: カンマ区切りのタグバッジ
  - `Description`: ホスト用途や詳細メモ
  - `Actions`: ホスト情報の編集・削除（管理者専用）
- ソート対応: ホスト名、IPアドレス、プラットフォーム、OS、タグ、最終更新日時での昇順・降順ソート。

### 2. クリーンなPlatform表示 & OS項目の管理
- プラットフォーム名からプロトコル表記（旧: `Linux(SSH)` など）を排除し、シンプルなプラットフォーム種別を表示。
- Platformの隣に **OS** 入力欄を配置。OS名（例: `Ubuntu 24.04 LTS`, `RHEL 9`, `Windows Server 2022` 等）を自由形式で入力可能（空欄も許可）。

### 3. AccessList (プロトコル・ポート管理 & ダイレクトアクセス)
- 単一ポートの制約をなくし、1ホストに対して複数の接続プロトコルとポート（例: `http(3000)`, `https(8080)`, `ssh(10022)`）を管理可能。
- **HTTP / HTTPS URL生成 & 外部リンク機能**:
  - Webサービス（HTTP/HTTPS）には任意のパス（例: `/app`, `/admin`）を設定可能。
  - テーブル上に完全なURLが表示され、ポップアップアイコン（外部リンク）をクリックすることで、ブラウザの新規タブで対象Webコンソールに直接アクセス可能。

### 4. 柔軟なユーザー認証情報 (User Credentials) 管理
- 1つのホストに対して複数のユーザーアカウント（Username / Password）を登録可能。
- **認証情報なし（0件）のホスト登録に対応**: 認証情報を必要としないホストや未設定ホストもそのまま保存可能。
- 編集・作成フォーム上でユーザー行の追加および個別削除（ゴミ箱アイコン）が手軽に行えます。

### 5. セキュアなパスワード視認制御 (Secure Password Visibility)
- パスワードはデフォルトでマスキング（`••••••••`）表示。
- **15秒タイマー開示**: 目玉アイコンをクリックすると15秒間のみ平文が表示され、秒数カウントダウン後に自動でマスキング状態に戻ります。
- **ワンクリックコピー**: パスワードやユーザー名、ホスト名、IPアドレスを1クリックで安全にクリップボードへコピー。

### 6. パスワードジェネレータ内蔵 (Smart Password Builder)
- 長さ調整（8〜32文字）、英大文字・小文字・数字・記号の有無を柔軟に制御できる安全なパスワード生成エンジンを搭載。
- パスワード強度判定（Weak / Medium / Strong / Excellent）をリアルタイムに視覚的フィードバック。
- サイドバーの独立ツールとして利用できるほか、ホスト編集フォーム内でも直接パスワードを生成・反映可能。

### 7. ロールベースアクセス制御 (RBAC) & セッション認証
- **ログイン機能**: `admin`（管理者）と `user`（一般ユーザー）の2つのロールによるアクセス制御。
  - `admin`: ホスト情報の閲覧、新規追加、編集、削除、CSVインポート/エクスポートなど全操作が可能。
  - `user`: ホスト一覧の閲覧、パスワードの確認・コピー、パスワードジェネレータの利用（編集・削除・インポート・エクスポートは制限）。
- セキュアなHTTP-only Cookieによるセッション管理とログアウト機能。

### 8. IPアドレス制限 & CLI/FZF連携
- `data/config.toml` に指定した許可IPリスト（`permit_ip_list`）に基づくクライアントアクセス制限機能を実装。
- **CLI/FZF連携API (`/api/ssh-fzf`)**: シェルスクリプトやfzfなどのターミナルツールから、マスターパスワードを用いてホストの認証情報を直接クエリ可能。

### 9. TOMLデータストア & CSVインポート/エクスポート
- **TOMLによる分離保存**:
  - `data/hostlist.toml`: ホストメタデータ（ホスト名、IP、プラットフォーム、OS、タグ、説明、AccessList）。
  - `data/host_credentials.toml`: 各ホストに紐付くユーザー名・パスワード情報。
  - `data/config.toml`: 許可IPアドレスおよび各種認証パスワード設定。
- **CSVインポート**:
  - ドラッグ＆ドロップまたはファイル選択に対応。
  - 既存データに追記する「マージ (Merge)」と、全データを置き換える「上書き (Overwrite)」の2モードを選択可能。
- **CSVエクスポート**:
  - 1クリックで現在の全ホストデータ（OS、AccessListのJSON表現を含む）をCSV形式でバックアップダウンロード。

### 10. 自動TLS / HTTPS 対応
- サーバー起動時に `cert/cert.pem` および `cert/key.pem` を自動検出し、証明書が存在する場合は自動的に **HTTPSモード (TLS)** で起動。
- 証明書が存在しない場合は自動的に通常の **HTTPモード** で起動。
- ローカル証明書作成スクリプト (`cert/create_certs.sh`) や、ブラウザ接続時のTLSエラーに関するドキュメント (`docs/about_cert_err.md`) を完備。

### 11. 単一バイナリ (Single Binary) アーキテクチャ
- Goの `go:embed` 機能を使用し、Reactフロントエンドのビルド成果物 (`front/dist`) をGoバイナリ内に完全内包。
- 外部の静的ファイル配置やNode.jsランタイム不要で、1つの実行バイナリのみで完結して動作。

---

## 🛠️ 技術スタック

| レイヤー | 使用技術 |
| :--- | :--- |
| **Frontend** | React (v19) + Vite (v8) + TypeScript |
| **Styling** | Tailwind CSS (v4) |
| **Icons** | Lucide React |
| **Backend** | Go (1.26+) + Echo (v4) |
| **Data Format** | TOML ([go-toml/v2](https://github.com/pelletier/go-toml)), CSV (PapaParse / 標準 `encoding/csv`) |
| **Packaging** | `go:embed` によるフロントエンド完全内包の単一バイナリ |
| **Container** | Docker (Multi-stage build) / Docker Compose |

---

## 📁 ディレクトリ構成

```text
├── cert/                     # TLS/HTTPS証明書および生成スクリプト
│   ├── cacert.pem            # ローカルCA証明書
│   ├── cert.pem              # サーバーSSL証明書
│   ├── key.pem               # サーバー秘密鍵
│   └── create_certs.sh       # 証明書自己署名生成スクリプト
├── data/                     # TOMLデータストア
│   ├── config.toml           # 許可IPリストおよびシステムパスワード設定
│   ├── hostlist.toml         # ホスト一覧およびアクセスリスト情報
│   └── host_credentials.toml# 各ホストのユーザー認証情報
├── docker/                   # Docker関連設定
│   ├── Dockerfile            # マルチステージビルドDockerfile
│   ├── docker-compose.yml    # コンテナ定義
│   └── start.sh / stop.sh    # 起動・停止スクリプト
├── docs/                     # 詳細ドキュメント
│   └── about_cert_err.md     # TLSハンドシェイクエラーの原因と対処法
├── front/                    # Reactフロントエンド (Vite + TypeScript)
│   ├── src/
│   │   ├── components/       # UIコンポーネント (CredentialForm, Login, etc.)
│   │   ├── manager.tsx       # メインマネージャーUI
│   │   └── types.ts          # データ型定義
│   └── package.json
├── go_src/                   # Goバックエンドソースコード
│   ├── credentials/          # パスワード生成ロジック
│   ├── db/                   # TOML/CSVデータベースアクセス・初期シード
│   ├── models/               # Goデータ構造体定義
│   └── server/               # Echoルーティング、ミドルウェア、RBAC認証
├── build_linux_one_binary.sh # Linux向け単一バイナリ作成スクリプト
├── dev.sh                    # 開発用同時起動スクリプト (Air + Vite)
├── main.go                   # アプリケーションエントリーポイント
└── README.md
```

---

## ⚙️ 設定ファイル (`data/config.toml`)

初回起動時に `./data/config.toml` が自動生成されます。必要に応じて許可IPやパスワードを変更してください：

```toml
# アクセスを許可するクライアントのIPアドレス一覧
permit_ip_list = ['127.0.0.1', '192.168.0.70', '192.168.0.79']

# 管理者パスワード (ログインユーザー名: admin)
admin_password = 'admin'

# 一般ユーザーパスワード (ログインユーザー名: user)
user_password = 'user'

# CLI / FZF 照会API用のマスターパスワード
master_password = 'password'
```

---

## 🚀 起動方法

### 1. 開発環境での起動 (Live Reload)

Goのライブリロードツール [Air](https://github.com/air-verse/air) と Vite 開発サーバーを同時に立ち上げるシェルスクリプトが用意されています：

```bash
# フロントエンド依存関係のインストール（初回のみ）
cd front && npm install && cd ..

# 開発サーバーの一括起動
./dev.sh
```

または個別に起動する場合：
```bash
# ターミナル1: フロントエンド開発サーバー (http://localhost:5173)
cd front && npm run dev

# ターミナル2: Goバックエンド (http://localhost:8080)
NODE_ENV=development go run main.go
```

---

### 2. プロダクション用 単一バイナリのビルド & 実行 (Linux / macOS)

Viteでフロントエンドをビルドした後、Goバイナリに成果物を埋め込み、単一の実行ファイルを生成します。

#### ワンクリックビルドスクリプトを使用する場合：
```bash
./build_linux_one_binary.sh
```

#### 手動でビルドする場合：
```bash
# 1. フロントエンドのビルド
cd front
npm run build
cd ..

# 2. Goバイナリのビルド
go build -o host-credential-manager main.go

# 3. 実行
./host-credential-manager
```

起動後、ブラウザで `http://localhost:8080`（証明書がある場合は `https://localhost:8080`）にアクセスします。ポート番号を変更したい場合は環境変数 `PORT` を指定します（例: `PORT=9000 ./host-credential-manager`）。

---

### 3. Dockerでの起動

Dockerコンテナとしてコンパイル・実行することも可能です：

```bash
cd docker
docker compose up -d
```
起動スクリプト `./docker/start.sh` および停止スクリプト `./docker/stop.sh` も利用できます。

---

## 🔒 セキュリティと証明書について

- **HTTPSモード**: `cert/cert.pem` および `cert/key.pem` が配置されている場合、サーバーは自動的にTLS暗号化通信で起動します。
- **TLSハンドシェイクエラーについて**:
  自己署名証明書を使用している場合、ブラウザアクセス時に `tls: unknown certificate` というエラーログが出力されることがあります。詳細な原因および対処方法については [docs/about_cert_err.md](docs/about_cert_err.md) をご参照ください。
- **HTTPモードで動作させたい場合**:
  `cert/cert.pem` または `cert/key.pem` の名前を変更するか削除してサーバーを再起動すると、自動的にHTTPモードで動作します。
