package webx

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/tkdeng/goutil"
	"github.com/tkdeng/regex"
)

type Config struct {
	Title    string
	AppTitle string
	Desc     string
	Icon     string

	PublicURI string

	Origins []string
	Proxies []string

	Vars Map

	PortHTTP uint16
	PortSSL  uint16

	DebugMode bool

	Root string

	CSP     bool
	csp     CSP
	cspText string
}

type CSP struct {
	DefaultSrc             string
	ScriptSrc              string
	StyleSrc               string
	ImgSrc                 string
	ObjectSrc              string
	FontSrc                string
	ConnectSrc             string
	BaseUri                string
	FormAction             string
	FrameAncestors         string
	RequireTrustedTypesFor string
	ReportUri              string
}

type App struct {
	*fiber.App
	Config Config

	compiler *compiler

	hasFailedSSL bool
}

type Map map[string]string

var Helmet helmet.Config

// New loads a new server
func New(root string, config ...fiber.Config) (App, error) {
	appConfig := Config{
		Title:    "Web Server",
		AppTitle: "WebServer",
		Desc:     "A Web Server.",

		PortHTTP: 8080,
		PortSSL:  8443,
	}

	// load config file
	loadConfig(root, &appConfig)

	// compile src
	compiler := compile(&appConfig)

	if len(config) == 0 {
		config = append(config, fiber.Config{
			AppName:      appConfig.AppTitle,
			ServerHeader: appConfig.Title,
			TrustProxyConfig: fiber.TrustProxyConfig{
				Proxies: appConfig.Proxies,
			},
			TrustProxy:         true,
			EnableIPValidation: true,
		})
	} else {
		config[0].AppName = appConfig.AppTitle
		config[0].ServerHeader = appConfig.Title

		if config[0].TrustProxyConfig.Proxies == nil {
			config[0].TrustProxyConfig.Proxies = appConfig.Proxies
		} else {
			config[0].TrustProxyConfig.Proxies = append(config[0].TrustProxyConfig.Proxies, appConfig.Proxies...)
		}
	}

	app := App{
		App:    fiber.New(config[0]),
		Config: appConfig,

		compiler: compiler,
	}

	app.Use(helmet.New(Helmet))

	// preform header sanity check to reduce potential bot spam
	app.Use(app.verifyHeaders())

	// enforce specific domain and ip origins
	app.Use(app.verifyOrigin(appConfig.Origins, appConfig.Proxies))

	// auto redirect http to https
	if appConfig.PortSSL != 0 {
		app.Use(app.redirectSSL(appConfig.PortHTTP, appConfig.PortSSL))
	}

	compressAssets := !appConfig.DebugMode
	app.Get("/theme/*", static.New(appConfig.Root+"/theme", static.Config{Compress: compressAssets}))
	app.Get("/assets/*", static.New(appConfig.Root+"/assets", static.Config{Compress: compressAssets}))
	if appConfig.PublicURI != "" {
		app.Get(appConfig.PublicURI, static.New(appConfig.Root+"/public", static.Config{Compress: compressAssets, Browse: true}))
	}

	// reduce bot spam on post requests
	app.Post("/api/*", app.BlockBotHeader)
	app.Post("/apis/*", app.BlockBotHeader)

	// app.Use("/*", static.New(appConfig.Root+"/dist", static.Config{Compress: compressAssets}))

	app.Use("/*", func(c fiber.Ctx) error {
		url := goutil.Clean(c.Path())
		if url == "/404" {
			return c.Next()
		}

		// dont render @widgets
		if regex.Comp(`/@[^\\/]*?$`).Match([]byte(url)) {
			return c.Next()
		}

		return app.Render(c, url)
	})

	return app, nil
}

// Listen to both http and https ports and
// auto generate a self signed ssl certificate
// (will also auto renew every year)
//
// by using self signed certs, you can use a proxy like cloudflare and
// not have to worry about verifying a certificate athority like lets encrypt
func (app *App) Listen() error {
	app.Use(func(c fiber.Ctx) error {
		return app.Error(c, 404, "Page Not Found")
	})

	if DebugCompiler {
		return errors.New("DebugCompiler is enabled, please disable it before running the server")
	}

	return app.listenAutoTLS(app.Config.PortHTTP, app.Config.PortSSL, app.Config.Root+"/db/ssl/auto_ssl")
}

