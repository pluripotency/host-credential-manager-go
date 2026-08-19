#! /bin/bash
# 1. フロントエンドのビルド
cd front && npm run build && cd ..

# 2. Windows向けにクロスコンパイル (環境変数を指定)
export GOOS=windows
export GOARCH=amd64
go build -o host-credential-manager.exe main.go
