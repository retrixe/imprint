package platform

import (
	"os"
)

type Platform interface {
	OsOpen(name string) (*os.File, error)
	OsGeteuid() int
	RuntimeGOOS() string
}
