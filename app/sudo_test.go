package app_test

import (
	"errors"
	"os"
	"os/exec"
	"slices"
	"testing"

	"github.com/retrixe/imprint/app"
)

type mockSudoPlatform struct {
	app.Platform
	*testing.T
	os             string
	elevated       bool
	expectedCmd    string
	expectedCmdArg []string
	elevationAgent string
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
	return &exec.Cmd{Args: append([]string{name}, arg...)}
}

func (p mockSudoPlatform) ExecLookPath(file string) (string, error) {
	if file == p.elevationAgent {
		return "/usr/bin/" + file, nil
	}
	return "", os.ErrNotExist
}

func TestIsElevated(t *testing.T) {
	testCases := []struct {
		name     string
		os       string
		elevated bool
	}{
		{"works when elevated on Windows", "windows", true},
		{"works when not elevated on Windows", "windows", false},
		{"works when elevated on Linux", "linux", true},
		{"works when not elevated on Linux", "linux", false},
		{"works when elevated on macOS", "darwin", true},
		{"works when not elevated on macOS", "darwin", false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			mockPlatform := mockSudoPlatform{T: t, os: testCase.os, elevated: testCase.elevated}
			if app.IsElevated(mockPlatform) != testCase.elevated {
				t.Errorf("Expected IsElevated to return %v", testCase.elevated)
			}
		})
	}
}

func TestElevatedCommand(t *testing.T) {
	testCases := []struct {
		name            string
		platform        app.Platform
		providedCmdName string
		providedCmdArgs []string
		expectedCmdName string
		expectedCmdArgs []string
		expectedErr     error
	}{
		{
			"works when elevated on Windows",
			mockSudoPlatform{os: "windows", elevated: true},
			"cmd.exe",
			[]string{"test1", "test2"},
			"cmd.exe",
			[]string{"test1", "test2"},
			nil,
		},
		{
			"fails when not elevated on Windows",
			mockSudoPlatform{os: "windows", elevated: false},
			"cmd.exe",
			[]string{"test1", "test2"},
			"",
			[]string{},
			app.ErrWindowsNoOp,
		},
		{
			"works when elevated on Linux",
			mockSudoPlatform{os: "linux", elevated: true},
			"bash",
			[]string{"test1", "test2"},
			"bash",
			[]string{"test1", "test2"},
			nil,
		},
		{
			"works when not elevated on Linux and pkexec found",
			mockSudoPlatform{os: "linux", elevated: false, elevationAgent: "pkexec"},
			"bash",
			[]string{"test1", "test2"},
			"/usr/bin/pkexec",
			[]string{"--disable-internal-agent",
				"env", "DISPLAY=" + os.Getenv("DISPLAY"), "XAUTHORITY=" + os.Getenv("XAUTHORITY"),
				"bash", "test1", "test2"},
			nil,
		},
		{
			"fails when not elevated on Linux and pkexec not found",
			mockSudoPlatform{os: "linux", elevated: false},
			"bash",
			[]string{"test1", "test2"},
			"",
			[]string{},
			app.ErrPkexecNotFound,
		},
		{
			"works when elevated on macOS",
			mockSudoPlatform{os: "darwin", elevated: true},
			"bash",
			[]string{"test1", "test2"},
			"bash",
			[]string{"test1", "test2"},
			nil,
		},
		{
			"works when not elevated on macOS and osascript found",
			mockSudoPlatform{os: "darwin", elevated: false, elevationAgent: "osascript"},
			"bash",
			[]string{"test1", "test2"},
			"/usr/bin/osascript",
			[]string{"-e", `do shell script "exec bash \"test1\" \"test2\"" with administrator privileges`},
			nil,
		},
		{
			"fails when not elevated on macOS and osascript not found",
			mockSudoPlatform{os: "darwin", elevated: false},
			"bash",
			[]string{"test1", "test2"},
			"",
			[]string{},
			app.ErrOsascriptNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			mockPlatform := testCase.platform.(mockSudoPlatform)
			mockPlatform.T = t
			mockPlatform.expectedCmd = testCase.expectedCmdName
			mockPlatform.expectedCmdArg = testCase.expectedCmdArgs
			cmd, err := app.ElevatedCommand(mockPlatform, testCase.providedCmdName, testCase.providedCmdArgs...)
			if testCase.expectedErr == nil && err != nil {
				t.Errorf("Expected ElevatedCommand to succeed, got %v", err)
			} else if testCase.expectedErr != nil && !errors.Is(err, testCase.expectedErr) {
				t.Errorf("Expected ElevatedCommand to return error %v, got %v", testCase.expectedErr, err)
			}
			if testCase.expectedErr == nil && cmd == nil {
				t.Errorf("Expected ElevatedCommand to return a command")
			} else if testCase.expectedErr != nil && cmd != nil {
				t.Errorf("Expected ElevatedCommand to return nil command")
			}
		})
	}
}
