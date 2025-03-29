package app

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Mocks
var osOpen = os.Open
var osGeteuid = os.Geteuid
var runtimeGOOS = runtime.GOOS

// IsElevated returns if the application is running with elevated privileges.
func IsElevated() bool {
	if runtimeGOOS == "windows" { // https://stackoverflow.com/a/59147866
		f, err := osOpen("\\\\.\\PHYSICALDRIVE0")
		if f != nil {
			defer f.Close()
		}
		return err == nil
	}
	return osGeteuid() == 0
}

// ErrPkexecNotFound is returned when `pkexec` (needed on Linux, BSD and the like) is not found.
var ErrPkexecNotFound = errors.New("unable to find `pkexec`, run app with `sudo` directly")

// ErrOsascriptNotFound is returned when `osascript` (needed on macOS) is not found.
var ErrOsascriptNotFound = errors.New("unable to find `osascript`, run app with `sudo` directly")

// ErrWindowsNoOp is returned when attempting to run a command with elevation on Windows.
var ErrWindowsNoOp = errors.New(
	"graphical elevation is unavailable on Windows, run this app as an administrator",
)

// ElevatedCommand executes a command with elevated privileges.
func ElevatedCommand(name string, arg ...string) (*exec.Cmd, error) {
	if IsElevated() {
		return exec.Command(name, arg...), nil
	} else if runtime.GOOS == "windows" {
		// https://stackoverflow.com/questions/31558066/how-to-ask-for-administer-privileges-on-windows-with-go
		return nil, ErrWindowsNoOp
	} else if runtime.GOOS == "darwin" {
		return elevatedMacCommand(name, arg...)
	}
	return elevatedLinuxCommand(name, arg...)
}

func elevatedLinuxCommand(name string, arg ...string) (*exec.Cmd, error) {
	// We used to prefer gksudo over pkexec since it enabled a better prompt.
	// However, gksudo cannot run multiple commands concurrently.
	pkexec, err := exec.LookPath("pkexec")
	if err != nil {
		return nil, ErrPkexecNotFound
	}
	// "Upon successful completion, the return value is the return value of
	// PROGRAM. If the calling process is not authorized or an
	// authorization could not be obtained through authentication or an
	// error occured, pkexec exits with a return value of 127. If the
	// authorization could not be obtained because the user dismissed the
	// authentication dialog, pkexec exits with a return value of 126."
	// pkexec's internal agent is text based, so disable it as this is a GUI.
	display := "DISPLAY=" + os.Getenv("DISPLAY")
	xauthority := "XAUTHORITY=" + os.Getenv("XAUTHORITY")
	args := []string{"--disable-internal-agent", "env", display, xauthority, name}
	cmd := exec.Command(pkexec, append(args, arg...)...)
	return cmd, nil
}

func elevatedMacCommand(name string, args ...string) (*exec.Cmd, error) {
	osascript, err := exec.LookPath("osascript")
	if err != nil {
		return nil, ErrOsascriptNotFound
	}
	command := "exec " + name
	for _, arg := range args {
		command += ` \"` + strings.ReplaceAll(arg, `"`, `\\\"`) + `\"`
	}
	cmd := exec.Command(
		osascript,
		"-e",
		`do shell script "`+command+`" with administrator privileges`,
	)
	return cmd, nil
}
