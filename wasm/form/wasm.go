//go:build js && wasm
// +build js,wasm

package form

import (
	"fmt"
	"syscall/js"
)

func main() {
	global := js.Global()
	_ = global

	fmt.Println("Hello, WASM!")
}
