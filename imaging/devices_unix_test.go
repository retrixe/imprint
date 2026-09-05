//go:build !darwin && !windows

package imaging_test

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"testing"

	"github.com/retrixe/imprint/imaging"
)

type mockDevicesPlatform struct {
	imaging.Platform
	*testing.T
	allowedCmds map[string][]mockDevicesPlatformCommand
	readFiles   map[string][]byte
}

type mockDevicesPlatformCommand struct {
	args   []string
	output []byte
	err    error
}

func (p mockDevicesPlatform) ExecCommand(name string, arg ...string) *exec.Cmd {
	cmd := &exec.Cmd{Path: name, Args: arg}
	for allowedCmdName, allowedCmds := range p.allowedCmds {
		if name != allowedCmdName {
			continue
		}
		for _, allowedCmd := range allowedCmds {
			if slices.Equal(arg, allowedCmd.args) {
				cmd.Err = allowedCmd.err
				return cmd
			}
		}
		p.T.Errorf("ExecCommand called with unexpected args for %s: %v", name, arg)
		cmd.Err = fmt.Errorf("ExecCommand called with unexpected args for %s: %v", name, arg)
		return cmd
	}
	cmd.Err = exec.ErrNotFound
	return cmd
}

func (p mockDevicesPlatform) ExecCommandOutput(cmd *exec.Cmd) ([]byte, error) {
	if cmd.Err != nil {
		return nil, cmd.Err
	}
	for _, allowedCmd := range p.allowedCmds[cmd.Path] {
		if slices.Equal(cmd.Args, allowedCmd.args) {
			return allowedCmd.output, nil
		}
	}
	return nil, exec.ErrNotFound
}

func (p mockDevicesPlatform) OsReadFile(name string) ([]byte, error) {
	if contents, ok := p.readFiles[name]; ok {
		return contents, nil
	}
	return nil, os.ErrNotExist
}

var lsblkExitError = errors.New("lsblk mock error")

func TestGetDevices(t *testing.T) {
	t.Parallel()

	lsblkListArgs := []string{"--pairs", "-d", "-b", "-o", "KNAME,TYPE,RM,SIZE,TRAN,MODEL"}
	procMounts := map[string][]byte{
		"/proc/mounts": []byte("" +
			"/dev/mapper/luks-283e2319-0541-4588-93ef-a2687dd09fc7 / btrfs rw,relatime 0 0\n" +
			"/dev/mapper/luks-283e2319-0541-4588-93ef-a2687dd09fc7 /home btrfs rw,relatime 0 0\n"),
	}

	testCases := []struct {
		name            string
		cmds            map[string][]mockDevicesPlatformCommand
		readFiles       map[string][]byte
		expectedDevices []imaging.Device
		expectedError   error
	}{
		{
			"fails upon missing lsblk",
			map[string][]mockDevicesPlatformCommand{},
			procMounts,
			[]imaging.Device{},
			exec.ErrNotFound,
		},
		{
			"fails upon lsblk error",
			map[string][]mockDevicesPlatformCommand{
				"lsblk": {{
					args: lsblkListArgs,
					err:  lsblkExitError,
				}},
			},
			procMounts,
			[]imaging.Device{},
			lsblkExitError,
		},
		{
			"fails upon missing /proc/mounts",
			map[string][]mockDevicesPlatformCommand{
				"lsblk": {{
					args:   lsblkListArgs,
					output: []byte(`KNAME="zram0" TYPE="disk" RM="0" SIZE="8589934592" TRAN="" MODEL=""` + "\n"),
				}},
			},
			nil,
			[]imaging.Device{},
			os.ErrNotExist,
		},
		{
			"works on Fedora 42 on ASUS Zenbook S 14 w/ dual boot, btrfs, LUKS with 0 devices attached",
			map[string][]mockDevicesPlatformCommand{
				"lsblk": {
					{args: lsblkListArgs, output: []byte(`KNAME="zram0" TYPE="disk" RM="0" SIZE="8589934592" TRAN="" MODEL=""` + "\n" +
						`KNAME="nvme0n1" TYPE="disk" RM="0" SIZE="1024209543168" TRAN="nvme" MODEL="WD PC SN560 SDDPNQE-1T00-1102"` + "\n")},
				},
			},
			procMounts,
			[]imaging.Device{},
			nil,
		},
		{
			"works on Fedora 42 on ASUS Zenbook S 14 w/ dual boot, btrfs, LUKS with 1 device attached",
			map[string][]mockDevicesPlatformCommand{
				"lsblk": {
					{args: lsblkListArgs, output: []byte(`KNAME="sda" TYPE="disk" RM="1" SIZE="2000748032" TRAN="usb" MODEL="Cruzer"` + "\n" +
						`KNAME="zram0" TYPE="disk" RM="0" SIZE="8589934592" TRAN="" MODEL=""` + "\n" +
						`KNAME="nvme0n1" TYPE="disk" RM="0" SIZE="1024209543168" TRAN="nvme" MODEL="WD PC SN560 SDDPNQE-1T00-1102"` + "\n")},
				},
			},
			procMounts,
			[]imaging.Device{
				{Name: "/dev/sda", Model: "Cruzer", Size: imaging.BytesToString(2000748032, false), Bytes: 2000748032},
			},
			nil,
		},
		{
			"works on Fedora 42 on ASUS Zenbook S 14 w/ dual boot, btrfs, LUKS with 2 devices attached",
			map[string][]mockDevicesPlatformCommand{
				"lsblk": {
					{args: lsblkListArgs, output: []byte(`KNAME="sda" TYPE="disk" RM="1" SIZE="2000748032" TRAN="usb" MODEL="Cruzer"` + "\n" +
						`KNAME="sdb" TYPE="disk" RM="1" SIZE="61530439680" TRAN="usb" MODEL="SanDisk 3.2Gen1"` + "\n" +
						`KNAME="zram0" TYPE="disk" RM="0" SIZE="8589934592" TRAN="" MODEL=""` + "\n" +
						`KNAME="nvme0n1" TYPE="disk" RM="0" SIZE="1024209543168" TRAN="nvme" MODEL="WD PC SN560 SDDPNQE-1T00-1102"` + "\n")},
				},
			},
			procMounts,
			[]imaging.Device{
				{Name: "/dev/sda", Model: "Cruzer", Size: imaging.BytesToString(2000748032, false), Bytes: 2000748032},
				{Name: "/dev/sdb", Model: "SanDisk 3.2Gen1", Size: imaging.BytesToString(61530439680, false), Bytes: 61530439680},
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
				readFiles:   testCase.readFiles,
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
