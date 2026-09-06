//go:build !darwin && !windows

package imaging

import (
	"io/fs"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Cheers to https://stackoverflow.com/a/6525975
var kvRegex = regexp.MustCompile(`([a-zA-Z0-9_-]+)=("[^"\\]*(?:\\.[^"\\]*)*")`)

// man getmntent(3) says that mountpoints are escaped in /proc/mounts,
// so we need to unescape them before passing them to umount.
var mountpointUnescaper = strings.NewReplacer(
	`\040`, " ",
	`\011`, "\t",
	`\012`, "\n",
	`\134`, "\\",
)

var systemMountpoints = []string{
	"/", "/usr", "/home", "/boot", "/boot/efi", "/var", "/efi",
	// Live media handling for Fedora, Debian, Ubuntu casper
	"/run/initramfs/live", "/run/live/medium", "/lib/live/mount/medium", "/cdrom",
}

type mount struct{ device, mountpoint string }

func readMounts(platform Platform) ([]mount, error) {
	// Discover mounted device partitions.
	mounts, err := platform.OsReadFile("/proc/mounts")
	if err != nil {
		return nil, err
	}

	mountedDevices := make([]mount, 0)
	for _, mountLine := range strings.Split(string(mounts), "\n") {
		fields := strings.Fields(mountLine)
		if len(fields) < 2 {
			continue
		}
		device := fields[0]
		mountpoint := mountpointUnescaper.Replace(fields[1])
		mountedDevices = append(mountedDevices, mount{device, mountpoint})
	}

	return mountedDevices, nil
}

func parseLsblkFields(line string) map[string]string {
	row := make(map[string]string)
	for _, match := range kvRegex.FindAllStringSubmatch(line, -1) {
		value, err := strconv.Unquote(match[2])
		if err != nil {
			value = strings.Trim(match[2], `"`)
		}
		row[match[1]] = value
	}
	return row
}

// findSystemDevices returns the KNAMEs of disks mounted at critical system mountpoints.
func findSystemDevices(platform Platform, mounts []mount) []string {
	// Find devices mounted on system critical mountpoints
	var systemMounts []string
	for _, m := range mounts {
		if slices.Contains(systemMountpoints, m.mountpoint) && !slices.Contains(systemMounts, m.device) {
			systemMounts = append(systemMounts, m.device)
		}
	}

	var systemDevices []string
	// Go through all systemMounts to find their parent disk
	for _, source := range systemMounts {
		// Exclude non-disk sources like overlay, tmpfs or a zfs dataset.
		if !strings.HasPrefix(source, "/dev/") {
			continue
		}

		// -s = --inverse (list the device's parents instead of its children)
		res, err := platform.ExecCommandOutput(platform.ExecCommand(
			"lsblk", "--pairs", "-s", "-o", "KNAME,TYPE", source))
		if err != nil {
			continue // Allow listing to continue even if this source is not real (e.g. /dev/root)
		}

		// Find the disks in the listing (usually the last entry) and insert it into systemDevices
		for _, line := range strings.Split(strings.TrimSpace(string(res)), "\n") {
			fields := parseLsblkFields(line)
			if fields["TYPE"] == "disk" && !slices.Contains(systemDevices, fields["KNAME"]) {
				systemDevices = append(systemDevices, fields["KNAME"])
			}
		}
	}
	return systemDevices
}

// GetDevices returns the list of USB devices available to read/write from.
func GetDevices(platform Platform) ([]Device, error) {
	// --pairs
	// TODO: -J = --json (available since Ubuntu 16.04)
	// -d = --nodeps
	// -b = --bytes
	// -o = --output
	res, err := platform.ExecCommandOutput(platform.ExecCommand(
		"lsblk", "--pairs", "-d", "-b", "-o", "KNAME,TYPE,RM,SIZE,TRAN,MODEL"))
	if err != nil {
		return nil, err
	}

	// Identify all system mounts and exclude devices mounted on system critical mountpoints
	mounts, err := readMounts(platform)
	if err != nil {
		return nil, err
	}
	systemDevices := findSystemDevices(platform, mounts)

	devices := []Device{}
	for _, deviceString := range strings.Split(strings.TrimSpace(string(res)), "\n") {
		deviceInfo := parseLsblkFields(deviceString)

		// Display only removable, USB and IEEE1394 disks
		// https://lxr.kde.org/source/frameworks/solid/src/solid/devices/backends/udisks2/udisksstoragedrive.cpp
		// TODO: Exclude UDISKS_SYSTEM=1 if set on udev
		if deviceInfo["TYPE"] != "disk" ||
			(deviceInfo["RM"] != "1" && deviceInfo["TRAN"] != "usb" && deviceInfo["TRAN"] != "sbp") {
			continue
		}
		// Exclude any "system" devices from being enumerated
		if slices.Contains(systemDevices, deviceInfo["KNAME"]) {
			continue
		}

		bytes, _ := strconv.Atoi(deviceInfo["SIZE"])
		devices = append(devices, Device{
			Model: deviceInfo["MODEL"],
			Name:  "/dev/" + deviceInfo["KNAME"],
			Size:  BytesToString(bytes, false),
			Bytes: bytes,
		})
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
	mounts, err := readMounts(platform)
	if err != nil {
		return err
	}

	// Unmount device partitions.
	for _, mount := range mounts {
		mountpoint, mountedDevice := mount.mountpoint, mount.device
		if strings.HasPrefix(mountedDevice, device) {
			if err := platform.SyscallUnmount(mountpoint, 0); err != nil {
				return err
			}
		}
	}
	return nil
}
