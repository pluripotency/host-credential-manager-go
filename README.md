# Host Credential Manager (ホスト接続情報管理システム)

hostname, IPアドレス, プラットフォーム, OS, アクセスプロトコル/ポート, ユーザー認証情報などを高密度かつ高速に検索・管理できる、セキュアなフルスタック接続情報管理システムです。

---

## 🚀 クイックスタート & 開発・ビルド方法

### 1. `./dev.sh` での開発サーバー起動 (Live Reload)

Goのライブリロードツール [Air](https://github.com/air-verse/air) と Vite 開発サーバーを同時に立ち上げるシェルスクリプトが用意されています：

```bash
# フロントエンド依存関係のインストール（初回のみ）
cd front && npm install && cd ..

# 開発サーバーの一括起動
./dev.sh
```
- 起動後、ブラウザで `http://localhost:5173` (Vite) または `http://localhost:8080` (Go バックエンドプロキシ) にアクセスします（証明書が存在する場合は `https://localhost:8080`）。
- バックエンド Go コードの編集時は Air が自動検知して即座に再コンパイル・再起動し、フロントエンドは Vite HMR によりブラウザが自動更新されます。

---

### 2. ワンバイナリにする方法 (Single Binary Build)

Vite でフロントエンドをビルドした後、Goの `go:embed` 機能によりビルド成果物 (`front/dist`) をバイナリ内に直接埋め込み、単一の自己完結実行可能ファイル（One Binary）を作成します。

#### ワンクリックビルドスクリプトを使用する場合：
```bash
./build_linux_one_binary.sh
```

#### 手動でビルドする場合：
```bash
# 1. フロントエンドのビルド
cd front && npm run build && cd ..

# 2. Go単一バイナリのビルド (埋め込み)
go build -o host-credential-manager main.go

# 3. 実行
./host-credential-manager
```
- 外部の静的ファイル配置や Node.js ランタイムは不要で、生成された `./host-credential-manager` 単体のみで完全動作します。
- 待ち受けポートを変更したい場合は環境変数 `PORT` を指定します（例: `PORT=9000 ./host-credential-manager`）。

---

### 3. Dockerでの起動方法

Multi-stage build による最小構成の Alpine コンテナとしてビルド・実行が可能です：

```bash
cd docker
docker compose up -d
```
起動スクリプト `./docker/start.sh` および停止スクリプト `./docker/stop.sh` も利用できます。

---

### 4. `hcm-client` の開発方法 & ビルド方法

ターミナルからインクリメンタル検索でホストを絞り込み、マスターパスワード認証を経て即座に対象ホストへ SSH / Telnet 接続できる専用 CLI クライアントです。
[`goplur`](../goplur) をベースに pure Go で実装されており、**Python、`fzf`、`sshpass` への依存が一切不要**で、単一の静的バイナリとして配布・実行できます。

#### 特徴
- **外部依存ゼロ**: Python、`fzf`、`sshpass` は不要。標準ターミナルで対話型TUI（リアルタイム絞り込み・カーソル選択）が動作します。
- **SSH & Telnet 両対応**: Linux / AlmaLinux 9 等への SSH 接続だけでなく、Cisco等のネットワーク機器への Telnet 接続（Ctrl+] によるエスケープ切断対応）もサポート。
- **CA証明書内蔵 (One-Binary 配布)**: ビルド時に `cert/cacert.pem` がバイナリに自己内包（`//go:embed`）されるため、自己署名CA環境下でも追加ファイル不要で単体動作します。

#### ビルド方法
`hcm-client/build.sh` を実行するだけで、Root CA証明書を取り込んで単一実行バイナリが生成されます：
```bash
./hcm-client/build.sh
# ./hcm-client/hcm-client に単一バイナリが生成されます
```

#### 実行方法
- **登録されているCLI対象一覧の取得確認 (接続を行わずに対象・ポート・ユーザー・プロトコルの一覧をテスト)**:
  ```bash
  ./hcm-client/hcm-client --list
  ```

- **対話型 TUI による接続**:
  ```bash
  ./hcm-client/hcm-client
  ```
  文字を入力するとリアルタイムにインクリメンタル検索が行われます。上下矢印キーで対象を選択し、Enter キーを押すとマスターパスワード入力後に自動接続されます。

- **接続先サーバーURLや証明書を明示的に指定して実行する場合**:
  ```bash
  ./hcm-client/hcm-client --url https://127.0.0.1:8080 --cert cert/cacert.pem
  ```

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

### 8. IPアドレス制限 & 専用CLIクライアント (`./hcm-client` / goplur連携)
- `data/config.toml` に指定した許可IPリスト（`permit_ip_list`）に基づくクライアントアクセス制限機能を実装。
- **SSH/Telnet対象ホスト自動リスト取得 (`GET /api/ssh-fzf`)**:
  - `Accesslist` に `ssh` または `telnet`（ポート番号問わず）が設定されているホストを自動抽出。
  - ホストと登録ユーザーの組み合わせ（ホスト名, IP, ポート, ユーザー名, プロトコル, プラットフォーム, OS）を一覧で返却。
- **専用CLIクライアント ([`./hcm-client`](file:///home/worker/Documents/antigravity/host-credential-manager-go/hcm-client))**:
  - `goplur` をベースにした pure Go のワンバイナリクライアント。
  - プロジェクトの証明書 (`cert/cacert.pem`) を自己内包して安全にHCMサーバーと通信。
  - リアルタイムインクリメンタル検索で接続先ホスト・ユーザーを選択。
  - マスターパスワードを入力することで安全に認証パスワードを取得し、SSH または Telnet で対象サーバーへ一発接続。

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
├── hcm-client/               # 専用CLIクライアント (Go / goplur連携)
│   ├── build.sh              # hcm-client ビルドスクリプト
│   ├── cert/                 # 埋め込み用証明書 (cacert.pem)
│   └── main.go               # hcm-client エントリーポイント
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

## 🔒 セキュリティと証明書について

- **HTTPSモード**: `cert/cert.pem` および `cert/key.pem` が配置されている場合、サーバーは自動的にTLS暗号化通信で起動します。
- **TLSハンドシェイクエラーについて**:
  自己署名証明書を使用している場合、ブラウザアクセス時に `tls: unknown certificate` というエラーログが出力されることがあります。詳細な原因および対処方法については [docs/about_cert_err.md](docs/about_cert_err.md) をご参照ください。
- **HTTPモードで動作させたい場合**:
  `cert/cert.pem` または `cert/key.pem` の名前を変更するか削除してサーバーを再起動すると、自動的にHTTPモードで動作します。
