package app

import (
	"os"
	"os/exec"
	"runtime"
)

type Platform interface {
	OsOpen(name string) (*os.File, error)
	OsGeteuid() int
	RuntimeGOOS() string
	ExecCommand(name string, arg ...string) *exec.Cmd
	ExecCommandOutput(cmd *exec.Cmd) ([]byte, error)
	ExecLookPath(file string) (string, error)
}

type systemPlatform struct{}

var SystemPlatform Platform = systemPlatform{}

func (p systemPlatform) OsOpen(name string) (*os.File, error) {
	return os.Open(name)
}

func (p systemPlatform) OsGeteuid() int {
	return os.Geteuid()
}

func (p systemPlatform) RuntimeGOOS() string {
	return runtime.GOOS
}

func (p systemPlatform) ExecCommand(name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
}

func (p systemPlatform) ExecCommandOutput(cmd *exec.Cmd) ([]byte, error) {
	return cmd.Output()
}

func (p systemPlatform) ExecLookPath(file string) (string, error) {
	return exec.LookPath(file)
}
