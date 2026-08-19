#! /bin/bash
# 1. フロントエンドのビルド
cd front && npm run build && cd ..

# 2. Linux向けにコンパイル (環境変数を指定)
export GOOS=linux
export GOARCH=amd64
go build -o host-credential-manager main.go
