package platform

import (
	"os"
	"os/exec"
	"runtime"
)

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
