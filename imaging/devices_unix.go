//go:build !darwin && !windows

package imaging

import (
	"io/fs"
	"regexp"
	"strconv"
	"strings"
)

// Device is a struct representing a block device.
type Device struct {
	Name  string
	Model string
	Size  string
	Bytes int
}

// Cheers to https://stackoverflow.com/a/6525975
var kvRegex = regexp.MustCompile(`([a-zA-Z0-9_-]+)=("[^"\\]*(?:\\.[^"\\]*)*")`)

// GetDevices returns the list of USB devices available to read/write from.
func GetDevices(platform Platform) ([]Device, error) {
	// --pairs
	// TODO: -J = --json (available since Ubuntu 16.04)
	// -d = --nodeps
	// -b = --bytes
	// -o = --output
	res, err := platform.ExecCommandOutput(platform.ExecCommand(
		"lsblk", "--pairs", "-d", "-b", "-o", "KNAME,TYPE,RM,SIZE,MODEL"))
	if err != nil {
		return nil, err
	}
	deviceStrings := strings.Split(strings.TrimSpace(string(res)), "\n")

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
		kv := kvRegex.FindAllStringSubmatch(deviceString, -1)
		deviceInfo := make(map[string]string)
		for _, match := range kv {
			key, value := match[1], match[2]
			deviceInfo[key], err = strconv.Unquote(value)
			if err != nil {
				deviceInfo[key] = strings.Trim(value, `"`) // Ideally, we should not hit this, but fallback
			}
		}

		if deviceInfo["TYPE"] == "disk" && deviceInfo["RM"] == "1" {
			// Exclude any "system" devices (as defined by /etc/fstab) from being enumerated
			for _, systemDevice := range systemDevices {
				if strings.HasPrefix(systemDevice, "/dev/"+deviceInfo["KNAME"]) {
					continue nextDevice
				}
			}
			bytes, _ := strconv.Atoi(deviceInfo["SIZE"])
			device := Device{
				Model: deviceInfo["MODEL"],
				Name:  "/dev/" + deviceInfo["KNAME"],
				Size:  BytesToString(bytes, false),
				Bytes: bytes,
			}

			devices = append(devices, device)
		}
	}

	return devices, nil
}

// UnmountDevice unmounts a block device's partitions before flashing to it.
func UnmountDevice(device string) error {
	return UnmountDeviceWithPlatform(UnixSystemPlatform, device)
}

// UnmountDevice unmounts a block device's partitions before flashing to it.
// It accepts a [UnixPlatform] to allow for testing with a mock platform.
func UnmountDeviceWithPlatform(platform UnixPlatform, device string) error {
	// Check if device exists and is a block device.
	stat, err := platform.OsStat(device)
	if err != nil {
		return err
	} else if stat.Mode().Type()&fs.ModeDevice == 0 {
		return ErrNotBlockDevice
	}

	// Discover mounted device partitions.
	mounts, err := platform.OsReadFile("/proc/mounts")
	if err != nil {
		return err
	}

	// Unmount device partitions.
	mountpointUnescaper := strings.NewReplacer(
		// man getmntent(3) says that mountpoints are escaped in /proc/mounts,
		// so we need to unescape them before passing them to umount.
		`\040`, " ",
		`\011`, "\t",
		`\012`, "\n",
		`\134`, "\\",
	)
	for _, mount := range strings.Split(string(mounts), "\n") {
		if strings.HasPrefix(mount, device) {
			mountpoint := strings.Fields(mount)[1]
			mountpoint = mountpointUnescaper.Replace(mountpoint)
			if err := platform.SyscallUnmount(mountpoint, 0); err != nil {
				return err
			}
		}
	}
	return nil
}
