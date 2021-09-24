//go:build !launcher

package main

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	_ "embed"

	"github.com/sqweek/dialog"
	"github.com/webview/webview"
)

// TODO: Design UI (with live warnings/errors). Validate written image?
// LOW-TODO: Future support for flashing to an internal drive?

const html = `
<html lang="en">
<head>
  <meta charset="UTF-8">
  <!-- Use minimum-scale=1 to enable GPU rasterization -->
  <meta
    name='viewport'
    content='user-scalable=0, initial-scale=1, minimum-scale=1, width=device-width, height=device-height'
  />
	<style>
	body {
		margin: 0;
		font-family: -apple-system,BlinkMacSystemFont,"Segoe UI",
		  Ubuntu,Cantarell,Oxygen-Sans,"Helvetica Neue",Arial,Roboto,sans-serif;
	}
  </style>
</head>
<body><div id="app"></div><script>initiateReact()</script></body>
</html>
`

const version = "1.0.0-alpha.2"

var w webview.WebView

//go:embed dist/main.js
var js string

// var file = ""

// ParseToJsString takes a string and escapes slashes and double-quotes,
// and converts it to a string that can be passed to JavaScript.
func ParseToJsString(s string) string {
	return "\"" + strings.ReplaceAll(strings.ReplaceAll(s, "\\", "\\\\"), "\"", "\\\"") + "\""
}

// SetFile sets the value of the file variable in both Go and React.
// func SetFile(value string) {file = value;w.Eval("setFileReact(" + ParseToJsString(value) + ")")}

func main() {
	if len(os.Args) >= 2 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		println("writer version v" + version)
		return
	} else if len(os.Args) >= 2 && os.Args[1] == "dd" {
		if len(os.Args) != 4 {
			println("Invalid usage: writer dd <file> <dest>")
			os.Exit(1)
		}
		runDd()
		return
	}
	debug := false
	if val, exists := os.LookupEnv("DEBUG"); exists {
		debug = val == "true"
	}
	w = webview.New(debug)
	defer w.Destroy()
	w.SetSize(420, 210, webview.HintNone)
	w.SetTitle("Writer " + version)

	// Bind variables.
	// w.Bind("setFileGo", func(newFile string) {file = newFile})

	// Bind a function to initiate React via webview.Eval.
	w.Bind("initiateReact", func() { w.Eval(js) })

	// Bind a function to request refresh of devices attached.
	w.Bind("refreshDevices", func() {
		devices, err := GetDevices()
		if err != nil {
			w.Eval("setDialogReact(" + ParseToJsString("Error: "+err.Error()) + ")")
			return
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
		if len(jsonifiedDevices) >= 1 {
			w.Eval("setSelectedDeviceReact(" + jsonifiedDevices[0] + ")")
		}
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
	var currentDdProcess *exec.Cmd
	var cancelled bool = false
	w.Bind("flash", func(file string, selectedDevice string) {
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
		channel, dd, err := CopyConvert(file, selectedDevice)
		currentDdProcess = dd
		if err != nil {
			w.Eval("setDialogReact(" + ParseToJsString("Error: "+err.Error()) + ")")
			return
		}
		go (func() {
			errored := false
			for {
				progress, ok := <-channel
				if cancelled {
					cancelled = false
					return
				} else if ok {
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
		pipe, err := currentDdProcess.StdinPipe()
		if err != nil {
			w.Dispatch(func() { w.Eval("setProgressReact(\"Error occurred when cancelling.\")") })
		}
		pipe.Write([]byte("stop\n"))
		w.Dispatch(func() { w.Eval("setProgressReact(\"Cancelled the operation!\")") })
	})

	w.Navigate("data:text/html," + html)
	w.Run()
}

func runDd() {
	cmd := exec.Command(
		"dd", "if="+os.Args[2], "of="+os.Args[3], "status=progress", "bs=1M", "conv=fdatasync")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	go io.Copy(os.Stdout, stdout)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}
	go io.Copy(os.Stderr, stderr)
	err = cmd.Start()
	if err != nil {
		panic(err)
	}
	quit := make(chan bool, 1)
	go (func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			select {
			case <-quit:
				return
			default:
				text, err := reader.ReadString('\n')
				if strings.TrimSpace(text) == "stop" {
					cmd.Process.Kill()
				}
				if err != nil {
					return
				}
			}
		}
	})()
	err = cmd.Wait()
	quit <- true
	if err != nil && cmd.ProcessState.ExitCode() != 0 {
		os.Exit(cmd.ProcessState.ExitCode())
	} else if err != nil {
		panic(err)
	}
}

/*
// 5335 bytes (5.3 kB, 5.2 KiB) copied, 0.00908493 s, 587 kB/s
filePath, err := filepath.Abs(os.Args[1])
if err != nil {log.Fatalln("Unable to resolve path to file.")}
destPath, err := filepath.Abs(os.Args[2])
if err != nil {log.Fatalln("Unable to resolve path to dest.")}
file, err := os.Open(filePath)
if err != nil && os.IsNotExist(err) {log.Fatalln("This file does not exist!")}
else if err != nil {log.Fatalln("An error occurred while opening the file.")}
defer file.Close()
fileStat, err := file.Stat()
if err != nil {log.Fatalln("An error occurred while opening the file.")}
else if !fileStat.Mode().IsRegular() {log.Fatalln("The specified file is not a regular file!")}
// TODO: Untested on macOS or other platforms.
dest, err := os.OpenFile(destPath, os.O_RDWR|os.O_EXCL|os.O_CREATE, os.ModePerm)
if err != nil {log.Fatalln("An error occurred while opening the dest.")}
defer dest.Close()
destStat, err := dest.Stat()
if err != nil {log.Fatalln("An error occurred while opening the file.")}
else if destStat.Mode().IsDir() {log.Fatalln("The specified destination is a directory!")}
var total int
for {
	data := make([]byte, 4096) // TODO: Has to be 512 on Windows.
	n1, err := file.Read(data)
	if err != nil {
		if io.EOF == err {break}
		else {log.Panicln("Encountered error while reading file!")}
	}
	n2, err := dest.Write(data[0:n1])
	if err != nil {log.Panicln("Encountered error while writing to dest!")}
	else if n2 != n1 {log.Panicln("Read/write mismatch! Is the dest too small!")}
	total += n1
} // TODO: Print progress.
err = dest.Sync()
if err != nil {log.Fatalln("Failed to sync writes to disk!")}
*/
