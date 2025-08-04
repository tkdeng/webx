package webx

import (
	"testing"
)

func Test(t *testing.T) {
	modDevelopmentMode = true

	// DebugCompiler = true

	app, err := New("./test")
	if err != nil {
		t.Error(err)
	}

	app.NewForm("/login", func(c FormCtx) error {
		return c.Render("@form")
	})

	app.Listen()
}
