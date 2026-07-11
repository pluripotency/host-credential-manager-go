#! /bin/bash
# 1. フロントエンドのビルド
cd frontend && npm run build && cd ..

# 2. Windows向けにクロスコンパイル (環境変数を指定)
SET GOOS=windows
SET GOARCH=amd64
go build -o myapp.exe main.go