// Compile runs the compiler without loading a new server
func Compile(root string) {
	appConfig := Config{
		Title:    "Web Server",
		AppTitle: "WebServer",
		Desc:     "A Web Server.",

		PortHTTP: 8080,
		PortSSL:  8443,
	}

	// load config file
	loadConfig(root, &appConfig)

	// compile src
	compile(&appConfig)
}

func loadConfig(root string, config *Config) {
	// load config file
	if path, err := filepath.Abs(root); err == nil {
		root = path
	}
	root = strings.TrimSuffix(root, "/")

	goutil.ReadConfig(root+"/config.yml", &config)
	config.Root = root
}

// Render a page
//
// if the page is not found, it will return a 404 error
func (app *App) Render(c fiber.Ctx, url string, vars ...Map) error {
	if url == "/" || url == "" {
		url = "index"
	}
	url = strings.TrimPrefix(url, "/")

	path, err := goutil.JoinPath(app.Config.Root+"/dist", url)
	if err != nil {
		return c.Next()
	}
	path += ".html.gz"

	useGzip := true
	if _, err := os.Stat(path); err != nil {
		useGzip = false
		path = strings.TrimSuffix(path, ".gz")
	}

	// check for `#page.html` to load dynamic nonce keys
	if _, err := os.Stat(path); err != nil {
		cPath := string(regex.Comp(`\/([^\/]+)$`).Rep([]byte(path), []byte("/#$1")))
		if buf, err := os.ReadFile(cPath); err == nil {
			nonceKey := goutil.RandBytes(16)

			buf = regex.Comp(`{nonce}`).RepLit(buf, nonceKey)

			cspValue := regex.Comp(`'nonce(-.*|)'`).RepLit([]byte(app.Config.cspText), []byte(`'nonce-`+string(nonceKey)+`'`))
			c.Set(fiber.HeaderContentSecurityPolicy, string(cspValue))

			c.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
			c.SendStatus(200)
			return c.Send(buf)
		}
	}

	if _, err := os.Stat(path); err != nil {
		return c.Next()
	}

	// unzip if gzip is not supported by the browser
	if useGzip {
		if c.Get(fiber.HeaderAcceptEncoding) != "gzip" {
			if buf, err := Gunzip(path); err == nil {
				c.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
				c.SendStatus(200)

				return c.Send(buf)
			}

			// return 500 error
			return app.Error(c, 500, "Internal Server Error")
		}
	}

	if url[0] == '@' {
		buf, err := os.ReadFile(path)
		if err != nil {
			return app.Error(c, 500, "Internal Server Error")
		}

		c.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
		c.SendStatus(200)

		if len(vars) == 0 {
			vars = append(vars, Map{})
		}

		app.compiler.compileDynamicPage(&buf, vars[0])

		return c.Send(buf)
	}

	c.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
	c.SendStatus(200)

	return c.SendFile(path)
}

// Error renders an error page
//
// if the page is not found, it will return a default error page
func (app *App) Error(c fiber.Ctx, status uint16, msg string) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
	c.SendStatus(int(status))

	path, err := goutil.JoinPath(app.Config.Root+"/dist", "@"+strconv.FormatUint(uint64(status), 10)+".html")
	if err != nil {
		return c.SendString("<h1>Error " + strconv.FormatUint(uint64(status), 10) + "</h1><h2>" + msg + "</h2>")
	}

	dynamic := false
	if _, err := os.Stat(path); err != nil {
		dynamic = true
		path, err = goutil.JoinPath(app.Config.Root+"/dist", "@error.html")
		if err != nil {
			return c.SendString("<h1>Error " + strconv.FormatUint(uint64(status), 10) + "</h1><h2>" + msg + "</h2>")
		}
	}

	if _, err := os.Stat(path); err != nil {
		return c.SendString("<h1>Error " + strconv.FormatUint(uint64(status), 10) + "</h1><h2>" + msg + "</h2>")
	}

	if dynamic {
		buf, err := os.ReadFile(path)
		if err != nil {
			return c.SendString("<h1>Error " + strconv.FormatUint(uint64(status), 10) + "</h1><h2>" + msg + "</h2>")
		}

		app.compiler.compileDynamicPage(&buf, Map{
			"error": strconv.FormatUint(uint64(status), 10),
			"msg":   msg,
		})

		return c.Send(buf)
	}

	return c.SendFile(path)
}
