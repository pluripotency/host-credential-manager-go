package main

import (
	"embed"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/labstack/echo/v4"
)

//go:embed front/dist/*
var embededFiles embed.FS

func getFileSystem() http.FileSystem {
	fsys, err := fs.Sub(embededFiles, "front/dist")
	if err != nil {
		panic(err)
	}
	return http.FS(fsys)
}

// 開発環境（Vite）へのプロキシハンドラーを作成する関数
func newViteProxyHandler(targetURL string) echo.HandlerFunc {
	target, err := url.Parse(targetURL)
	if err != nil {
		panic(err)
	}
	
	// 標準ライブラリの ReverseProxy を生成
	proxy := httputil.NewSingleHostReverseProxy(target)

	return func(c echo.Context) error {
		// リクエストをそのままViteの開発サーバーへ横流しする
		proxy.ServeHTTP(c.Response().Writer, c.Request())
		return nil
	}
}

func main() {
	e := echo.New()

	// 1. APIのエンドポイント（ここは環境に関わらずGoが処理）
	api := e.Group("/api")
	api.GET("/hello", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "Hello from Go!"})
	})

	// 2. 環境変数に応じてフロントエンドのハンドラーを切り替える
	nodeEnv := os.Getenv("NODE_ENV")

	if nodeEnv == "development" {
		// 【開発環境】localhost:5173 (Vite開発サーバー) へプロキシ
		// ※ポートが5173固定なら直書きでOKです
		e.Logger.Info("Running in development mode. Proxying static files to Vite...")
		e.GET("/*", newViteProxyHandler("http://localhost:5173"))
	} else {
		// 【本番環境】embedした静的ファイルを配信
		e.Logger.Info("Running in production mode. Serving embedded static files...")
		assetHandler := http.FileServer(getFileSystem())
		e.GET("/*", echo.WrapHandler(assetHandler))
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	e.Logger.Fatal(e.Start(":" + port))
}
