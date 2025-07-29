package webx

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	bash "github.com/tkdeng/gobash"
	"github.com/tkdeng/goutil"

	_ "embed"
)

//todo: see if embedding directories in go is possible

//go:embed wasm_dist/*
var wasmCoreFiles embed.FS

func (comp *compiler) compWASM() {
	files, err := os.ReadDir(comp.config.Root + "/wasm")
	if err != nil {
		return
	}

	for _, file := range files {
		if file.IsDir() {
			if wasmPath, err := goutil.JoinPath(comp.config.Root, "wasm", file.Name()); err == nil {
				if outPath, err := goutil.JoinPath(comp.config.Root, "plugins/assets", file.Name()+".wasm"); err == nil {
					_, err := bash.Run([]string{"go", "build", "-o", outPath, file.Name()}, wasmPath, []string{"GOOS=js", "GOARCH=wasm"})
					if err != nil {
						fmt.Println(err)
					}
				}
			}
		}
	}

	comp.compCoreWASM()
}

func (comp *compiler) compCoreWASM() {
	files, err := wasmCoreFiles.ReadDir("wasm_dist")
	if err != nil {
		return
	}

	for _, file := range files {
		name := file.Name()
		if !file.IsDir() && strings.HasSuffix(name, ".wasm") {
			if buf, err := wasmCoreFiles.ReadFile("wasm_dist/" + name); err == nil {
				if assetPath, err := goutil.JoinPath(comp.config.Root, "plugins/assets", file.Name()); err == nil {
					os.WriteFile(assetPath, buf, 0755)
				}
			}
		}
	}

	// compile core wasm in development mode
	if !modDevelopmentMode {
		return
	}

	os.RemoveAll("wasm_dist")
	os.MkdirAll("wasm_dist", 0755)
	os.WriteFile("wasm_dist/_", []byte{}, 0755)

	files, err = os.ReadDir("wasm")
	if err != nil {
		return
	}

	for _, file := range files {
		if file.IsDir() {
			if wasmPath, err := goutil.JoinPath("wasm", file.Name()); err == nil {
				if outPath, err := goutil.JoinPath("wasm_dist", file.Name()+".wasm"); err == nil {
					_, err := bash.Run([]string{"go", "build", "-o", outPath, file.Name()}, wasmPath, []string{"GOOS=js", "GOARCH=wasm"})
					if err != nil {
						fmt.Println(err)
					}

					if buf, err := os.ReadFile(outPath); err == nil {
						if assetPath, err := goutil.JoinPath(comp.config.Root, "plugins/assets", file.Name()+".wasm"); err == nil {
							os.WriteFile(assetPath, buf, 0755)
						}
					}
				}
			}
		}
	}

	// live file watcher
	if wasmRoot, err := filepath.Abs("wasm"); err == nil {
		fw := goutil.FileWatcher()

		fw.OnFileChange = func(path, op string) {
			if relPath, err := filepath.Rel(wasmRoot, path); err == nil {
				name := strings.SplitN(relPath, "/", 2)[0]

				if wasmPath, err := goutil.JoinPath(wasmRoot, name); err == nil {
					if outPath, err := goutil.JoinPath("wasm_dist", name+".wasm"); err == nil {
						_, err := bash.Run([]string{"go", "build", "-o", outPath, name}, wasmPath, []string{"GOOS=js", "GOARCH=wasm"})
						if err != nil {
							fmt.Println(err)
						}

						if buf, err := os.ReadFile(outPath); err == nil {
							if assetPath, err := goutil.JoinPath(comp.config.Root, "plugins/assets", name+".wasm"); err == nil {
								os.WriteFile(assetPath, buf, 0755)
							}
						}
					}
				}
			}
		}

		fw.OnRemove = func(path, op string) (removeWatcher bool) {
			if relPath, err := filepath.Rel(wasmRoot, path); err == nil {
				name := strings.SplitN(relPath, "/", 2)[0]
				fmt.Println(name)

				if outPath, err := goutil.JoinPath("wasm_dist", name+".wasm"); err == nil {
					os.Remove(outPath)

					if assetPath, err := goutil.JoinPath(comp.config.Root, "plugins/assets", name+".wasm"); err == nil {
						os.Remove(assetPath)
					}
				}
			}

			return true
		}

		fw.WatchDir("wasm")
	}
}
