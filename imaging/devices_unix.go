//go:build !darwin && !windows

package imaging

import (
	"io/fs"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// Device is a struct representing a block device.
type Device struct {
	Name  string
	Model string
	Size  string
	Bytes int
}

// GetDevices returns the list of USB devices available to read/write from.
func GetDevices(platform Platform) ([]Device, error) {
	// TODO: -J = --json (available since Ubuntu 16.04)
	// -d = --nodeps
	// -b = --bytes
	// -o = --output
	res, err := platform.ExecCommandOutput(platform.ExecCommand(
		"lsblk", "-d", "-b", "-o", "KNAME,TYPE,RM,SIZE,MODEL"))
	if err != nil {
		return nil, err
	}
	deviceStrings := strings.Split(string(res), "\n")
	deviceStrings = deviceStrings[:len(deviceStrings)-1]

	// FIXME: Iterate through /etc/fstab for all system mounts (skip noauto,nofail)
	res, err = platform.ExecCommandOutput(platform.ExecCommand("df", "/", "/home"))
	if err != nil {
		return nil, err
	}

	systemDevices := strings.Split(strings.TrimSpace(string(res)), "\n")
	for idx, device := range systemDevices {
		systemDevices[idx] = strings.Fields(device)[0]
		// FIXME: Get the parent device of each of those devices (PKNAME in lsblk)
	}

	devices := []Device{}

nextDevice:
	for _, deviceString := range deviceStrings {
		deviceFields := strings.Fields(deviceString)
		if deviceFields[1] == "disk" && deviceFields[2] == "1" {
			// Exclude any "system" devices (as defined by /etc/fstab) from being enumerated
			for _, systemDevice := range systemDevices {
				if strings.HasPrefix(systemDevice, "/dev/"+deviceFields[0]) {
					continue nextDevice
				}
			}
			bytes, _ := strconv.Atoi(deviceFields[3])
			device := Device{
				Name:  "/dev/" + deviceFields[0],
				Size:  BytesToString(bytes, false),
				Bytes: bytes,
			}

			if len(deviceFields) >= 4 {
				device.Model = strings.TrimSpace(strings.Join(deviceFields[4:], " "))
			}

			devices = append(devices, device)
		}
	}

	return devices, nil
}

// UnmountDevice unmounts a block device's partitons before flashing to it.
func UnmountDevice(device string) error {
	// FIXME: Write unit tests
	// Check if device is mounted.
	stat, err := os.Stat(device)
	if err != nil {
		return err
	} else if stat.Mode().Type()&fs.ModeDevice == 0 {
		return ErrNotBlockDevice
	}
	// Discover mounted device partitions.
	mounts, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return err
	}
	// Unmount device partitions.
	for _, mount := range strings.Split(string(mounts), "\n") {
		if strings.HasPrefix(mount, device) {
			mountpoint := strings.Fields(mount)[0]
			err = syscall.Unmount(mountpoint, 0)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
