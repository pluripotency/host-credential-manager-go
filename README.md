# Host Credential Manager (ホスト接続情報管理システム)

hostname, ip, username, password, platform などの大量の接続情報を高密度かつ高速に検索・管理できる、フルスタックな安全接続情報管理データベースアプリケーションです。

## 🌟 主な機能・特徴

1. **高密度・省スペースのテーブル設計 (High-Density View)**
   - 大量のホスト情報を1画面に整理して表示。
   - レイアウト密度を3段階 (`normal` / `dense` / `super-dense`) でリアルタイムに切り替え可能。
   - 各種プラットフォーム (Linux, Windows, Network, DB, クラウド等) に合わせたスマートなバッジ表示。

2. **インクリメンタルサーチ & タグフィルタ (Real-time Filter & Search)**
   - ホスト名、IPアドレス、ユーザー名、プラットフォーム、タグ、説明文など、あらゆるフィールドを瞬時に対象とする高速な文字入力インクリメンタルサーチ。
   - 各種プラットフォーム種別、Production/Staging などのカテゴリ、及びタグのクリックによる連動フィルタ。

3. **セキュアなパスワード視認制御 (Secure Password Visibility)**
   - パスワードはデフォルトでマスキング表示でコピーや手動表示が可能。
   - **パスワード表示**: 15秒間のみパスワードを表示し、残り時間がタイマーカウントダウンされたあと、自動的に再度マスキングされます。

4. **パスワードジェネレータ内蔵 (Smart Password Builder)**
   - 長さ調整 (8〜32文字)、大文字、小文字、数字、記号の有無をコントロールできる安全な自動パスワード生成機能をサイドバー及び作成・更新フォーム内に内蔵。
   - 生成したパスワードの強度判定 (Weak, Medium, Strong, Excellent) を視覚的にフィードバック。

5. **バックエンド連携CSV保存 (CSV File Datastore)**
   - データストアは高速なローカルCSVファイル (`data/hostlist.csv`) に連動。
   - **インポート機能**: CSVファイルをドラッグ＆ドロップまたは選択して、既存データへのマージ (Merge) もしくは、新規データによる上書き (Overwrite) を選択してアップロード可能。
   - **エクスポート機能**: 1クリックで現在のデータベース全量を標準フォーマットのCSVでバックアップダウンロード。

6. **フルスタック構成 (Vite + Echo)**
   - バックエンド (API/CSV操作) とフロントエンドを安全に連携。本番環境のビルドシステムにも完全対応。

---

## 🛠️ 技術スタック

- **Frontend**: React (v19) + Vite (v8) + TypeScript
- **Styling**: Tailwind CSS (v4)
- **Icons**: Lucide React
- **Backend / API**: Go (1.26+) + Echo (v4) + Go-Toml (v2)
- **Data Parsing**: PapaParse (Frontend) + 標準 `encoding/csv` (Backend)
- **Single Binary**: Goの `go:embed` を活用し、フロントエンドのビルド成果物 (`front/dist`) をGoバイナリ内に直接内包した単一バイナリ (One Binary) 構成

---

## 🚀 起動方法

### 開発環境での起動

開発時には、Goのライブリロードツールである [Air](https://github.com/air-verse/air) と、フロントエンド開発サーバーを別々に立ち上げて開発を行います。

#### 1. フロントエンド開発サーバーの起動
```bash
cd front
npm install
npm run dev
```
起動すると、`http://localhost:5173` でVite開発サーバーが立ち上がります。

#### 2. バックエンド開発サーバーの起動
別のターミナルでルートディレクトリに移動し、以下を実行します：
```bash
# ライブリロードツール（Air）の実行
air
```
または手動で：
```bash
NODE_ENV=development go run main.go
```
起動すると、`http://localhost:8080` でGoバックエンドが立ち上がります。開発環境では、Goサーバーへのアクセス、またはVite開発サーバーへのアクセスのどちらからでも相互に連携します。

---

### プロダクションビルド & 実行 (Linux / macOS)

Vite でフロントエンドをビルドした後、Goバイナリに静的ファイルを埋め込んで単一の実行ファイルをビルドします。

```bash
# 1. フロントエンドのビルド
cd front
npm run build
cd ..

# 2. バイナリのビルド
go build -o host-credential-manager main.go

# 3. 実行 (起動後、 http://localhost:8080 にアクセス)
./host-credential-manager
```

---

## 💻 Windowsでの起動方法

本システムはWindows環境向けに単一の実行可能ファイル（`.exe`）としてコンパイルして動かすことができます。

### 1. Windows用バイナリのビルド (クロスコンパイル)

開発環境（LinuxやmacOS）からWindows用バイナリを一発でビルドするためのシェルスクリプトが用意されています：

```bash
# スクリプトの実行（ビルド完了後、ルートディレクトリに myapp.exe が生成されます）
./build_windows_one_binary.sh
```

### 2. Windows上での起動

1. 生成された `myapp.exe` ファイルを対象のWindowsマシンにコピーします。
2. `myapp.exe` をダブルクリックして実行、またはコマンドプロンプト／PowerShellから以下のように実行します：
   ```cmd
   myapp.exe
   ```
3. 起動後、ブラウザで `http://localhost:8080` にアクセスして利用します。

> [!NOTE]
> アプリケーションの動作に必要なCSVデータベースやTOML設定ファイルは、起動したディレクトリの `./data` フォルダ以下に自動的に初期生成・更新されます。
