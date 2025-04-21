package platform

import (
	"os"
	"os/exec"
)

type Platform interface {
	OsOpen(name string) (*os.File, error)
	OsGeteuid() int
	RuntimeGOOS() string
	ExecCommand(name string, arg ...string) *exec.Cmd
}
