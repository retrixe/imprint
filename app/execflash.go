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

	"github.com/retrixe/imprint/imaging"
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
// Technically, this isn't true anymore, it executes Imprint itself
// with some special parameters as admin. The new Imprint process
// wraps `dd` and accepts "stop\n" stdin to terminate dd. This is
// because killing the process doesn't work with pkexec/osascript,
// and this approach enables us to reimplement dd fully.
func CopyConvert(iff string, of string) (chan DdProgress, io.WriteCloser, error) {
	// FIXME: Write unit tests
	channel := make(chan DdProgress)
	executable, err := os.Executable()
	if err != nil {
		return nil, nil, err
	}
	ddFlag := "--use-system-dd=" + strconv.FormatBool(os.Getenv("__USE_SYSTEM_DD") == "true")
	cmd, err := ElevatedCommand(imaging.SystemPlatform, executable, "flash", ddFlag, iff, of)
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
		scanner.Split(ScanCROrLFLines)
		for scanner.Scan() {
			text := scanner.Text()
			println(text)
			lastLine = text
			before, after, ok := strings.Cut(text, " ")
			if strings.HasPrefix(text, "[flash] Phase") {
				phase = after
				channel <- DdProgress{
					Bytes: 0,
					Speed: "0 MB/s",
					Phase: phase,
				}
			} else if ok && strings.HasPrefix(after, "bytes (") {
				// TODO: Probably handle error, but we can't tell full dd behavior without seeing the code.
				// Well, custom dd is the default now anyways.
				bytes, _ := strconv.Atoi(before)
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

// dropCR drops a terminal \r from the data.
func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}

// ScanCROrLFLines is a split function for a Scanner that returns each line of
// text, stripped of any trailing end-of-line marker. The returned line may
// be empty. The end-of-line marker is one carriage return or one mandatory
// newline. In regular expression notation, it is `\r|\n`. The last
// non-empty line of input will be returned even if it has no newline.
//
// Modified from [bufio.ScanLines] to support \r as a line terminator on its own,
// in addition to \n and \r\n.
func ScanCROrLFLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	i := bytes.IndexByte(data, '\r')
	j := bytes.IndexByte(data, '\n')
	if j >= 0 && (i < 0 || j < i || j == i+1) { // No \r, or \n comes before \r, or \r\n sequence.
		// We have a full newline-terminated line.
		return j + 1, dropCR(data[0:j]), nil
	} else if i >= 0 {
		// We have a full carriage return-terminated line.
		return i + 1, data[0:i], nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}
