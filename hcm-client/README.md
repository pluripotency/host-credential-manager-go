# Host Credential Manager CLI Client (`hcm-client`)

`hcm-client` は、Host Credential Manager (HCM) サーバーに登録されたホスト一覧（SSH / Telnet）を取得し、ターミナル上で対話的に検索・選択して安全かつ自動的に接続を行う専用 CLI クライアントです。

[`goplur`](https://github.com/pluripotency/goplur) を基盤エンジンとして採用し、**Python、`requests`、`fzf`、`sshpass` などの外部依存パッケージを一切必要としない Pure Go 製の単一実行バイナリ（One-Binary）** として設計されています。

---

## 📌 プログラムの役割とアーキテクチャ

### 1. 解決する課題
従来のクライアントは Python スクリプトで実装されており、実行環境に Python 3、`requests` ライブラリ、`fzf`、`sshpass` がインストールされている必要がありました。本 Go 版 `hcm-client` ではこれらをすべて Go の標準機能および `goplur` に統合し、以下を実現しています：
- **ゼロ依存性**: OS の標準端末さえあれば追加ツールなしで動作。
- **SSH & Telnet 両対応**: Linux/UNIX サーバーへの SSH 接続に加え、Cisco 等のネットワーク機器への Telnet 接続（プロンプト自動応答・エスケープ切断対応）もサポート。
- **自己完結型 TLS 通信**: ビルド時に HCM サーバーの Root CA 証明書 (`cert/cacert.pem`) をバイナリ内に `//go:embed` することで、自己署名 CA 環境下でも単一ファイルで証明書検証を通過。

### 2. アーキテクチャの概要
1. **API 通信**:
   - `GET /api/ssh-fzf`（または `/api/targets`）へリクエストを送信し、SSH / Telnet 接続が可能なノード一覧（ホスト名、IP、ポート、ユーザー名、プロトコル、OS、タグ）を取得。
2. **対話型 TUI (インクリメンタル検索)**:
   - `golang.org/x/term` の Raw モードを用いて、端末上でリアルタイムなインクリメンタルフィルタリング（複数トークン AND 検索）と上下キーによるカーソル選択を提供。
3. **マスターパスワード照合 & パスワード取得**:
   - 選択されたホストに対して `POST /api/ssh-fzf` を送信し、HCM サーバーのマスターパスワード照合を経て対象ノードのログインパスワードを取得（画面やディスクには一切残しません）。
4. **セッション接続 (`goplur`)**:
   - **SSH**: `goplur.NewSshNode` を作成し、Pty 経由でログインパスワードを自動送出して `s.Interact()` へ移行。
   - **Telnet**: `goplur.NewTelnetNode` を作成し、Telnet ログインシーケンスおよび `Ctrl+]` エスケープ切断ハンドラを登録して対話セッションを起動。

---

## 🛠️ ビルド方法

### 1. ビルドスクリプトによるビルド（推奨）
リポジトリのルートまたは `hcm-client` ディレクトリから `build.sh` を実行します：

```bash
# hcm-client ディレクトリ内で実行する場合
./build.sh

# プロジェクトルートから実行する場合
./hcm-client/build.sh
```

このスクリプトは以下の処理を自動で行います：
1. `cert/cacert.pem`、`cert/client_cert.pem`、`cert/client_key.pem` を検出し、バイナリ埋め込み用 (`hcm-client/cert/`) および配布用 (`hcm-client/built/cert/`) に同期。
2. Go コンパイラで最適化・シンボル除去（`-ldflags="-s -w"`）を行い、`hcm-client/built/hcm-client` に単一バイナリを出力。
3. 実行権限を付与。

### 2. 手動・クロスプラットフォームコンパイル (macOS / Windows)
事前に証明書ファイルを `hcm-client/cert/` に配置した上で、ターゲット OS を指定してビルドします：

```bash
# 証明書の同期（初回または証明書更新時）
mkdir -p hcm-client/cert
cp cert/cacert.pem cert/client_cert.pem cert/client_key.pem hcm-client/cert/

# Linux x86_64 向けビルド
go build -ldflags="-s -w" -o hcm-client/built/hcm-client ./hcm-client

# macOS (Apple Silicon / M1, M2, M3) 向けビルド
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o hcm-client/built/hcm-client-darwin-arm64 ./hcm-client

# macOS (Intel CPU) 向けビルド
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o hcm-client/built/hcm-client-darwin-amd64 ./hcm-client

# Windows 向けビルド (.exe)
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o hcm-client/built/hcm-client.exe ./hcm-client
```

---

## 📖 詳細な使用方法

### 1. コマンドラインオプション

```
Usage of ./hcm-client:
  -url string
        HCM サーバーのベースURL (デフォルト: https://127.0.0.1:8080 または環境変数 $HCM_URL)
  -cert string
        CA/サーバー証明書ファイルのパス (省略時はバイナリ内蔵の cert/cacert.pem を使用)
  -client-cert string
        mTLS クライアント証明書ファイルのパス (省略時はバイナリ内蔵の cert/client_cert.pem を使用)
  -client-key string
        mTLS クライアント秘密鍵ファイルのパス (省略時はバイナリ内蔵の cert/client_key.pem を使用)
  -insecure
        TLS 証明書の検証をスキップ
  -list
        対象ホスト一覧を整形表示して即座に終了 (接続テスト用)
  -h, --help
        ヘルプメッセージを表示
```

### 2. 環境変数

コマンドライン引数の代わりに環境変数でデフォルト動作を設定できます：

| 環境変数 | 説明 | デフォルト値 |
| :--- | :--- | :--- |
| `HCM_URL` | HCM サーバーのベース URL | `https://127.0.0.1:8080` |
| `HCM_CERT` | CA 証明書ファイルのパス | （内蔵の埋め込み証明書） |
| `HCM_CLIENT_CERT` | mTLS クライアント証明書のパス | （内蔵の埋め込みクライアント証明書） |
| `HCM_CLIENT_KEY` | mTLS クライアント秘密鍵のパス | （内蔵の埋め込みクライアント秘密鍵） |
| `HCM_INSECURE` | `1` または `true` で TLS 検証をスキップ | `false` |

### 3. 主な実行パターン

#### パターン A: 配布パッケージの起動スクリプトを使用
```bash
cd built
./run.sh

# 接続先サーバー IP を指定する場合
SERVER_IP=192.168.1.100 ./run.sh
```

#### パターン B: 登録ターゲットの一覧確認 (`--list`)
サーバーに接続して、取得可能なホストとプロトコル（SSH / TELNET）、ポート、ユーザー名を一覧表示します：
```bash
./built/hcm-client --list
```
出力例：
```
Found 15 CLI targets on https://127.0.0.1:8080:
HOSTNAME                     PROTOCOL USERNAME       IP:PORT              PLATFORM / OS        TAGS
----------------------------------------------------------------------------------------------------
ssh-prod-web01.internal      SSH      ubuntu         10.0.2.14:10022      Linux                production,web,frontend
switch-floor2-core           TELNET   admin          192.168.20.2:23      Cisco                network,switch,cisco
almalinux9-srv01.internal    SSH      deploy         10.0.3.50:22         Linux / AlmaLinux 9  production,database,ssh
...
```

#### パターン C: 対話型 TUI 接続
```bash
./built/hcm-client
```
1. 画面上の `Query>` に文字を入力（例: `cisco` や `almalinux`、`10.0.` など）。
2. 空白区切りで複数ワードを入力することで絞り込みが可能（AND 検索）。
3. `↑` / `↓` キー（または `Ctrl+P` / `Ctrl+N`）で対象ホストを選択。
4. `Enter` キーを押して決定。
5. `Enter masterpassword for HCM server:` とプロンプトが表示されるので、HCM のマスターパスワードを入力。
6. パスワードが検証され、自動で SSH または Telnet セッションが開始されます。

---

## 🔒 セキュリティ、Mutual TLS (mTLS) & 証明書失効運用

### 1. Mutual TLS (mTLS) 認証
- `hcm-client` は、HCM サーバーとの通信時に Root CA 証明書に加え、**クライアント証明書 (`client_cert.pem`, `client_key.pem`)** を送信して双方向認証を行います。
- クライアント証明書はビルド時にバイナリ内部に `//go:embed` されるため、配布されたバイナリ単体で追加設定なく mTLS 認証が行われます。
- CLI エンドポイント（`/api/ssh-fzf`, `/api/targets`）は、正規のクライアント証明書を持たないアクセス（通常のブラウザアクセス等）を拒否（401 Unauthorized）します。

### 2. サーバー証明書・Root CA更新時の旧クライアント自動失効
- HCM サーバー側で `./cert/create_certs.sh` を実行して Root CA およびサーバー証明書を更新すると、サーバー側の `ClientCAs` も一新されます。
- 古い Root CA で署名されたクライアント証明書を持つ既存の `hcm-client` は、**TLS ハンドシェイクの時点でサーバーによって即座に接続拒否 (`unknown certificate authority`)** されます。
- 利用者は Web UI の「hcm-client.tgz」ダウンロードボタン、または `./hcm-client/build.sh` で再ビルドされた新しいバイナリを取得するだけで、新しい証明書で再接続できるようになります。

### 3. CRL（証明書失効リスト）による失効運用
- クライアント証明書が漏洩または退役した場合、CA全体を再生成することなく特定の証明書を無効化できます。
- サーバー管理者は `./cert/revoke_cert.sh` を実行することで `cert/crl.pem` に失効情報を登録します。
- サーバーは TLS 接続時およびエンドポイント受信時にリアルタイムに CRL を参照し、失効された証明書での接続を直ちに遮断 (`tls: bad certificate`) します。

---

## 💡 接続時のヒント & トラブルシューティング

### 1. Telnet 接続時のエスケープ切断
Telnet 接続時に対象機器からログアウトせずに強制切断したい場合は、**`Ctrl + ]`** を入力して `telnet>` プロンプトを表示させた後、`quit` と入力して Enter を押してください。

### 2. 主なエラーと解決手順

| エラー | 原因 | 対処方法 |
| :--- | :--- | :--- |
| `remote error: tls: bad certificate` | クライアント証明書が CRL で失効されている | Web UI にログインして新しい `hcm-client.tgz` を再ダウンロードしてください。 |
| `remote error: tls: unknown certificate authority` | サーバー側で CA が更新された | Web UI にログインして最新の `hcm-client.tgz` を再ダウンロードしてください。 |
| `status 401: Client certificate required (mTLS)` | クライアント証明書なしで接続した | 正規の `hcm-client` を使用しているか、または HTTPS で接続しているか確認してください。 |
| `status 401: Invalid masterpassword` | マスターパスワード不一致 | サーバー側 `data/config.toml` の `masterpassword` を確認してください。 |
| `status 403: Forbidden` | 接続元 IP が許可されていない | サーバー側の `data/config.toml` の `permit_ip_list` に自端末の IP を追加してください。 |

より詳細なエラー対応は [docs/troubleshooting.md](../docs/troubleshooting.md) をご参照ください。
