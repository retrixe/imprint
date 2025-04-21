package app

import (
	"errors"
	"os"
	"os/exec"
	"slices"
	"testing"

	"github.com/retrixe/imprint/app/platform"
)

type mockSudoPlatform struct {
	platform.Platform
	*testing.T
	os             string
	elevated       bool
	expectedCmd    string
	expectedCmdArg []string
}

func (p mockSudoPlatform) RuntimeGOOS() string {
	return p.os
}

func (p mockSudoPlatform) OsOpen(name string) (*os.File, error) {
	if p.os == "windows" {
		if name != "\\\\.\\PHYSICALDRIVE0" {
			p.T.Errorf("OsOpen called with unexpected name: %s", name)
			return nil, os.ErrNotExist
		}
		if p.elevated {
			return &os.File{}, nil
		} else {
			return nil, os.ErrPermission
		}
	} else {
		p.T.Errorf("OsOpen called on non-windows platform")
		return nil, os.ErrNotExist
	}
}

func (p mockSudoPlatform) OsGeteuid() int {
	if p.os == "linux" || p.os == "darwin" {
		if p.elevated {
			return 0
		} else {
			return -1
		}
	} else {
		p.T.Errorf("OsGeteuid called on non-unix platform")
		return -1
	}
}

func (p mockSudoPlatform) ExecCommand(name string, arg ...string) *exec.Cmd {
	if name != p.expectedCmd {
		p.T.Errorf("ExecCommand called with unexpected name: %s", name)
	} else if !slices.Equal(arg, p.expectedCmdArg) {
		p.T.Errorf("ExecCommand called with unexpected args: %v", arg)
	}
	return &exec.Cmd{}
}

func TestIsElevated(t *testing.T) {
	t.Run("works when elevated on Windows", func(t *testing.T) {
		t.Parallel()
		if !IsElevated(mockSudoPlatform{T: t, os: "windows", elevated: true}) {
			t.Errorf("Expected IsElevated to return true")
		}
	})
	t.Run("works when not elevated on Windows", func(t *testing.T) {
		t.Parallel()
		if IsElevated(mockSudoPlatform{T: t, os: "windows", elevated: false}) {
			t.Errorf("Expected IsElevated to return false")
		}
	})
	t.Run("works when elevated on Linux", func(t *testing.T) {
		t.Parallel()
		if !IsElevated(mockSudoPlatform{T: t, os: "linux", elevated: true}) {
			t.Errorf("Expected IsElevated to return true")
		}
	})
	t.Run("works when not elevated on Linux", func(t *testing.T) {
		t.Parallel()
		if IsElevated(mockSudoPlatform{T: t, os: "linux", elevated: false}) {
			t.Errorf("Expected IsElevated to return false")
		}
	})
	t.Run("works when elevated on macOS", func(t *testing.T) {
		t.Parallel()
		if !IsElevated(mockSudoPlatform{T: t, os: "darwin", elevated: true}) {
			t.Errorf("Expected IsElevated to return true")
		}
	})
	t.Run("works when not elevated on macOS", func(t *testing.T) {
		t.Parallel()
		if IsElevated(mockSudoPlatform{T: t, os: "darwin", elevated: false}) {
			t.Errorf("Expected IsElevated to return false")
		}
	})
}

func TestElevatedCommand(t *testing.T) {
	t.Run("works when elevated on Windows", func(t *testing.T) {
		t.Parallel()
		mockPlatform := mockSudoPlatform{T: t, os: "windows", elevated: true, expectedCmd: "cmd.exe"}
		mockPlatform.expectedCmdArg = []string{"test1", "test2"}
		cmd, err := ElevatedCommand(mockPlatform, "cmd.exe", "test1", "test2")
		if err != nil {
			t.Errorf("Expected ElevatedCommand to succeed, got %v", err)
		}
		if cmd == nil {
			t.Errorf("Expected ElevatedCommand to return a command")
		}
	})
	t.Run("fails when not elevated on Windows", func(t *testing.T) {
		t.Parallel()
		mockPlatform := mockSudoPlatform{T: t, os: "windows", elevated: false, expectedCmd: "cmd.exe"}
		mockPlatform.expectedCmdArg = []string{"test1", "test2"}
		cmd, err := ElevatedCommand(mockPlatform, "cmd.exe", "test1", "test2")
		if err == nil {
			t.Errorf("Expected ElevatedCommand to fail, got %v", cmd)
		}
		if !errors.Is(err, ErrWindowsNoOp) {
			t.Errorf("Expected ElevatedCommand to return ErrWindowsNoOp, got %v", err)
		}
		if cmd != nil {
			t.Errorf("Expected ElevatedCommand to return nil command")
		}
	})
}
