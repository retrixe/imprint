package app

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ErrDeviceValidationFailed is returned when the image on the device is corrupt
var ErrDeviceValidationFailed = errors.New(
	"read/write mismatch, validation of image on device failed")

// ErrReadWriteMismatch is returned if written bytes are not the same as bytes as read.
// Typically caused by target device being too small.
var ErrReadWriteMismatch = errors.New("mismatch between bytes read and written")

// IsDirectoryError is returned if a path that was passed is a directory, but a file was expected.
type IsDirectoryError struct{ Name string }

func (e *IsDirectoryError) Error() string {
	return fmt.Sprintf("the specified file %s is a directory!", e.Name)
}

// NotExistsError is returned if a path that was passed does not point to a valid file or folder.
type NotExistsError struct{ Name string }

func (e *NotExistsError) Error() string {
	return fmt.Sprintf("the specified file %s does not exist!", e.Name)
}

// FormatProgress formats the progress of a dd-like operation.
// There's some minor differences in output with dd, mainly decimal places and kB vs KB.
func FormatProgress(total int, delta int64, action string, floatPrec bool) string {
	str := strconv.Itoa(total) + " bytes " +
		"(" + BytesToString(total, false) + ", " + BytesToString(total, true) + ") " + action + ", "
	if floatPrec {
		timeDifference := float64(delta) / 1000
		speed := 0
		if timeDifference > 0 {
			speed = int(float64(total) / timeDifference)
		}
		str += strconv.FormatFloat(timeDifference, 'f', 3, 64) + " s, " + BytesToString(speed, false) + "/s"
	} else {
		timeDifference := int(delta) / 1000
		speed := 0
		if timeDifference > 0 {
			speed = total / timeDifference
		}
		str += strconv.Itoa(timeDifference) + " s, " + BytesToString(speed, false) + "/s"
	}
	return str
}

// RunDd is a wrapper around the `dd` command. This wrapper behaves
// identically to dd, but accepts stdin input "stop\n".
func RunDd(iff string, of string) error {
	cmd := exec.Command("dd", "if="+iff, "of="+of, "status=progress", "bs=1M", "conv=fdatasync")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	go io.Copy(os.Stdout, stdout)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	go io.Copy(os.Stderr, stderr)
	err = cmd.Start()
	if err != nil {
		return err
	}
	quit := handleStopInput(os.Stdin, func() { cmd.Process.Kill() })
	err = cmd.Wait()
	quit <- true
	if err != nil && cmd.ProcessState.ExitCode() != 0 {
		os.Exit(cmd.ProcessState.ExitCode())
	}
	return err
}

// FlashFileToBlockDevice is a re-implementation of dd to work cross-platform on Windows as well.
func FlashFileToBlockDevice(iff string, of string) error {
	// References to use:
	// https://stackoverflow.com/questions/21032426/low-level-disk-i-o-in-golang
	// https://stackoverflow.com/questions/56512227/how-to-read-and-write-low-level-raw-disk-in-windows-and-go
	quit := handleStopInput(os.Stdin, func() { os.Exit(0) })
	src, err := openFile(iff, os.O_RDONLY, 0, "file")
	if err != nil {
		return err
	}
	defer src.Close()
	dest, err := openFile(of, os.O_WRONLY|os.O_EXCL, os.ModePerm, "destination")
	if err != nil {
		return err
	}
	defer dest.Close()
	bs := 4 * 1024 * 1024 // TODO: Allow configurability?
	timer := time.NewTimer(time.Second)
	startTime := time.Now().UnixMilli()
	var total int
	buf := make([]byte, bs)
	for {
		n1, err := src.Read(buf)
		if err != nil {
			if io.EOF == err {
				break
			} else {
				return fmt.Errorf("encountered error while reading file! %w", err)
			}
		}
		n2, err := dest.Write(buf[:n1])
		if err != nil {
			return fmt.Errorf("encountered error while writing to dest! %w", err)
		} else if n2 != n1 {
			return ErrReadWriteMismatch
		}
		total += n1
		if len(timer.C) > 0 {
			print(FormatProgress(total, time.Now().UnixMilli()-startTime, "copied", false) + "\r")
			<-timer.C
			timer.Reset(time.Second)
		}
	}
	// t, _ := io.CopyBuffer(dest, file, buf); total = int(t)
	err = dest.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync writes to disk! %w", err)
	} else {
		println(FormatProgress(total, time.Now().UnixMilli()-startTime, "copied", true))
	}
	quit <- true
	return nil
}

// ValidateBlockDeviceContent checks if the block device contents match the given file.
func ValidateBlockDeviceContent(iff string, of string) error {
	quit := handleStopInput(os.Stdin, func() { os.Exit(0) })
	src, err := openFile(iff, os.O_RDONLY, 0, "file")
	if err != nil {
		return err
	}
	defer src.Close()
	dest, err := openFile(of, os.O_RDONLY|os.O_EXCL, os.ModePerm, "destination")
	if err != nil {
		return err
	}
	bs := 4 * 1024 * 1024 // TODO: Allow configurability?
	timer := time.NewTimer(time.Second)
	startTime := time.Now().UnixMilli()
	var total int
	buf1 := make([]byte, bs)
	buf2 := make([]byte, bs)
	for {
		n1, err1 := src.Read(buf1)
		n2, err2 := dest.Read(buf2)
		if err1 == io.EOF {
			break
		} else if err1 != nil {
			return fmt.Errorf("encountered error while validating device! %w", err1)
		} else if err2 != nil {
			return fmt.Errorf("encountered error while validating device! %w", err2)
		} else if n2 < n1 || !bytes.Equal(buf1[:n1], buf2[:n1]) {
			return ErrDeviceValidationFailed
		}
		total += n1
		if len(timer.C) > 0 {
			print(FormatProgress(total, time.Now().UnixMilli()-startTime, "validated", false) + "\r")
			<-timer.C
			timer.Reset(time.Second)
		}
	}
	println(FormatProgress(total, time.Now().UnixMilli()-startTime, "validated", true))
	quit <- true
	return nil
}

func openFile(filePath string, flag int, mode fs.FileMode, name string) (*os.File, error) {
	path, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve path to %s! %w", name, err)
	}
	fileStat, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return nil, &NotExistsError{Name: name}
	} else if err != nil {
		return nil, fmt.Errorf("an error occurred while opening %s! %w", name, err)
	} else if fileStat.Mode().IsDir() {
		return nil, &IsDirectoryError{Name: name}
	}
	file, err := os.OpenFile(path, flag, mode)
	if err != nil {
		return nil, fmt.Errorf("an error occurred while opening %s! %w", name, err)
	}
	return file, nil
}

func handleStopInput(input io.Reader, cancel func()) chan bool {
	quit := make(chan bool, 1)
	go (func() {
		reader := bufio.NewReader(input)
		for {
			select {
			case <-quit:
				return
			default:
				text, err := reader.ReadString('\n')
				if strings.TrimSpace(text) == "stop" {
					cancel()
				} else if err != nil {
					return
				}
			}
		}
	})()
	return quit
}
