//go:build !launcher

package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	_ "embed"

	"github.com/retrixe/imprint/app"
	"github.com/sqweek/dialog"
	webview "github.com/webview/webview_go"
)

// FIXME: Design UI (with live warnings/errors).
// FIXME: Validate written image.
// TODO: Future support for flashing to an internal drive?

const version = "1.0.0-alpha.2"

var w webview.WebView

//go:embed renderer/index.html
var html string
var overrideUrl = ""

//go:embed dist/index.css
var css string

//go:embed dist/index.js
var js string

// ParseToJsString takes a string, escapes slashes and double-quotes, adds newlines for multi-line
// strings and wraps it in double-quotes, allowing it to be passed to JavaScript.
func ParseToJsString(s string) string {
	return strings.ReplaceAll(
		`"`+strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`)+`"`,
		"\n", `\n`)
}

func main() {
	if len(os.Args) >= 2 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		println("imprint version v" + version)
		return
	} else if len(os.Args) >= 2 && os.Args[1] == "flash" {
		log.SetFlags(0)
		log.SetOutput(os.Stderr)
		log.SetPrefix("[flash] ")
		if len(os.Args) < 4 {
			println("Invalid usage: imprint flash <file> <destination> (--use-system-dd)")
			os.Exit(1)
		}
		if err := app.UnmountDevice(os.Args[3]); err != nil {
			log.Println(err)
			if !strings.HasSuffix(os.Args[3], "debug.iso") {
				os.Exit(1)
			}
		}
		if len(os.Args) > 4 && os.Args[4] == "--use-system-dd" {
			app.RunDd(os.Args[2], os.Args[3])
		} else {
			app.FlashFileToBlockDevice(os.Args[2], os.Args[3])
		}
		return
	}
	debug := false
	if val, exists := os.LookupEnv("DEBUG"); exists {
		debug = val == "true"
	}
	w = webview.New(debug)
	defer w.Destroy()
	w.SetSize(640, 320, webview.HintNone)
	w.SetTitle("Imprint " + version)

	// Bind a function to inject JavaScript and CSS via webview.Eval.
	w.Bind("initiate", func() {
		w.Eval(`document.getElementById('inject-css').textContent = ` + ParseToJsString(css))
		w.Eval(js)
	})

	// Bind a function to request refresh of devices attached.
	w.Bind("refreshDevices", func() {
		devices, err := app.GetDevices()
		if err != nil {
			w.Eval("setDialogReact(" + ParseToJsString("Error: "+err.Error()) + ")")
			return
		}
		if os.Getenv("DEBUG") == "true" {
			homedir, err := os.UserHomeDir()
			if err == nil {
				devices = append(devices, app.Device{
					Name:  filepath.Join(homedir, "debug.iso"),
					Model: "Write to debug ISO in home dir",
					Bytes: 10000000000,
					Size:  "10 TB",
				})
			}
		}
		jsonifiedDevices := make([]string, len(devices))
		for index, device := range devices {
			base := strconv.Itoa(device.Bytes) + " " + device.Name
			if device.Model == "" {
				jsonifiedDevices[index] = ParseToJsString(base + " (" + device.Size + ")")
			} else {
				jsonifiedDevices[index] = ParseToJsString(base + " (" + device.Model + ", " + device.Size + ")")
			}
		}
		// Call setDevicesReact.
		w.Eval("setDevicesReact([" + strings.Join(jsonifiedDevices, ", ") + "])")
	})

	// Bind a function to prompt for file.
	w.Bind("promptForFile", func() {
		homedir, err := os.UserHomeDir()
		if err != nil {
			homedir = "/"
		}
		filename, err := dialog.File().Title("Select image to flash").SetStartDir(homedir).Filter("Disk image file", "raw", "iso", "img", "dmg").Load()
		if err != nil && err.Error() != "Cancelled" {
			w.Eval("setDialogReact(" + ParseToJsString("Error: "+err.Error()) + ")")
			return
		} else if err == nil {
			stat, err := os.Stat(filename)
			if err != nil {
				w.Eval("setDialogReact(" + ParseToJsString("Error: "+err.Error()) + ")")
			} else if !stat.Mode().IsRegular() {
				w.Eval("setDialogReact(" + ParseToJsString("Error: Select a regular file!") + ")")
			} else { // Send this back to React.
				w.Eval("setFileSizeReact(" + strconv.Itoa(int(stat.Size())) + ")")
				w.Eval("setFileReact(" + ParseToJsString(filename) + ")")
			}
		}
	})

	// Bind flashing.
	var inputPipe io.WriteCloser
	var cancelled bool = false
	var mutex sync.Mutex
	w.Bind("flash", func(file string, selectedDevice string) {
		cancelled = false
		stat, err := os.Stat(file)
		if err != nil {
			w.Eval("setDialogReact(" + ParseToJsString("Error: "+err.Error()) + ")")
			return
		} else if !stat.Mode().IsRegular() {
			w.Eval("setDialogReact(" + ParseToJsString("Error: Select a regular file!") + ")")
			return
		} else {
			w.Eval("setFileSizeReact(" + strconv.Itoa(int(stat.Size())) + ")")
		}
		channel, stdin, err := app.CopyConvert(file, selectedDevice)
		inputPipe = stdin
		if err != nil {
			w.Eval("setDialogReact(" + ParseToJsString("Error: "+err.Error()) + ")")
			return
		} else {
			w.Eval("setSpeedReact(" + ParseToJsString("0 MB/s") + ")") // Show progress instantly.
			w.Eval("setProgressReact(0)")
		}
		go (func() {
			errored := false
			for {
				progress, ok := <-channel
				mutex.Lock()
				if cancelled {
					defer mutex.Unlock()
					return
				}
				mutex.Unlock()
				if ok {
					w.Dispatch(func() {
						if progress.Error != nil { // Error is always the last emitted.
							errored = true
							w.Eval("setDialogReact(" + ParseToJsString("Error: "+progress.Error.Error()) + ")")
						} else {
							w.Eval("setSpeedReact(" + ParseToJsString(progress.Speed) + ")")
							w.Eval("setProgressReact(" + strconv.Itoa(progress.Bytes) + ")")
						}
					})
				} else {
					break
				}
			}
			if !errored {
				w.Dispatch(func() { w.Eval("setProgressReact(\"Done!\")") })
			}
		})()
	})

	w.Bind("cancelFlash", func() {
		_, err := inputPipe.Write([]byte("stop\n"))
		if err != nil {
			w.Dispatch(func() { w.Eval("setProgressReact(\"Error occurred when cancelling.\")") })
		} else {
			mutex.Lock()
			defer mutex.Unlock()
			cancelled = true
			w.Dispatch(func() { w.Eval("setProgressReact(\"Cancelled the operation!\")") })
		}
	})

	if overrideUrl != "" {
		w.Navigate(overrideUrl)
	} else {
		w.Navigate("data:text/html," + strings.ReplaceAll(html,
			"<script type=\"module\" src=\"./index.tsx\" />", "<script>initiate()</script>"))
	}
	w.Run()
}
