package imaging

import "errors"

// ErrNotBlockDevice is returned when the specified device is not a block device.
var ErrNotBlockDevice = errors.New("specified device is not a block device")

// Device is a struct representing a block device.
type Device struct {
	Name  string
	Model string
	Size  string
	Bytes int
}
