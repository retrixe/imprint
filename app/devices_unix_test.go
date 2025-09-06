//go:build !darwin && !windows

package app_test

import (
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"testing"

	"github.com/retrixe/imprint/app"
)

type mockDevicesPlatform struct {
	app.Platform
	*testing.T
	allowedCmds map[string]mockDevicesPlatformCommand
}

type mockDevicesPlatformCommand struct {
	args   []string
	output []byte
	err    error
}

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

func TestGetDevices(t *testing.T) {
	t.Parallel()

	var lsblkExitError = errors.New("lsblk mock error")

	var dfExitError = errors.New("df mock error")

	testCases := []struct {
		name            string
		cmds            map[string]mockDevicesPlatformCommand
		expectedDevices []app.Device
		expectedError   error
	}{
		{
			"fails upon missing lsblk",
			map[string]mockDevicesPlatformCommand{},
			[]app.Device{},
			exec.ErrNotFound,
		},
		{
			"fails upon lsblk error",
			map[string]mockDevicesPlatformCommand{
				"lsblk": {
					args: []string{"-d", "-b", "-o", "KNAME,TYPE,RM,SIZE,MODEL"},
					err:  lsblkExitError,
				},
			},
			[]app.Device{},
			lsblkExitError,
		},
		{
			"fails upon missing df",
			map[string]mockDevicesPlatformCommand{
				"lsblk": {
					args:   []string{"-d", "-b", "-o", "KNAME,TYPE,RM,SIZE,MODEL"},
					output: []byte("KNAME   TYPE RM          SIZE MODEL\nzram0   disk  0    8589934592 \n"),
				},
			},
			[]app.Device{},
			exec.ErrNotFound,
		},
		{
			"fails upon df error",
			map[string]mockDevicesPlatformCommand{
				"lsblk": {
					args:   []string{"-d", "-b", "-o", "KNAME,TYPE,RM,SIZE,MODEL"},
					output: []byte("KNAME   TYPE RM          SIZE MODEL\nzram0   disk  0    8589934592 \n"),
				},
				"df": {
					args: []string{"/", "/home"},
					err:  dfExitError,
				},
			},
			[]app.Device{},
			dfExitError,
		},
		{
			"works on Fedora 42 on ASUS Zenbook S 14 w/ dual boot, btrfs, LUKS with 0 devices attached",
			map[string]mockDevicesPlatformCommand{
				"lsblk": {
					args: []string{"-d", "-b", "-o", "KNAME,TYPE,RM,SIZE,MODEL"},
					output: []byte("KNAME   TYPE RM          SIZE MODEL\n" +
						"zram0   disk  0    8589934592 \n" +
						"nvme0n1 disk  0 1024209543168 WD PC SN560 SDDPNQE-1T00-1102\n"),
				},
				"df": {
					args: []string{"/", "/home"},
					output: []byte("Filesystem                                            1K-blocks      Used Available Use% Mounted on\n" +
						"/dev/mapper/luks-283e2319-0541-4588-93ef-a2687dd09fc7 535805952 503377676  28342100  95% /\n" +
						"/dev/mapper/luks-283e2319-0541-4588-93ef-a2687dd09fc7 535805952 503377676  28342100  95% /home\n"),
				},
			},
			[]app.Device{},
			nil,
		},
		{
			"works on Fedora 42 on ASUS Zenbook S 14 w/ dual boot, btrfs, LUKS with 1 device attached",
			map[string]mockDevicesPlatformCommand{
				"lsblk": {
					args: []string{"-d", "-b", "-o", "KNAME,TYPE,RM,SIZE,MODEL"},
					output: []byte("KNAME   TYPE RM          SIZE MODEL\n" +
						"sda     disk  1    2000748032 Cruzer\n" +
						"zram0   disk  0    8589934592 \n" +
						"nvme0n1 disk  0 1024209543168 WD PC SN560 SDDPNQE-1T00-1102\n"),
				},
				"df": {
					args: []string{"/", "/home"},
					output: []byte("Filesystem                                            1K-blocks      Used Available Use% Mounted on\n" +
						"/dev/mapper/luks-283e2319-0541-4588-93ef-a2687dd09fc7 535805952 503377676  28342100  95% /\n" +
						"/dev/mapper/luks-283e2319-0541-4588-93ef-a2687dd09fc7 535805952 503377676  28342100  95% /home\n"),
				},
			},
			[]app.Device{
				{Name: "/dev/sda", Model: "Cruzer", Size: app.BytesToString(2000748032, false), Bytes: 2000748032},
			},
			nil,
		},
		{
			"works on Fedora 42 on ASUS Zenbook S 14 w/ dual boot, btrfs, LUKS with 2 devices attached",
			map[string]mockDevicesPlatformCommand{
				"lsblk": {
					args: []string{"-d", "-b", "-o", "KNAME,TYPE,RM,SIZE,MODEL"},
					output: []byte("KNAME   TYPE RM          SIZE MODEL\n" +
						"sda     disk  1    2000748032 Cruzer\n" +
						"sdb     disk  1   61530439680 SanDisk 3.2Gen1\n" +
						"zram0   disk  0    8589934592 \n" +
						"nvme0n1 disk  0 1024209543168 WD PC SN560 SDDPNQE-1T00-1102\n"),
				},
				"df": {
					args: []string{"/", "/home"},
					output: []byte("Filesystem                                            1K-blocks      Used Available Use% Mounted on\n" +
						"/dev/mapper/luks-283e2319-0541-4588-93ef-a2687dd09fc7 535805952 503377676  28342100  95% /\n" +
						"/dev/mapper/luks-283e2319-0541-4588-93ef-a2687dd09fc7 535805952 503377676  28342100  95% /home\n"),
				},
			},
			[]app.Device{
				{Name: "/dev/sda", Model: "Cruzer", Size: app.BytesToString(2000748032, false), Bytes: 2000748032},
				{Name: "/dev/sdb", Model: "SanDisk 3.2Gen1", Size: app.BytesToString(61530439680, false), Bytes: 61530439680},
			},
			nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			devices, err := app.GetDevices(mockDevicesPlatform{
				Platform:    app.SystemPlatform,
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
