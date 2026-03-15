//go:build js

package render

import "syscall/js"

// IsTutorialDone checks localStorage for tutorial completion.
func IsTutorialDone() bool {
	val := js.Global().Get("localStorage").Call("getItem", "swarmsim_tutorial_done")
	return !val.IsNull() && val.String() == "done"
}

// MarkTutorialDone saves tutorial completion to localStorage.
func MarkTutorialDone() {
	js.Global().Get("localStorage").Call("setItem", "swarmsim_tutorial_done", "done")
}
