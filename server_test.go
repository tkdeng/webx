package webx

import (
	"testing"

	"github.com/tkdeng/goutil"
)

func Test(t *testing.T) {
	// DebugCompiler = true

	if !DebugCompiler {
		fw := goutil.FileWatcher()
		fw.OnFileChange = func(path, op string) {
			goutil.CopyFile("./templates/core.js", "./test/assets/core.js")
			goutil.CopyFile("./templates/core.css", "./test/assets/core.css")
		}
		fw.WatchDir("./templates")
	}

	app, err := New("./test")
	if err != nil {
		t.Error(err)
	}

	app.Listen()
}
