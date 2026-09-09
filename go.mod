module host-credential-manager-go

go 1.26.4

require (
	github.com/labstack/echo/v4 v4.15.4
	github.com/pelletier/go-toml/v2 v2.4.3
	golang.org/x/term v0.45.0
	goplur v0.0.0-00010101000000-000000000000
)

replace goplur => ../goplur

require (
	github.com/google/goterm v0.0.0-20190703233501-fc88cf888a3f // indirect
	github.com/labstack/gommon v0.5.0 // indirect
	github.com/mattn/go-colorable v0.1.15 // indirect
	github.com/mattn/go-isatty v0.0.22 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	golang.org/x/crypto v0.53.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	golang.org/x/time v0.15.0 // indirect
)
