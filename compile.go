package webx

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	lorem "github.com/drhodes/golorem"
	"github.com/gomarkdown/markdown"
	mdhtml "github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tkdeng/regex"
	"github.com/tkdeng/goutil"
	"gopkg.in/yaml.v3"
)

var DebugCompiler = false

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

//go:embed templates/example/head.html
var tempExHead []byte

//go:embed templates/example/header.html
var tempExHeader []byte

//go:embed templates/example/body.md
var tempExBody []byte

//go:embed templates/example/about.md
var tempExAbout []byte

//go:embed templates/example/@widget.html
var tempExWidget []byte

//go:embed templates/example/@error.html
var tempExError []byte

type compiler struct {
	config *Config
}

func compile(appConfig *Config) *compiler {
	initExample := false
	if _, err := os.Stat(appConfig.Root); err != nil {
		initExample = true
	}

	os.MkdirAll(appConfig.Root, 0755)
	os.MkdirAll(appConfig.Root+"/pages", 0755)
	os.MkdirAll(appConfig.Root+"/theme", 0755)
	os.MkdirAll(appConfig.Root+"/assets", 0755)
	os.MkdirAll(appConfig.Root+"/db", 0755)

	if appConfig.PublicURI != "" {
		os.MkdirAll(appConfig.Root+"/public", 0755)
	}

	//todo: sandbox download directory
	// os.MkdirAll(appConfig.Root+"/download", 2600)

	PrintMsg("warn", "Compiling Server Pages...", 50, false)

	os.RemoveAll(appConfig.Root + "/dist")
	if err := os.Mkdir(appConfig.Root+"/dist", 0755); err != nil {
		panic(err)
	}

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
		}
	}

	os.WriteFile(appConfig.Root+"/assets/core.js", coreScript, 0755)
	os.WriteFile(appConfig.Root+"/assets/core.css", coreStyle, 0755)

	if initExample {
		os.WriteFile(appConfig.Root+"/config.yml", tempExConfig, 0755)
		os.WriteFile(appConfig.Root+"/pages/csp.yml", tempExCSP, 0755)
		os.WriteFile(appConfig.Root+"/pages/head.html", tempExHead, 0755)
		os.WriteFile(appConfig.Root+"/pages/header.html", tempExHeader, 0755)
		os.WriteFile(appConfig.Root+"/pages/body.md", tempExBody, 0755)
		os.MkdirAll(appConfig.Root+"/pages/about", 0755)
		os.WriteFile(appConfig.Root+"/pages/about/body.md", tempExAbout, 0755)
		os.WriteFile(appConfig.Root+"/pages/@widget.html", tempExWidget, 0755)
		os.WriteFile(appConfig.Root+"/pages/@error.html", tempExError, 0755)
	}

	comp := compiler{
		config: appConfig,
	}

	comp.loadCSP()

	comp.compPages()
	comp.compileLive()

	//todo: generate manifest.json (and auto generate icons) and allow config.yml file to modify

	PrintMsg("confirm", "Compiled Server!", 50, true)

	return &comp
}

func (comp *compiler) compileLive() {
	fw := goutil.FileWatcher()

	fw.OnFileChange = func(path, op string) {
		path, err := filepath.Rel(comp.config.Root+"/pages", path)
		if err != nil {
			return
		}

		if path == "csp.yml" {
			comp.loadCSP()
			comp.compPages()
			return
		}

		if strings.HasSuffix(path, ".html") || strings.HasSuffix(path, ".md") {
			path = filepath.Dir(path)

			if path == "." || path == "" {
				comp.compPages()
				return
			}

			comp.compPages(path)
			return
		}
	}

	fw.OnDirAdd = func(path, op string) bool {
		path, err := filepath.Rel(comp.config.Root+"/pages", path)
		if err != nil {
			return true
		}

		comp.compPages(path)
		return true
	}

	fw.OnRemove = func(path, op string) bool {
		path, err := filepath.Rel(comp.config.Root+"/pages", path)
		if err != nil {
			return true
		}

		if path == "csp.yml" {
			comp.config.csp = CSP{}
			comp.config.cspText = ""
			comp.compPages()
			return true
		}

		if strings.HasSuffix(path, ".html") || strings.HasSuffix(path, ".md") {
			path = filepath.Dir(path)

			if path == "." || path == "" {
				comp.compPages()
				return true
			}

			comp.compPages(path)
			return true
		}

		if dist, err := goutil.JoinPath(comp.config.Root+"/dist", path); err == nil {
			os.Remove(dist + ".html")
			os.RemoveAll(dist)
		}

		return true
	}

	fw.WatchDir(comp.config.Root + "/pages")
}

