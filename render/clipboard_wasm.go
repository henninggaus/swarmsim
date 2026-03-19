//go:build js

package render

import "syscall/js"

// ClipboardWrite copies text to the system clipboard via the JS Clipboard API.
func ClipboardWrite(text string) {
	nav := js.Global().Get("navigator")
	if nav.IsUndefined() {
		return
	}
	clip := nav.Get("clipboard")
	if clip.IsUndefined() {
		return
	}
	clip.Call("writeText", text)
}

// ClipboardRead reads text from the system clipboard.
// Because the JS Clipboard API is async, we use a callback approach.
func ClipboardRead(callback func(string)) {
	nav := js.Global().Get("navigator")
	if nav.IsUndefined() {
		callback("")
		return
	}
	clip := nav.Get("clipboard")
	if clip.IsUndefined() {
		callback("")
		return
	}
	promise := clip.Call("readText")
	if promise.IsUndefined() {
		callback("")
		return
	}
	var thenFn, catchFn js.Func
	thenFn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer thenFn.Release()
		defer catchFn.Release()
		if len(args) > 0 {
			callback(args[0].String())
		}
		return nil
	})
	catchFn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer thenFn.Release()
		defer catchFn.Release()
		callback("")
		return nil
	})
	promise.Call("then", thenFn).Call("catch", catchFn)
}
