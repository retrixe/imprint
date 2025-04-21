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
	ExecLookPath(file string) (string, error)
}

type SystemPlatform struct{}

var System Platform = SystemPlatform{}

func (p SystemPlatform) OsOpen(name string) (*os.File, error) {
	return os.Open(name)
}

func (p SystemPlatform) OsGeteuid() int {
	return os.Geteuid()
}

func (p SystemPlatform) RuntimeGOOS() string {
	return runtime.GOOS
}

func (p SystemPlatform) ExecCommand(name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
}

func (p SystemPlatform) ExecLookPath(file string) (string, error) {
	return exec.LookPath(file)
}
