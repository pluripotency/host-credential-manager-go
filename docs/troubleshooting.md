# トラブルシューティング & エラー対応ガイド (`troubleshooting.md`)

本書では、Host Credential Manager (HCM) サーバーおよび `hcm-client` の利用・運用時に発生するエラーメッセージ、原因、および具体的な復旧・解決手順について解説します。

---

## 1. エラーメッセージ別 原因・対処法クイックリファレンス

| エラーメッセージ / 症状 | 発生箇所 | 主な原因 | 解決・復旧手順 |
| :--- | :--- | :--- | :--- |
| **`remote error: tls: bad certificate`** | `hcm-client` | 使用中のクライアント証明書が、サーバー側の CRL（失効リスト）に登録され失効している | 管理者に証明書再発行（CA再生成）を依頼するか、Web UI にログインして新しく再生成された `hcm-client.tgz` を再ダウンロードして差し替える。 |
| **`remote error: tls: unknown certificate authority`** | `hcm-client` | サーバー側で CA が更新され、古い Root CA で署名された旧クライアント証明書が拒否された | Web UI にログインし、新しい CA/証明書が埋め込まれた最新の `hcm-client.tgz` を再ダウンロードする。 |
| **`x509: certificate signed by unknown authority`** | `hcm-client` | クライアントが保持する CA 証明書と、接続先サーバーが提示した証明書の CA が一致しない | 1. 接続先 `--url` が正しいか確認する。<br>2. サーバー証明書が更新されている場合は最新の `hcm-client.tgz` を再ダウンロードする。 |
| **`status 401: {"error":"Client certificate required (mTLS)"}`** | `hcm-client` / curl | mTLS 必須エンドポイント（`/api/ssh-fzf` 等）にクライアント証明書なしでアクセスした | 1. プレーンな HTTP ではなく HTTPS でアクセスしているか確認。<br>2. curl の場合は `--cert` と `--key` を指定する。<br>3. `hcm-client` の場合は証明書が埋め込まれた正規バイナリを使用する。 |
| **`status 401: Invalid masterpassword`** | `hcm-client` | 入力したマスターパスワードが、サーバー側 `data/config.toml` の `masterpassword` と一致しない | 正しいマスターパスワードを入力する。忘れた場合はサーバー管理者に `data/config.toml` の値を確認してもらう。 |
| **`status 403: Forbidden`** | ブラウザ / `hcm-client` | クライアントの IP アドレスがサーバーの `permit_ip_list` に含まれていない | サーバー管理者側で `data/config.toml` の `permit_ip_list` 配列に接続元のクライアント IP アドレスを追加し、サーバーを再起動する。 |
| **`connection refused` / 接続タイムアウト** | ブラウザ / `hcm-client` | HCM サーバープロセスが停止している、ポート番号が異なる、またはファイアウォールで遮断されている | 1. サーバー上でプロセスが起動しているか確認（`ss -tulpn \| grep 8080`）。<br>2. ポート番号（デフォルト: 8080）および OS のファイアウォール（firewalld, ufw 等）の開放状況を確認する。 |
| **`TLS handshake error ... remote error: tls: unknown certificate`** | サーバー側ログ | ブラウザやツールが自己署名 CA（`MyLocalSSHCA`）を信頼していないため、クライアント側が TLS を中断した | 実害はありません。詳細は [docs/about_cert_err.md](about_cert_err.md) を参照し、ブラウザで証明書の例外許可を行うか、CA を OS 信頼ストアにインポートしてください。 |

---

## 2. クライアント (`hcm-client`) の復旧手順

サーバー管理者による CA の更新（ローテーション）や証明書の失効（CRL 更新）が発生した場合、既存の `hcm-client` は安全のため TLS レベルで自動的に遮断されます。

### 利用者の復旧ステップ:
1. Web ブラウザで HCM サーバーの Web UI（例: `https://<server-ip>:8080`）を開きます。
   - ※ Web UI はクライアント証明書なしでアクセス可能です。
2. ユーザー名（`admin` または `user`）とパスワードでログインします。
3. 画面右上のクイックアクションにある **「hcm-client.tgz」** ダウンロードボタンをクリックします。
   - サーバーは最新の証明書をもとに自動で再ビルドしたパッケージを提供します。
4. ダウンロードした `hcm-client.tgz` を端末上で展開します：
   ```bash
   tar -xzvf hcm-client.tgz
   cd hcm-client
   ./run.sh
   ```
5. これで最新の証明書で即座に再接続が可能になります。

---

## 3. SSH / Telnet 接続時の操作トラブル

### 3.1 Telnet セッションのエスケープ切断
- Cisco 機器などの Telnet 接続時、プロンプトが応答しなくなったり、ログアウトせずに即座にセッションを強制切断したい場合：
  - **エスケープキー入力**: キーボードで `Ctrl + ]`（Ctrl キーを押しながら角括弧の閉じキー）を押します。
  - `telnet>` プロンプトが表示されたら、`quit` と入力して Enter を押すと接続が終了します。

### 3.2 SSH 接続タイムアウト
- `hcm-client` はパスワード取得後、直接対象ホスト（IP:Port）への TCP/SSH 接続を開始します。
- ターゲットホストが停止している、またはファイアウォールでブロックされている場合は接続タイムアウトが発生します。HCM サーバーではなく、**クライアント端末から対象ホストへのネットワークルーティングが確立しているか**（Ping や `nc -zv <target-ip> <port>` 等）をご確認ください。

---

## 4. クロスプラットフォーム対応 (macOS / Windows)

プロジェクト同梱の `hcm-client/build.sh` は Linux 向けにバイナリをビルドしますが、Go のクロスコンパイル機能を利用することで、macOS や Windows 向けのバイナリを簡単に生成できます。

```bash
# プロジェクトルートで実行

# 1. 事前に証明書一式を hcm-client/cert/ にコピー
mkdir -p hcm-client/cert
cp cert/cacert.pem cert/client_cert.pem cert/client_key.pem hcm-client/cert/

# 2. macOS (Apple Silicon / M1, M2, M3) 向けビルド
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o hcm-client-darwin-arm64 ./hcm-client

# 3. macOS (Intel CPU) 向けビルド
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o hcm-client-darwin-amd64 ./hcm-client

# 4. Windows 向けビルド (.exe)
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o hcm-client.exe ./hcm-client
```

> [!NOTE]
> Windows 環境で `hcm-client.exe` を実行する場合は、従来のコマンドプロンプト (`cmd.exe`) よりも、ANSI エスケープシーケンスおよび Raw モードに完全対応している **Windows Terminal** または **PowerShell 7** 上での実行を推奨します。
