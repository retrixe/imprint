package app

import (
	"os"
	"testing"

	"github.com/retrixe/imprint/app/platform"
)

type mockWindowsPlatform struct{ platform.Platform }

func (mockWindowsPlatform) RuntimeGOOS() string {
	return "windows"
}

type mockElevatedWindowsPlatform struct{ mockWindowsPlatform }

func (p mockElevatedWindowsPlatform) OsOpen(name string) (*os.File, error) {
	if name != "\\\\.\\PHYSICALDRIVE0" {
		return nil, os.ErrNotExist
	}
	return &os.File{}, nil
}

type mockRegularWindowsPlatform struct{ mockWindowsPlatform }

func (p mockRegularWindowsPlatform) OsOpen(name string) (*os.File, error) {
	if name != "\\\\.\\PHYSICALDRIVE0" {
		return nil, os.ErrNotExist
	}
	return nil, os.ErrPermission
}

type mockUnixPlatform struct{ platform.Platform }

func (mockUnixPlatform) RuntimeGOOS() string {
	return "linux"
}

type mockElevatedUnixPlatform struct{ mockUnixPlatform }

func (p mockElevatedUnixPlatform) OsGeteuid() int {
	return 0
}

type mockRegularUnixPlatform struct{ mockUnixPlatform }

func (p mockRegularUnixPlatform) OsGeteuid() int {
	return -1
}

func TestIsElevated(t *testing.T) {
	t.Run("works when elevated on Windows", func(t *testing.T) {
		t.Parallel()
		if !IsElevated(mockElevatedWindowsPlatform{}) {
			t.Errorf("Expected IsElevated to return true")
		}
	})
	t.Run("works when not elevated on Windows", func(t *testing.T) {
		t.Parallel()
		if IsElevated(mockRegularWindowsPlatform{}) {
			t.Errorf("Expected IsElevated to return false")
		}
	})
	t.Run("works when elevated on Linux", func(t *testing.T) {
		t.Parallel()
		if !IsElevated(mockElevatedUnixPlatform{}) {
			t.Errorf("Expected IsElevated to return true")
		}
	})
	t.Run("works when not elevated on Linux", func(t *testing.T) {
		t.Parallel()
		if IsElevated(mockRegularUnixPlatform{}) {
			t.Errorf("Expected IsElevated to return false")
		}
	})
}
