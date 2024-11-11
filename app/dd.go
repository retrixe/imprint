package app

import (
	"bufio"
	"errors"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ErrNotBlockDevice is returned when the specified device is not a block device.
var ErrNotBlockDevice = errors.New("specified device is not a block device")

// RunDd is a wrapper around the `dd` command. This wrapper behaves
// identically to dd, but accepts stdin input "stop\n".
func RunDd(iff string, of string) {
	cmd := exec.Command("dd", "if="+iff, "of="+of, "status=progress", "bs=1M", "conv=fdatasync")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	go io.Copy(os.Stdout, stdout)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}
	go io.Copy(os.Stderr, stderr)
	err = cmd.Start()
	if err != nil {
		panic(err)
	}
	quit := handleStopInput(func() { cmd.Process.Kill() })
	err = cmd.Wait()
	quit <- true
	if err != nil && cmd.ProcessState.ExitCode() != 0 {
		os.Exit(cmd.ProcessState.ExitCode())
	} else if err != nil {
		panic(err)
	}
}

// FlashFileToBlockDevice is a re-implementation of dd to work cross-platform on Windows as well.
func FlashFileToBlockDevice(iff string, of string) {
	// References to use:
	// https://stackoverflow.com/questions/21032426/low-level-disk-i-o-in-golang
	// https://stackoverflow.com/questions/56512227/how-to-read-and-write-low-level-raw-disk-in-windows-and-go
	quit := handleStopInput(func() { os.Exit(0) })
	src := openFile(iff, os.O_RDONLY, 0, "file")
	defer src.Close()
	dest := openFile(of, os.O_WRONLY|os.O_EXCL, os.ModePerm, "destination")
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
				log.Fatalln("Encountered error while reading file!", err)
			}
		}
		n2, err := dest.Write(buf[:n1])
		if err != nil {
			log.Fatalln("Encountered error while writing to dest!", err)
		} else if n2 != n1 {
			log.Fatalln("Read/write mismatch! Is the dest too small!")
		}
		total += n1
		if len(timer.C) > 0 {
			// There's some minor differences in output with dd, mainly decimal places and kB vs KB.
			timeDifference := time.Now().UnixMilli() - startTime
			print(strconv.Itoa(total) + " bytes " +
				"(" + BytesToString(total, false) + ", " + BytesToString(total, true) + ") copied, " +
				strconv.Itoa(int(timeDifference/1000)) + " s, " +
				BytesToString(total/(int(timeDifference)/1000), false) + "/s\r")
			<-timer.C
			timer.Reset(time.Second)
		}
	}
	// t, _ := io.CopyBuffer(dest, file, buf); total = int(t)
	err := dest.Sync()
	if err != nil {
		log.Fatalln("Failed to sync writes to disk!", err)
	} else {
		timeDifference := float64(time.Now().UnixMilli()-startTime) / 1000
		println(strconv.Itoa(total) + " bytes " +
			"(" + BytesToString(total, false) + ", " + BytesToString(total, true) + ") copied, " +
			strconv.FormatFloat(timeDifference, 'f', 3, 64) + " s, " +
			BytesToString(int(float64(total)/timeDifference), false) + "/s")
	}
	quit <- true
}

// ValidateBlockDeviceContent checks if the block device contents match the given file.
func ValidateBlockDeviceContent(iff string, of string) {
	quit := handleStopInput(func() { os.Exit(0) })
	src := openFile(iff, os.O_RDONLY, 0, "file")
	dest := openFile(of, os.O_RDONLY|os.O_EXCL, os.ModePerm, "destination")
	bs := 4 * 1024 * 1024 // TODO: Allow configurability?
	timer := time.NewTimer(time.Second)
	startTime := time.Now().UnixMilli()
	var total int
	buf1 := make([]byte, bs)
	buf2 := make([]byte, bs)
	for {
		n1, err1 := src.Read(buf1)
		n2, err2 := dest.Read(buf2)
		if err1 == io.EOF && err2 == io.EOF {
			break
		} else if err1 != nil && err1 != io.EOF {
			log.Fatalln("Encountered error while validating device!", err1)
		} else if err2 != nil && err2 != io.EOF {
			log.Fatalln("Encountered error while validating device!", err2)
		} else if n2 != n1 || err1 != nil || err2 != nil || !bytes.Equal(buf1[:n1], buf2[:n2]) {
			log.Fatalln("Read/write mismatch! Validation of image failed. It is unsafe to boot this device.")
		}
		total += n1
		if len(timer.C) > 0 {
			// There's some minor differences in output with dd, mainly decimal places and kB vs KB.
			timeDifference := time.Now().UnixMilli() - startTime
			print(strconv.Itoa(total) + " bytes " +
				"(" + BytesToString(total, false) + ", " + BytesToString(total, true) + ") validated, " +
				strconv.Itoa(int(timeDifference/1000)) + " s, " +
				BytesToString(total/(int(timeDifference)/1000), false) + "/s\r")
			<-timer.C
			timer.Reset(time.Second)
		}
	}
	timeDifference := float64(time.Now().UnixMilli()-startTime) / 1000
	println(strconv.Itoa(total) + " bytes " +
		"(" + BytesToString(total, false) + ", " + BytesToString(total, true) + ") validated, " +
		strconv.FormatFloat(timeDifference, 'f', 3, 64) + " s, " +
		BytesToString(int(float64(total)/timeDifference), false) + "/s")
	quit <- true
}

func openFile(filePath string, flag int, mode fs.FileMode, name string) *os.File {
	path, err := filepath.Abs(filePath)
	if err != nil {
		log.Fatalln("Unable to resolve path to " + name + "!")
	}
	fileStat, err := os.Stat(path)
	if err != nil {
		log.Fatalln("An error occurred while opening "+name+"!", err)
	} else if fileStat.Mode().IsDir() {
		log.Fatalln("The specified " + name + " is a directory!")
	}
	file, err := os.OpenFile(path, flag, mode)
	if err != nil && os.IsNotExist(err) {
		log.Fatalln("This " + name + " does not exist!")
	} else if err != nil {
		log.Fatalln("An error occurred while opening "+name+"!", err)
	}
	return file
}

func handleStopInput(cancel func()) chan bool {
	quit := make(chan bool, 1)
	go (func() {
		reader := bufio.NewReader(os.Stdin)
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