func (comp *compiler) compPages(path ...string) {
	dir, err := goutil.JoinPath(comp.config.Root+"/pages", path...)
	if err != nil {
		return
	}

	dist, err := goutil.JoinPath(comp.config.Root+"/dist", path...)
	if err != nil {
		return
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	dynPage := []string{}

	var wg sync.WaitGroup
	for _, file := range files {
		if file.IsDir() {
			wg.Add(1)
			go func() {
				defer wg.Done()
				comp.compPages(append(path, file.Name())...)
			}()
		} else if strings.HasPrefix(file.Name(), "@") {
			dynPage = append(dynPage, file.Name())
		}
	}
	defer wg.Wait()

	for _, page := range dynPage {
		comp.precompDynamicPage(dir, dist, page, path)
	}

	buf := goutil.CloneBytes(tempLayout)
	configVars := comp.compPage(&buf, path)
	comp.compVars(&buf, path, false, configVars)

	if len(path) == 0 || dist == comp.config.Root+"/dist" {
		dist += "/index"
	}
	dist += ".html"

	// check if CSP is enabled
	if comp.config.cspText != "" && ((comp.config.CSP && configVars["csp"] != "no" && configVars["csp"] != "false") || configVars["csp"] == "yes" || configVars["csp"] == "true") {
		if regex.Comp(`'nonce(-.*?|)'`).Match([]byte(comp.config.csp.ScriptSrc)) {
			buf = regex.Comp(`<script(\s.*?|)>`).RepFunc(buf, func(data func(int) []byte) []byte {
				return regex.JoinBytes(`<script`, data(1), ` nonce="{nonce}"`, '>')
			})
		}

		if regex.Comp(`'nonce(-.*?|)'`).Match([]byte(comp.config.csp.StyleSrc)) {
			buf = regex.Comp(`<style(\s.*?|)>`).RepFunc(buf, func(data func(int) []byte) []byte {
				return regex.JoinBytes(`<style`, data(1), ` nonce="{nonce}"`, '>')
			})
		}

		cDist := string(regex.Comp(`\/([^\/]+)$`).Rep([]byte(dist), []byte("/#$1")))
		os.Remove(dist)
		os.Remove(dist + ".gz")

		os.MkdirAll(filepath.Dir(cDist), 0755)
		os.WriteFile(cDist, buf, 0755)
		return
	}

	// compress if not debug mode
	if !comp.config.DebugMode {
		var b bytes.Buffer
		if gz, err := gzip.NewWriterLevel(&b, 6); err == nil {
			defer gz.Close()

			if _, err = gz.Write(buf); err == nil {
				if err = gz.Close(); err == nil {
					buf = b.Bytes()
					dist += ".gz"
				}
			}

			gz.Flush()
			gz.Close()
		}
	}

	cDist := string(regex.Comp(`\/([^\/]+)$`).Rep([]byte(dist), []byte("/#$1")))
	os.Remove(cDist)

	os.MkdirAll(filepath.Dir(dist), 0755)
	os.WriteFile(dist, buf, 0755)
}

func (comp *compiler) compPage(buf *[]byte, uriPath []string) Map {
	*buf = bytes.TrimSpace(*buf)
	*buf = goutil.CloneBytes(*buf)

	config := Map{}

	readFile := func(path string, uri []string, name string) ([]byte, error) {
		var b []byte
		var err error = errors.New("file not found")

		isMD := false

		// embed parent #page.html files
		if strings.Join(uri, "/") != strings.Join(uriPath, "/") {
			cPath := string(regex.Comp(`\/([^\/]+)$`).Rep([]byte(path), []byte("/#$1")))

			if err != nil {
				isMD = false
				b, err = os.ReadFile(cPath + ".html")
			}

			if err != nil {
				isMD = true
				b, err = os.ReadFile(cPath + ".md")
			}
		}

		// embed regular page files
		if err != nil {
			isMD = false
			b, err = os.ReadFile(path + ".html")
		}

		if err != nil {
			isMD = true
			b, err = os.ReadFile(path + ".md")
		}

		// embed @widgets
		if err != nil {
			dPath := string(regex.Comp(`\/([^\/]+)$`).Rep([]byte(path), []byte("/@$1")))

			hasDyn := false
			if _, e := os.Stat(dPath + ".html"); e == nil {
				hasDyn = true
				isMD = false
				dPath += ".html"
			} else if _, e := os.Stat(dPath + ".md"); e == nil {
				hasDyn = true
				isMD = true
				dPath += ".md"
			}

			if hasDyn {
				b, err = os.ReadFile(dPath)
				if err == nil {
					if isMD {
						comp.compileMD(&b)
					}

					configVars := comp.compPage(&b, uriPath)
					comp.compVars(&b, uriPath, false, configVars)
					return b, nil
				}
				// return []byte(`<param name="load-widget" value="/` + strings.Join(append(uri, "@"+name), "/") + `"/>`), nil
				return []byte{}, err
			}
		}

		if err != nil {
			return []byte{}, err
		}

		// get config from file
		b = regex.Comp(`(?s)^---\r?\n(.*?)\r?\n---\r?\n`).RepFunc(b, func(data func(int) []byte) []byte {
			b := regex.Comp(`(?m)^(\s*(?:-\s+|))([\w_\-]+):`).RepFunc(data(1), func(data func(int) []byte) []byte {
				return regex.JoinBytes(data(1), bytes.ReplaceAll(bytes.ReplaceAll(bytes.ToLower(data(2)), []byte{'-'}, []byte{}), []byte{'_'}, []byte{}), ':')
			})
			yaml.Unmarshal(b, &config)

			return []byte{}
		})

		if isMD {
			comp.compileMD(&b)
		}

		comp.compPage(&b, uriPath)
		return b, nil
	}

	*buf = regex.Comp(`\{@([\w_\-\.]+)\}`).RepFunc(*buf, func(data func(int) []byte) []byte {
		uri := uriPath
		for len(uri) != 0 {
			if dir, err := goutil.JoinPath(comp.config.Root+"/pages", uri...); err == nil {
				if path, err := goutil.JoinPath(dir, string(data(1))); err == nil {
					if b, err := readFile(path, uri, string(data(1))); err == nil {
						return b
					}

					uri = uri[:len(uri)-1]
				}
			}
		}

		if path, err := goutil.JoinPath(comp.config.Root+"/pages", string(data(1))); err == nil {
			if b, err := readFile(path, uri, string(data(1))); err == nil {
				return b
			}
		}

		return []byte{}
	})

	comp.compileHTML(buf)

	return config
}

func (comp *compiler) compVars(buf *[]byte, uriPath []string, dynamic bool, configVars Map) {
	if !dynamic {
		name := ""
		if len(uriPath) > 0 {
			name = capWords(uriPath[len(uriPath)-1])
		}
		comp.compTitleVars(buf, name, configVars)
	}

	*buf = regex.Comp(`\{#?uri\}`).RepLit(*buf, EscapeHTML([]byte(strings.Join(uriPath, "/"))))

	if len(uriPath) > 1 {
		*buf = regex.Comp(`\{#?parent\}`).RepLit(*buf, EscapeHTML([]byte(uriPath[len(uriPath)-2])))
		*buf = regex.Comp(`\{#?Parent\}`).RepLit(*buf, EscapeHTML([]byte(capWords(uriPath[len(uriPath)-2]))))
		*buf = regex.Comp(`\{#?PARENT\}`).RepLit(*buf, EscapeHTML([]byte(strings.ToUpper(uriPath[len(uriPath)-2]))))
	}

	if len(uriPath) > 0 {
		*buf = regex.Comp(`\{#?(page|parent)\}`).RepLit(*buf, EscapeHTML([]byte(uriPath[len(uriPath)-1])))
		*buf = regex.Comp(`\{#?(Page|Parent)\}`).RepLit(*buf, EscapeHTML([]byte(capWords(uriPath[len(uriPath)-1]))))
		*buf = regex.Comp(`\{#?(PAGE|PARENT)\}`).RepLit(*buf, EscapeHTML([]byte(strings.ToUpper(uriPath[len(uriPath)-1]))))

		*buf = regex.Comp(`\{#?root\}`).RepLit(*buf, EscapeHTML([]byte(uriPath[0])))
		*buf = regex.Comp(`\{#?Root\}`).RepLit(*buf, EscapeHTML([]byte(capWords(uriPath[0]))))
		*buf = regex.Comp(`\{#?ROOT\}`).RepLit(*buf, EscapeHTML([]byte(strings.ToUpper(uriPath[0]))))
	} else {
		*buf = regex.Comp(`\{#?(page|parent|root)\}`).RepLit(*buf, EscapeHTML([]byte(comp.config.AppTitle)))
		*buf = regex.Comp(`\{#?(Page|Parent|Root)\}`).RepLit(*buf, EscapeHTML([]byte(capWords(comp.config.AppTitle))))
		*buf = regex.Comp(`\{#?(PAGE|PARENT|ROOT)\}`).RepLit(*buf, EscapeHTML([]byte(strings.ToUpper(comp.config.AppTitle))))
	}

	if !dynamic {
		comp.compRandVars(buf)
	}

	*buf = regex.Comp(`\{(#|)([\w_\-\.]+)\}`).RepFunc(*buf, func(data func(int) []byte) []byte {
		if val, ok := configVars[string(data(2))]; ok {
			if len(data(1)) != 0 {
				return []byte(val)
			}

			//todo: detect if inside html arg
			return EscapeHTML([]byte(val))
		} else if val, ok := comp.config.Vars[string(data(2))]; ok {
			if len(data(1)) != 0 {
				return []byte(val)
			}

			//todo: detect if inside html arg
			return EscapeHTML([]byte(val))
		}

		if dynamic {
			return data(0)
		}
		return []byte{}
	})
}

func (comp *compiler) compTitleVars(buf *[]byte, name string, configVars Map) {
	*buf = regex.Comp(`\{#?sitetitle\}`).RepLit(*buf, EscapeHTML([]byte(comp.config.Title)))

	if val, ok := configVars["title"]; ok {
		*buf = regex.Comp(`\{#?title\}`).RepLit(*buf, EscapeHTML([]byte(val)))
	} else if name != "" {
		*buf = regex.Comp(`\{#?title\}`).RepLit(*buf, EscapeHTML([]byte(name+" | "+comp.config.Title)))
	} else {
		*buf = regex.Comp(`\{#?title\}`).RepLit(*buf, EscapeHTML([]byte(comp.config.Title)))
	}

	if val, ok := configVars["app"]; ok {
		*buf = regex.Comp(`\{#?app\}`).RepLit(*buf, EscapeHTML([]byte(val)))
	} else if val, ok := configVars["apptitle"]; ok {
		*buf = regex.Comp(`\{#?app\}`).RepLit(*buf, EscapeHTML([]byte(val)))
	} else {
		*buf = regex.Comp(`\{#?app\}`).RepLit(*buf, EscapeHTML([]byte(comp.config.AppTitle)))
	}

	if val, ok := configVars["desc"]; ok {
		*buf = regex.Comp(`\{#?desc\}`).RepLit(*buf, EscapeHTML([]byte(val)))
	} else if val, ok := configVars["description"]; ok {
		*buf = regex.Comp(`\{#?desc\}`).RepLit(*buf, EscapeHTML([]byte(val)))
	} else {
		*buf = regex.Comp(`\{#?desc\}`).RepLit(*buf, EscapeHTML([]byte(comp.config.Desc)))
	}

	if val, ok := configVars["icon"]; ok {
		*buf = regex.Comp(`\{#?icon\}`).RepLit(*buf, EscapeHTML([]byte(val)))
	} else {
		*buf = regex.Comp(`\{#?icon\}`).RepLit(*buf, EscapeHTML([]byte(comp.config.Icon)))
	}
}

func (comp *compiler) compRandVars(buf *[]byte) {
	*buf = regex.Comp(`\{#?rand\s*([0-9]*)\}`).RepFunc(*buf, func(data func(int) []byte) []byte {
		size := uint(16)
		if len(data(1)) > 0 {
			if s, e := strconv.ParseUint(string(data(1)), 10, 0); e == nil && s > 0 {
				size = uint(s)
			}
		}

		return goutil.RandBytes(size)
	})

	urand := [][]byte{}
	*buf = regex.Comp(`\{#?urand\s*([0-9]*)\}`).RepFunc(*buf, func(data func(int) []byte) []byte {
		size := uint(16)
		if len(data(1)) > 0 {
			if s, e := strconv.ParseUint(string(data(1)), 10, 0); e == nil && s > 0 {
				size = uint(s)
			}
		}

		return goutil.URandBytes(size, &urand)
	})

	*buf = regex.Comp(`\{#?randint\s*([0-9]*)\}`).RepFunc(*buf, func(data func(int) []byte) []byte {
		size := 10
		if len(data(1)) > 0 {
			if s, e := strconv.Atoi(string(data(1))); e == nil && s > 0 {
				size = s
			}
		}

		return []byte(strconv.Itoa(rand.Intn(size)))
	})

	*buf = regex.Comp(`\{#?(?:lorem|rand)(?:text|)\s*([pswehu][a-z]*|)\s*([0-9]*)([^0-9][0-9]+|)\}`).RepFunc(*buf, func(data func(int) []byte) []byte {
		t := byte('p')
		min := 3
		max := 5

		if len(data(1)) != 0 {
			t = data(1)[0]
		}

		if len(data(2)) != 0 {
			if s, e := strconv.Atoi(string(data(2))); e == nil && s > 0 {
				min = s
			}
		}

		if len(data(3)) != 0 {
			if s, e := strconv.Atoi(string(data(3)[1:])); e == nil && s > 0 {
				max = s
			}
		} else if len(data(2)) != 0 {
			max = min
		}

		if min > max {
			min, max = max, min
		}

		switch t {
		case 'p':
			return []byte(lorem.Paragraph(min, max))
		case 's':
			return []byte(lorem.Sentence(min, max))
		case 'w':
			return []byte(lorem.Word(min, max))
		case 'e':
			return []byte(lorem.Email())
		case 'h':
			return []byte(lorem.Host())
		case 'u':
			return []byte(lorem.Url())
		default:
			return []byte(lorem.Paragraph(min, max))
		}
	})
}

func (comp *compiler) precompDynamicPage(dir, dist string, page string, uriPath []string) {
	path, err := goutil.JoinPath(dir, page)
	if err != nil {
		return
	}

	out, err := goutil.JoinPath(dist, string(regex.Comp(`\.(html|md)$`).RepLit([]byte(page), []byte(".html"))))
	if err != nil {
		return
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return
	}

	buf := goutil.CloneBytes(tempLayout)
	buf = regex.Comp(`\{@body\}`).Rep(buf, b)
	configVars := comp.compPage(&buf, uriPath)
	comp.compVars(&buf, uriPath, true, configVars)

	os.MkdirAll(dist, 0755)
	os.WriteFile(out, buf, 0755)
}

func (comp *compiler) compileDynamicPage(buf *[]byte, vars Map) {
	comp.compTitleVars(buf, "", vars)
	comp.compRandVars(buf)

	*buf = regex.Comp(`\{(#|)([\w_\-\.]+)\}`).RepFunc(*buf, func(data func(int) []byte) []byte {
		if val, ok := vars[string(data(2))]; ok {
			if len(data(1)) != 0 {
				return []byte(val)
			}

			//todo: detect if inside html arg
			return EscapeHTML([]byte(val))
		}

		return []byte{}
	})
}

func (comp *compiler) compileHTML(buf *[]byte) {
	//todo: add plugin support with shortcodes
	// may have plugins written in elixir, lua, or javascript

	// minify HTML
	m := minify.New()
	m.AddFunc("text/html", html.Minify)

	m.Add("text/html", &html.Minifier{
		KeepQuotes:       true,
		KeepDocumentTags: true,
		KeepEndTags:      true,
		KeepWhitespace:   comp.config.DebugMode,
	})

	var b bytes.Buffer
	if err := m.Minify("text/html", &b, bytes.NewBuffer(*buf)); err == nil {
		*buf = b.Bytes()
	}
}

func (comp *compiler) compileMD(buf *[]byte) {
	// create markdown parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(*buf)

	// create HTML renderer with extensions
	htmlFlags := mdhtml.CommonFlags | mdhtml.HrefTargetBlank
	opts := mdhtml.RendererOptions{
		Flags:           htmlFlags,
		HeadingIDPrefix: "h-",
	}
	renderer := mdhtml.NewRenderer(opts)

	*buf = markdown.Render(doc, renderer)
}

func (comp *compiler) loadCSP() {
	comp.config.csp = CSP{}
	if err := goutil.ReadConfig(comp.config.Root+"/pages/csp.yml", &comp.config.csp); err == nil {
		comp.config.cspText = string(regex.JoinBytes(
			"default-src ", comp.config.csp.DefaultSrc, ';',
			" script-src ", comp.config.csp.ScriptSrc, ';',
			" style-src ", comp.config.csp.StyleSrc, ';',
			" img-src ", comp.config.csp.ImgSrc, ';',
			" object-src ", comp.config.csp.ObjectSrc, ';',
			" font-src ", comp.config.csp.FontSrc, ';',
			" connect-src ", comp.config.csp.ConnectSrc, ';',
			" base-uri ", comp.config.csp.BaseUri, ';',
			" form-action ", comp.config.csp.FormAction, ';',
			" frame-ancestors ", comp.config.csp.FrameAncestors, ';',
			" require-trusted-types-for ", comp.config.csp.RequireTrustedTypesFor, ';',
			" report-uri ", comp.config.csp.ReportUri, ';',
		))
	}
}
