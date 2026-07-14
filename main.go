package main

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"host-credential-manager-go/go_src/db"
	"host-credential-manager-go/go_src/server"
)

//go:embed front/dist
var embeddedFiles embed.FS

func getFileSystem() http.FileSystem {
	fsys, err := fs.Sub(embeddedFiles, "front/dist")
	if err != nil {
		panic(err)
	}
	return http.FS(fsys)
}

func newViteProxyHandler(targetURL string) echo.HandlerFunc {
	target, err := url.Parse(targetURL)
	if err != nil {
		panic(err)
	}
	
	proxy := httputil.NewSingleHostReverseProxy(target)

	return func(c echo.Context) error {
		proxy.ServeHTTP(c.Response().Writer, c.Request())
		return nil
	}
}

func main() {
	if err := db.InitDatabase(); err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		return
	}
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Register API routes & middleware
	server.RegisterRoutes(e)

	nodeEnv := os.Getenv("NODE_ENV")
	if nodeEnv == "development" {
		e.Logger.Info("Running in development mode. Proxying static files to Vite...")
		e.GET("/*", newViteProxyHandler("http://localhost:5173"))
	} else {
		e.Logger.Info("Running in production mode. Serving embedded static files...")
		assetHandler := http.FileServer(getFileSystem())
		e.GET("/*", echo.WrapHandler(assetHandler))
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	certFile := "cert/cert.pem"
	keyFile := "cert/key.pem"
	if _, errCert := os.Stat(certFile); errCert == nil {
		if _, errKey := os.Stat(keyFile); errKey == nil {
			e.Logger.Info("Certificates found. Starting server in HTTPS mode...")
			e.Logger.Fatal(e.StartTLS(":"+port, certFile, keyFile))
			return
		}
	}

	e.Logger.Info("No certificates found. Starting server in HTTP mode...")
	e.Logger.Fatal(e.Start(":" + port))
}
