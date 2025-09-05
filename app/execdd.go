package app

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// DdProgress is a struct containing progress of the dd operation.
type DdProgress struct {
	Bytes int
	Speed string
	Phase string
	Error error
}

// DdError is a struct containing dd errors.
type DdError struct {
	Message string
	Err     error
}

func (err *DdError) Error() string {
	output := strings.TrimSpace(err.Message)
	if _, ok := err.Err.(*exec.ExitError); ok && output != "" {
		return output
	} else if output == "" {
		return err.Err.Error()
	}
	return err.Err.Error() + ": " + output
}

// CopyConvert executes the `dd` Unix utility and provides its output.
//
// Technically, this isn't true anymore, it executes imprint itself
// with some special parameters as admin. The new imprint process
// wraps `dd` and accepts "stop\n" stdin to terminate dd. This is
// because killing the process doesn't work with pkexec/osascript,
// and this approach enables us to reimplement dd fully.
func CopyConvert(iff string, of string) (chan DdProgress, io.WriteCloser, error) {
	channel := make(chan DdProgress)
	executable, err := os.Executable()
	if err != nil {
		return nil, nil, err
	}
	ddFlag := ""
	if os.Getenv("__USE_SYSTEM_DD") == "true" {
		ddFlag = "--use-system-dd"
	}
	cmd, err := ElevatedCommand(SystemPlatform, executable, "flash", iff, of, ddFlag)
	if err != nil {
		return nil, nil, err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	output, input := io.Pipe()
	cmd.Stderr = input
	cmd.Stdout = input
	err = cmd.Start()
	if err != nil {
		return nil, nil, err
	}
	// Wait for command to exit.
	lastLine := ""
	channelClosed := false
	var mutex sync.Mutex
	go (func() {
		defer input.Close()
		err := cmd.Wait()
		if err != nil {
			channel <- DdProgress{
				Error: &DdError{Message: lastLine, Err: err},
			}
		}
		mutex.Lock()
		defer mutex.Unlock()
		channelClosed = true
		close(channel)
	})()
	// Read the output line by line.
	go (func() {
		phase := "Phase Unknown"
		scanner := bufio.NewScanner(output)
		scanner.Split(ScanCRLFLines)
		for scanner.Scan() {
			text := scanner.Text()
			println(text)
			lastLine = text
			firstSpace := strings.Index(text, " ")
			if strings.HasPrefix(text, "[flash] Phase") {
				phase = text[firstSpace+1:]
				channel <- DdProgress{
					Bytes: 0,
					Speed: "0 MB/s",
					Phase: phase,
				}
			} else if firstSpace != -1 && strings.HasPrefix(text[firstSpace+1:], "bytes (") {
				// TODO: Probably handle error, but we can't tell full dd behavior without seeing the code.
				// Well, custom dd is the default now anyways.
				bytes, _ := strconv.Atoi(text[:firstSpace])
				split := strings.Split(text, ", ")
				mutex.Lock()
				if channelClosed {
					return // We don't need to unlock as no deadlock is caused here.
				}
				channel <- DdProgress{
					Bytes: bytes,
					Speed: split[len(split)-1],
					Phase: phase,
				}
				mutex.Unlock()
			}
		}
	})()
	return channel, stdin, nil
}

// dropCRLF drops a terminal \r or \n from the data.
func dropCRLF(data []byte) []byte {
	if len(data) > 0 && (data[len(data)-1] == '\r' || data[len(data)-1] == '\n') {
		return data[0 : len(data)-1]
	}
	return data
}

// ScanCRLFLines is a split function for a Scanner that returns each line of
// text, stripped of any trailing end-of-line marker. The returned line may
// be empty. The end-of-line marker is one carriage return or one mandatory
// newline. In regular expression notation, it is `\r|\n`. The last
// non-empty line of input will be returned even if it has no newline.
func ScanCRLFLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		// We have a full newline-terminated line.
		return i + 1, dropCRLF(data[0:i]), nil
	} else if i := bytes.IndexByte(data, '\r'); i >= 0 {
		// We have a full carriage return-terminated line.
		return i + 1, dropCRLF(data[0:i]), nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), dropCRLF(data), nil
	}
	// Request more data.
	return 0, nil, nil
}
