# TLSハンドシェイクエラー（unknown certificate）について

`./dev.sh` 等で開発サーバーを起動した際に、以下のようなエラーログが連続して出力される原因と対処方法についての解説です。

```text
echo: http: TLS handshake error from [::1]:38194: remote error: tls: unknown certificate
echo: http: TLS handshake error from [::1]:38208: remote error: tls: unknown certificate
echo: http: TLS handshake error from [::1]:49068: remote error: tls: unknown certificate
echo: http: TLS handshake error from [::1]:49072: remote error: tls: unknown certificate
```

---

## 1. エラーメッセージの意味

* **`remote error`（リモートエラー）**:
  Goサーバー内部で発生したエラーではなく、**接続してきたクライアント側（ブラウザやNode.js等）がTLSハンドシェイクを中断し、サーバーへ返してきたTLSアラート通知**（TLS Alert 46: `unknown_certificate` / `unknown_ca`）です。
* **`tls: unknown certificate`**:
  Goサーバーが提示したSSL/TLS証明書（`cert/cert.pem`）が、クライアント側の信頼できる認証局リスト（OSやブラウザ、Node.js等の信頼ストア）に登録されていない（自己署名 / オレオレ証明書である）ため、クライアントが「信頼できない証明書」と判定して接続を拒否したことを意味します。

---

## 2. どこから接続されているのか (`[::1]`)

* **`[::1]`** はローカル環境の **IPv6 localhost**（`127.0.0.1` のIPv6版）です。外部の第三者ではなく、自分自身のマシン上で動作しているプロセスからの接続です。
* 主な接続元：
  1. **Webブラウザのタブ**:
     * `https://localhost:8080` を開いている（またはバックグラウンドに残っている）タブ。
     * 証明書の警告画面が出たままのタブや、バックグラウンドで定期通信（APIポーリング、Favicon取得、ServiceWorker等）を行っている場合にリクエストが繰り返され、ログが出続けます。
  2. **Vite 開発サーバーや Node.js / 開発ツール**:
     * `dev.sh` で起動している Vite のプロキシや、IDEのプレビュー機能、ブラウザ拡張機能などが `localhost:8080` にアクセスを試み、証明書検証で失敗している場合。

---

## 3. このプロジェクトでの発生背景

* `main.go` の起動ロジックにより、`cert/cert.pem` および `cert/key.pem` が存在する場合、自動的に **HTTPSモード（TLS）** でポート `8080` が起動します。
* `cert/create_certs.sh` で作成された証明書はローカル専用の認証局（`MyLocalSSHCA`）で自己署名されているため、OSやブラウザにそのCA証明書（`cert/cacert.pem`）がインポート・信頼されていない環境では、アクセスするたびにこのTLS警告が出力されます。

---

## 4. 対処・解決方法

### 方法 A: ブラウザで接続を許可する
* ブラウザで `https://localhost:8080` にアクセスした際、セキュリティ警告画面で「**詳細設定**」→「**localhost に進む（安全ではありません）**」をクリックして接続を許可します。

### 方法 B: ローカルCA証明書を信頼ストアに登録する
* `cert/cacert.pem` を OS またはブラウザの「信頼されたルート証明機関」にインポート・信頼設定します。

### 方法 C: 不要な接続プロセス・タブを閉じる
* バックグラウンドで `localhost:8080` にリクエストを送り続けているブラウザタブや開発ツールを閉じます。

### 方法 D: HTTPモードで起動する
* HTTPS が不要な場合は、`cert/cert.pem` または `cert/key.pem` を一時的にリネームまたは削除してサーバーを再起動すると、自動的に通常の HTTP モードで起動します。
