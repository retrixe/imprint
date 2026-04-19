//go:build darwin

package imaging_test

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"slices"
	"testing"
	"time"

	"github.com/retrixe/imprint/imaging"
)

type mockDevicesPlatform struct {
	imaging.Platform
	*testing.T
	allowedCmds  map[string]mockDevicesPlatformCommand
	allowedFiles map[string]fakeFileInfo
}

type mockDevicesPlatformCommand struct {
	args   []string
	output []byte
	err    error
}

type fakeFileInfo struct {
	mode fs.FileMode
}

func (f fakeFileInfo) Name() string       { return "" }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() fs.FileMode  { return f.mode }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return false }
func (f fakeFileInfo) Sys() any           { return nil }

func (p mockDevicesPlatform) ExecCommand(name string, arg ...string) *exec.Cmd {
	cmd := &exec.Cmd{Path: name, Args: arg}
	for allowedCmdName, allowedCmd := range p.allowedCmds {
		if name == allowedCmdName && slices.Equal(arg, allowedCmd.args) {
			cmd.Err = allowedCmd.err
			return cmd
		} else if name == allowedCmdName {
			p.T.Errorf("ExecCommand called with unexpected args for %s: %v", name, arg)
			cmd.Err = fmt.Errorf("ExecCommand called with unexpected args for %s: %v", name, arg)
			return cmd
		}
	}
	cmd.Err = exec.ErrNotFound
	return cmd
}

func (p mockDevicesPlatform) ExecCommandOutput(cmd *exec.Cmd) ([]byte, error) {
	if cmd.Err != nil {
		return nil, cmd.Err
	}
	return p.allowedCmds[cmd.Path].output, nil
}

func (m mockDevicesPlatform) OsStat(name string) (fs.FileInfo, error) {
	fileInfo, ok := m.allowedFiles[name]
	if ok {
		return fileInfo, nil
	}
	return nil, os.ErrNotExist
}

//go:embed test_outputs/diskutil_macbook_air_m4_vanilla_macos_26_0_devices.txt
var diskutilMacBookAirM4VanillaMacOS26NoDeviceOutput []byte

//go:embed test_outputs/diskutil_macbook_air_m4_vanilla_macos_26_1_device.txt
var diskutilMacBookAirM4VanillaMacOS26OneDeviceOutput []byte

var diskutilMockError = errors.New("diskutil mock error")

func TestGetDevices(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		cmds            map[string]mockDevicesPlatformCommand
		expectedDevices []imaging.Device
		expectedError   error
	}{
		{
			"fails upon missing diskutil",
			map[string]mockDevicesPlatformCommand{},
			[]imaging.Device{},
			exec.ErrNotFound,
		},
		{
			"fails upon diskutil error",
			map[string]mockDevicesPlatformCommand{
				"diskutil": {
					args: []string{"info", "-all"},
					err:  diskutilMockError,
				},
			},
			[]imaging.Device{},
			diskutilMockError,
		},
		{
			"works on MacBook Air M4, vanilla macOS 26 with 0 devices attached",
			map[string]mockDevicesPlatformCommand{
				"diskutil": {
					args:   []string{"info", "-all"},
					output: diskutilMacBookAirM4VanillaMacOS26NoDeviceOutput,
				},
			},
			[]imaging.Device{},
			nil,
		},
		{
			"works on MacBook Air M4, vanilla macOS 26 with 1 device attached",
			map[string]mockDevicesPlatformCommand{
				"diskutil": {
					args:   []string{"info", "-all"},
					output: diskutilMacBookAirM4VanillaMacOS26OneDeviceOutput,
				},
			},
			[]imaging.Device{
				{Name: "/dev/disk8", Model: "DataTraveler 3.0", Size: imaging.BytesToString(30943995904, false), Bytes: 30943995904},
			},
			nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			devices, err := imaging.GetDevices(mockDevicesPlatform{
				T:           t,
				allowedCmds: testCase.cmds,
			})
			if !errors.Is(err, testCase.expectedError) {
				t.Errorf("expected error %v, got %v", testCase.expectedError, err)
			} else if !slices.Equal(devices, testCase.expectedDevices) {
				if len(devices) != len(testCase.expectedDevices) {
					t.Errorf("expected %d devices, got %d", len(testCase.expectedDevices), len(devices))
				} else {
					for i := range devices {
						if devices[i] != testCase.expectedDevices[i] {
							t.Errorf("expected device %+v, got %+v", testCase.expectedDevices[i], devices[i])
						}
					}
				}
			}
		})
	}
}

func TestUnmountDeviceWithPlatform(t *testing.T) {
	t.Parallel()

	t.Run("stat error bubbles up", func(t *testing.T) {
		t.Parallel()
		err := imaging.UnmountDeviceWithPlatform(mockDevicesPlatform{
			T: t,
		}, "/dev/diskX")
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected stat error, got %v", err)
		}
	})

	t.Run("not a block device", func(t *testing.T) {
		t.Parallel()
		err := imaging.UnmountDeviceWithPlatform(mockDevicesPlatform{
			T: t,
			allowedFiles: map[string]fakeFileInfo{
				"/dev/diskX": {mode: 0},
			},
		}, "/dev/diskX")
		if !errors.Is(err, imaging.ErrNotBlockDevice) {
			t.Fatalf("expected stat error, got %v", err)
		}
	})

	t.Run("fails upon missing diskutil", func(t *testing.T) {
		t.Parallel()
		err := imaging.UnmountDeviceWithPlatform(mockDevicesPlatform{
			T: t,
			allowedFiles: map[string]fakeFileInfo{
				"/dev/diskX": {mode: os.ModeDevice},
			},
		}, "/dev/diskX")
		if !errors.Is(err, exec.ErrNotFound) {
			t.Fatalf("expected exec error, got %v", err)
		}
	})

	t.Run("fails upon diskutil error", func(t *testing.T) {
		t.Parallel()
		err := imaging.UnmountDeviceWithPlatform(mockDevicesPlatform{
			T: t,
			allowedFiles: map[string]fakeFileInfo{
				"/dev/diskX": {mode: os.ModeDevice},
			},
			allowedCmds: map[string]mockDevicesPlatformCommand{
				"diskutil": {
					args:   []string{"unmountDisk", "/dev/diskX"},
					output: []byte(""),
					err:    diskutilMockError,
				},
			},
		}, "/dev/diskX")
		if !errors.Is(err, diskutilMockError) {
			t.Fatalf("expected exec error, got %v", err)
		}
	})

	t.Run("successful unmounts", func(t *testing.T) {
		t.Parallel()
		err := imaging.UnmountDeviceWithPlatform(mockDevicesPlatform{
			T: t,
			allowedFiles: map[string]fakeFileInfo{
				"/dev/diskX": {mode: os.ModeDevice},
			},
			allowedCmds: map[string]mockDevicesPlatformCommand{
				"diskutil": {
					args:   []string{"unmountDisk", "/dev/diskX"},
					output: []byte(""),
					err:    nil,
				},
			},
		}, "/dev/diskX")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}
