package webx

import (
	"bytes"
	"embed"
	_ "embed"
	"os"
	"path/filepath"
	"strings"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tkdeng/goutil"
	"github.com/tkdeng/regex"
)

//go:embed templates/layout.html
var tempLayout []byte

//go:embed templates/core.js
var tempScript []byte

//go:embed templates/core.css
var tempStyle []byte

//go:embed templates/example/config.yml
var tempExConfig []byte

//go:embed templates/example/csp.yml
var tempExCSP []byte

//go:embed templates/example/config.css
var tempExCssConfig []byte

//go:embed templates/example/*
var tempExample embed.FS

//go:embed templates/assets/*
var tempAssets embed.FS

func addTemplateExample(file string, out string) {
	if buf, err := tempExample.ReadFile("templates/example/" + file); err == nil {
		os.WriteFile(out, buf, 0755)
	}
}

func compileTemplates(appConfig *Config, initExample bool) {
	coreScript := goutil.CloneBytes(tempScript)
	coreStyle := goutil.CloneBytes(tempStyle)

	if !appConfig.DebugMode {
		// minify tempScript
		{
			comment := []byte{}
			coreScript = regex.Comp(`(?s)^(/?/\*![^\r\n]*?\*/)\r?\n?`).RepFunc(coreScript, func(data func(int) []byte) []byte {
				comment = data(1)
				return []byte{}
			})

			m := minify.New()
			m.Add("text/javascript", &js.Minifier{})

			var b bytes.Buffer
			if err := m.Minify("text/javascript", &b, bytes.NewBuffer(coreScript)); err == nil {
				coreScript = regex.JoinBytes(
					comment, '\n',
					';', b.Bytes(), ';',
				)
			}

			for _, plugin := range plugins {
				for name, buf := range plugin.assets {
					if strings.HasSuffix(name, ".js") {
						var b bytes.Buffer
						if err := m.Minify("text/javascript", &b, bytes.NewBuffer(buf)); err == nil {
							plugin.assets[name] = regex.JoinBytes(
								';', b.Bytes(), ';',
							)
						}
					}
				}
			}
		}

		// minify tempStyle
		{
			comment := []byte{}
			coreStyle = regex.Comp(`(?s)^(/?/\*![^\r\n]*?\*/)\r?\n?`).RepFunc(coreStyle, func(data func(int) []byte) []byte {
				comment = data(1)
				return []byte{}
			})

			m := minify.New()
			m.Add("text/css", &css.Minifier{})

			var b bytes.Buffer
			if err := m.Minify("text/css", &b, bytes.NewBuffer(coreStyle)); err == nil {
				coreStyle = regex.JoinBytes(
					comment, '\n',
					b.Bytes(),
				)
			}

			for _, plugin := range plugins {
				for name, buf := range plugin.assets {
					if strings.HasSuffix(name, ".css") {
						var b bytes.Buffer
						if err := m.Minify("text/css", &b, bytes.NewBuffer(buf)); err == nil {
							plugin.assets[name] = b.Bytes()
						}
					}
				}
			}
		}
	}

	os.WriteFile(appConfig.Root+"/plugins/assets/core.js", coreScript, 0755)
	os.WriteFile(appConfig.Root+"/plugins/assets/core.css", coreStyle, 0755)

	if initExample {
		addTemplateExample("config.yml", appConfig.Root+"/config.yml")
		addTemplateExample("csp.yml", appConfig.Root+"/pages/csp.yml")
		addTemplateExample("config.css", appConfig.Root+"/theme/config.css")

		addTemplateExample("head.html", appConfig.Root+"/pages/head.html")
		addTemplateExample("header.html", appConfig.Root+"/pages/header.html")
		addTemplateExample("body.md", appConfig.Root+"/pages/body.md")

		os.MkdirAll(appConfig.Root+"/pages/about", 0755)
		addTemplateExample("about.md", appConfig.Root+"/pages/about/body.md")
		addTemplateExample("@widget.html", appConfig.Root+"/pages/@widget.html")
		addTemplateExample("@error.html", appConfig.Root+"/pages/@error.html")
	}

	if assets, err := tempAssets.ReadDir("templates/assets"); err == nil {
		for _, asset := range assets {
			path := filepath.Join("templates/assets", asset.Name())
			if strings.HasSuffix(asset.Name(), ".html") || strings.HasSuffix(asset.Name(), ".md") {
				if out, err := goutil.JoinPath(appConfig.Root, "pages", asset.Name()); err == nil {
					if !modDevelopmentMode {
						if _, err := os.Stat(out); err == nil {
							continue
						}
					}

					if buf, err := tempAssets.ReadFile(path); err == nil {
						os.WriteFile(out, buf, 0755)
					}
				}
			} else if strings.HasSuffix(asset.Name(), ".js") || strings.HasSuffix(asset.Name(), ".css") {
				if out, err := goutil.JoinPath(appConfig.Root, "plugins/assets", asset.Name()); err == nil {
					if buf, err := tempAssets.ReadFile(path); err == nil {
						if !appConfig.DebugMode {
							if strings.HasSuffix(asset.Name(), ".js") {
								minifyJS(&buf, asset.Name()[:len(asset.Name())-3])
							} else if strings.HasSuffix(asset.Name(), ".css") {
								minifyCSS(&buf, asset.Name()[:len(asset.Name())-4])
							}
						}

						os.WriteFile(out, buf, 0755)
					}
				}
			}
		}
	}

	// live updates
	if modDevelopmentMode {
		tempRoot, err := filepath.Abs("templates/assets")
		if err != nil {
			return
		}

		fw := goutil.FileWatcher()

		fw.OnFileChange = func(path, op string) {
			name, err := filepath.Rel(tempRoot, path)
			if err != nil {
				return
			}

			if strings.HasSuffix(name, ".html") || strings.HasSuffix(name, ".md") {
				if out, err := goutil.JoinPath(appConfig.Root, "pages", name); err == nil {
					if buf, err := os.ReadFile(path); err == nil {
						os.WriteFile(out, buf, 0755)
					}
				}
			} else if strings.HasSuffix(name, ".js") || strings.HasSuffix(name, ".css") {
				if out, err := goutil.JoinPath(appConfig.Root, "plugins/assets", name); err == nil {
					if buf, err := os.ReadFile(path); err == nil {
						os.WriteFile(out, buf, 0755)
					}
				}
			}
		}

		fw.WatchDir("templates/assets")
	}
}

func minifyJS(script *[]byte, name string) {
	m := minify.New()
	m.Add("text/javascript", &js.Minifier{})

	var b bytes.Buffer
	if err := m.Minify("text/javascript", &b, bytes.NewBuffer(*script)); err == nil {
		*script = regex.JoinBytes(
			`//`, `*! `, name, ` */`, '\n',
			';', b.Bytes(), ';',
		)
	}
}

func minifyCSS(style *[]byte, name string) {
	m := minify.New()
	m.Add("text/css", &css.Minifier{})

	var b bytes.Buffer
	if err := m.Minify("text/css", &b, bytes.NewBuffer(*style)); err == nil {
		*style = regex.JoinBytes(
			`/*! `, name, ` */`, '\n',
			b.Bytes(),
		)
	}
}
