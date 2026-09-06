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

// man getmntent(3) says that mountpoints and sources are escaped in /proc/mounts,
// so we need to unescape them before passing them to umount.
var mountUnescaper = strings.NewReplacer(
	`\040`, " ",
	`\011`, "\t",
	`\012`, "\n",
	`\043`, "#",
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
		device := mountUnescaper.Replace(fields[0])
		mountpoint := mountUnescaper.Replace(fields[1])
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

// findSystemDisks returns the KNAMEs of disks mounted at critical system mountpoints.
func findSystemDisks(platform Platform, mounts []mount) []string {
	// TODO: Use a single lsblk listing and use PKNAME to walk up the mounted device tree.
	// Notes: Some things which may not work (as reviewed by Claude Fable 5.1)
	//   - Swap partitions, which never appear in /proc/mounts.
	//   - Multi-device btrfs (only the member named in /proc/mounts is resolved) and
	//     bcachefs (its colon-joined source is rejected by lsblk).
	//   - Loop-backed live media whose medium is mounted somewhere not listed in
	//     systemMountpoints, or unmounted after the ISO is attached to a loop device.
	//   - Sources that are not block devices: ZFS datasets, overlayfs roots, and
	//     systemd autofs placeholders for an automounted /boot or /efi.
	//   - /dev/root on initramfs-less boots, and any other source lsblk cannot resolve.
	//     These are skipped rather than excluded, so the listing fails open.

	// Find devices mounted on system critical mountpoints
	var systemDevices []string
	for _, m := range mounts {
		if slices.Contains(systemMountpoints, m.mountpoint) && !slices.Contains(systemDevices, m.device) {
			systemDevices = append(systemDevices, m.device)
		}
	}

	var systemDisks []string
	// Go through all systemDevices to find their parent disk
	for _, source := range systemDevices {
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

		// Find the disks in the listing and insert them into systemDisks
		for _, line := range strings.Split(strings.TrimSpace(string(res)), "\n") {
			fields := parseLsblkFields(line)
			if fields["TYPE"] == "disk" && !slices.Contains(systemDisks, fields["KNAME"]) {
				systemDisks = append(systemDisks, fields["KNAME"])
			}
		}
	}
	return systemDisks
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
	systemDisks := findSystemDisks(platform, mounts)

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
		// Exclude any "system" disks from being enumerated
		if slices.Contains(systemDisks, deviceInfo["KNAME"]) {
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

// isPartitionOf reports whether partition is disk itself or one of its partitions,
// e.g. /dev/sda1 for /dev/sda, or /dev/nvme0n1p1 for /dev/nvme0n1. A plain prefix
// check would also match unrelated disks such as /dev/sdab1 for /dev/sda.
func isPartitionOf(partition, disk string) bool {
	suffix, ok := strings.CutPrefix(partition, disk)
	if !ok || disk == "" {
		return false
	} else if suffix == "" {
		return true
	}
	// Device names ending in a digit put a "p" before the partition number.
	if last := disk[len(disk)-1]; '0' <= last && last <= '9' {
		if suffix, ok = strings.CutPrefix(suffix, "p"); !ok {
			return false // Not a partition under disk
		}
	}
	return suffix != "" && strings.Trim(suffix, "0123456789") == ""
}

// UnmountDevice unmounts a block device's partitions before flashing to it.
func UnmountDevice(device string) error {
	return UnmountDeviceWithPlatform(UnixSystemPlatform, device)
}

// UnmountDeviceWithPlatform unmounts a block device's partitions before flashing to it.
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
		if isPartitionOf(mountedDevice, device) {
			if err := platform.SyscallUnmount(mountpoint, 0); err != nil {
				return err
			}
		}
	}

	// TODO: Use lsblk to check if any children are still mounted, and return an error if so.
	return nil
}
