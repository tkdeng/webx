# WebX

A Non-Framework written in Go.

What is a non-framework?
It's a minimal framework that does not rely on itself as a core dependency.
This framework generates static HTML from an easy to use template engine.

This project is useful for building a new prototype when you have not decided on a big framework for the project yet. Because it uses vanilla HTML, it becomes easier to migrate your project to a bigger framework later on when your ready.

It can also be useful for anyone who simply doesn't like big frameworks, or simply has a smaller project.

Under the hood, the optional server uses gofiber, with automatically generated ssl certificates.
The framework can also support dynamic page templates with embedded variables.
You can even enable a Content Security Policy with the `csp.yml` file, and enable or disable it for each page or globally by default. Script nonce keys will automatically be randomly generated and embedded into your script tags within the template at compile time (before any variables are embedded).

This freamwork also optionally compiles markdown files.

## Installation

```shell
go get github.com/tkdeng/webx
```

## Usage

```go

import (
  "github.com/tkdeng/webx"
)

func main(){
  app, err := webx.New("./app")
  
  // note: pages will automatically be rendered
  app.Use("/api", func(c fiber.Ctx) error {
    // use app.Render to render template pages
    // note: using `@` for dynamic pages matches the file name within the pages directory
    return app.Render(c, "@api", Map{
      "myvar": "example value",
    })
  })

  app.Use("/error", func(c fiber.Ctx) error {
    // use app.Error to render error pages
    // this will default to `@error.html`
    // but will first try `@404.html` or whatever error status is being used
    return app.Error(c, 404, "Page Not Found")
  })

  app.Listen()
}

```

- Page Head: `head.html` || `head.md` (embedded into the \<head> of the document)
- Page Body: `body.html` || `body.md` (embedded into the \<body> of the document)
- Child Pages: `#page.html` || `#page.md` (used as the default for child pages, without modifying the current directory of pages)
- Dynamic Pages: `@api.html` || `@api.md` (will not render by default, but can be called by your apis)
- Content Security Policy: `csp.yml` (only available in root of pages directory)

```html

<!-- embed html or md file -->
{@header}

<div class="widget">
  <!-- dynamic pages can also be statically embedded -->
  {@api}
</div>

<!-- {variables} will escape html by default -->
<div class="{myclass}">
  {myvar}
</div>

<!-- use the '#' prefix to allow html in variables -->
{#htmlvar}

```

## Just Using The Compiler

```go

import (
  "github.com/tkdeng/webx"
)

func main(){
  webx.Compile("./app")
}

```

In the `dist` directory, their are different types of files generated.

- Static Files: `index.html`, `about.html`, `about/more.html` can be rendered equaivalent to the url. Note sometimes these static files will also be compressed with gzip (`index.html.gz`, `about.html.gz`).
- Dynamic Files: `@api.html`, `about/@widget.html` can be rendered dyncmically and have variables populated.
- Static Dynamic Files: `#index.html`, `#login.html` are just like static files, but they have basic variables like {nonce} keys prepared for a CSP to populate. Most variables have already been statically compiled.
