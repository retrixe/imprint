package app

import (
	"io/fs"
	"os"
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

const wmicArgs = "diskdrive get deviceid, mediatype, model, caption, size"

// GetDevices returns the list of USB devices available to read/write from.
func GetDevices(platform Platform) ([]Device, error) {
	res, err := platform.ExecCommandOutput(platform.ExecCommand("wmic", strings.Split(wmicArgs, " ")...))
	if err != nil {
		return nil, err
	}

	availableDisks := strings.Split(strings.TrimSpace(string(res)), "\n")

	disks := []Device{}

	for _, availableDisk := range availableDisks {
		split := strings.Split(availableDisk, "  ")
		disk := []string{}
		for _, element := range split {
			trimmed := strings.TrimSpace(element)
			if trimmed != "" {
				disk = append(disk, trimmed)
			}
		}
		indexOffset := 0
		if len(disk) == 5 {
			indexOffset = 1
		}
		if disk[1+indexOffset] == "Removable Media" {
			bytes, _ := strconv.Atoi(disk[3+indexOffset])
			device := Device{
				Name:  disk[0+indexOffset],
				Size:  BytesToString(bytes, false),
				Bytes: bytes,
			}
			if indexOffset == 1 {
				device.Model = disk[0]
			}
			disks = append(disks, device)
		}
	}

	return disks, nil
}

// UnmountDevice unmounts a block device's partitons before flashing to it.
func UnmountDevice(device string) error {
	// Check if device is mounted.
	stat, err := os.Stat(device)
	if err != nil {
		return err
	} else if stat.Mode().Type()&fs.ModeDevice == 0 {
		return ErrNotBlockDevice
	}
	// FIXME: Discover device partitions and recursively unmount them.
	return nil
}
