//go:build !darwin && !windows

package app

import (
	"io/fs"
	"os"
	"os/exec"
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
	// TODO: -J = --json (available since Ubuntu 16.04)
	// -d = --nodeps
	// -b = --bytes
	// -o = --output
	res, err := platform.ExecCommandOutput(platform.ExecCommand(
		"lsblk", "-d", "-b", "-o", "KNAME,TYPE,RM,SIZE,MODEL"))
	if err != nil {
		return nil, err
	}

	// TODO: This would be better as an /etc/fstab check, to be honest... This is unreliable.
	bootDevices, err := platform.ExecCommandOutput(platform.ExecCommand("df", "/", "/home"))
	if err != nil {
		return nil, err
	}

	splitBoot := strings.Split(string(bootDevices), "\n")
	rootPart := splitBoot[1]
	homePart := splitBoot[2]

	files := strings.Split(string(res), "\n")
	files = files[:len(files)-1]

	disks := []Device{}

	for _, file := range files {
		disk := strings.Fields(file)
		if disk[1] == "disk" && disk[2] == "1" &&
			// !strings.HasPrefix(disk[0], "zram") && -- zram isn't removable anyway
			!strings.HasPrefix(rootPart, "/dev/"+disk[0]) &&
			!strings.HasPrefix(homePart, "/dev/"+disk[0]) {
			bytes, _ := strconv.Atoi(disk[3])
			device := Device{
				Name:  "/dev/" + disk[0],
				Size:  BytesToString(bytes, false),
				Bytes: bytes,
			}

			if len(disk) >= 4 {
				device.Model = strings.TrimSpace(strings.Join(disk[4:], " "))
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
	// Discover mounted device partitions.
	mounts, err := exec.Command("mount").Output()
	if err != nil {
		return err
	}
	// Unmount device partitions.
	for _, mount := range strings.Split(string(mounts), "\n") {
		if strings.HasPrefix(mount, device) {
			partition := strings.Fields(mount)[0]
			err = exec.Command("umount", partition).Run()
			if err != nil {
				return err
			}
		}
	}
	return nil
}
