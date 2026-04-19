package imaging

import (
	"io/fs"
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

// GetDevices returns the list of USB devices available to read/write from.
func GetDevices(platform Platform) ([]Device, error) {
	res, err := platform.ExecCommandOutput(platform.ExecCommand("diskutil", "info", "-all"))
	if err != nil {
		return nil, err
	}

	availableDisks := strings.Split(string(res), "\n**********\n")
	availableDisks = availableDisks[:len(availableDisks)-1]

	disks := []Device{}

	for _, availableDisk := range availableDisks {
		disk := make(map[string]string)
		lines := strings.Split(availableDisk, "\n")
		for _, rawLine := range lines {
			line := strings.SplitN(strings.TrimSpace(rawLine), ":", 2)
			if len(line) == 2 {
				disk[strings.TrimSpace(line[0])] = strings.TrimSpace(line[1])
			} else {
				disk[strings.TrimSpace(line[0])] = ""
			}
		}
		if disk["Virtual"] != "No" {
			continue
		} else if disk["Whole"] != "Yes" {
			continue
		} else if disk["Device Location"] == "Internal" {
			continue
		}
		splitDiskSize := strings.Split(disk["Disk Size"], " ")
		bytes, _ := strconv.Atoi(splitDiskSize[2][1:])
		device := Device{
			Name:  disk["Device Node"],
			Size:  splitDiskSize[0] + " " + splitDiskSize[1],
			Bytes: bytes,
			Model: disk["Device / Media Name"],
		}
		disks = append(disks, device)
	}

	return disks, nil
}

// UnmountDevice unmounts a block device's partitions before flashing to it.
func UnmountDevice(device string) error {
	return UnmountDeviceWithPlatform(SystemPlatform, device)
}

// UnmountDevice unmounts a block device's partitions before flashing to it.
// It accepts a [UnixPlatform] to allow for testing with a mock platform.
func UnmountDeviceWithPlatform(platform Platform, device string) error {
	// Check if device exists.
	stat, err := platform.OsStat(device)
	if err != nil {
		return err
	} else if stat.Mode().Type()&fs.ModeDevice == 0 {
		return ErrNotBlockDevice
	}
	// Unmount all partitions of disk using `diskutil`.
	// We could go through the mounts manually and call umount, but this seems more reliable.
	_, err = platform.ExecCommandOutput(platform.ExecCommand("diskutil", "unmountDisk", device))
	if err != nil {
		return err
	}
	return nil
}
