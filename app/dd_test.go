package app

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func GenerateTempFile(t *testing.T, suffix string, prefill bool) (*os.File, []byte) {
	t.Helper()
	sample, err := os.CreateTemp(t.TempDir(), t.Name()+"_"+suffix)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Remove(sample.Name()); err != nil {
			t.Fatalf("Failed to remove temp file: %v", err)
		}
	})
	// Generate random data, checksum it and write it to the file
	if !prefill {
		return sample, nil
	}
	sampleHash := sha256.New()
	for i := 0; i < 128; i++ { // 128MB
		data := make([]byte, 1024*1024) // 1MB
		_, _ = rand.Read(data)
		_, err = sampleHash.Write(data)
		if err != nil {
			t.Fatalf("Failed to write to hash: %v", err)
		}
		_, err := sample.Write(data)
		if err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
	}
	if err = sample.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}
	sampleSum := sampleHash.Sum(nil)
	return sample, sampleSum
}

func ChecksumFile(t *testing.T, file string) ([]byte, error) {
	t.Helper()
	fileHash := sha256.New()
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	_, err = io.Copy(fileHash, f)
	if err != nil {
		return nil, err
	}
	return fileHash.Sum(nil), nil
}

func TestRunDd(t *testing.T) {
	sample, sampleSum := GenerateTempFile(t, "sample", true)
	dest, _ := GenerateTempFile(t, "dest", false)
	t.Run("dd executes correctly", func(t *testing.T) {
		err := RunDd(sample.Name(), dest.Name())
		if err != nil {
			t.Errorf("RunDd failed: %v", err)
		} else if checksum, err := ChecksumFile(t, dest.Name()); err != nil {
			t.Errorf("Failed to generate checksum for dest: %v", err)
		} else if !bytes.Equal(sampleSum, checksum) {
			t.Errorf("Checksum mismatch: expected %x, got %x", sampleSum, checksum)
		}
	})
}

func TestFlashAndValidation(t *testing.T) {
	sample, sampleSum := GenerateTempFile(t, "sample", true)
	dest, _ := GenerateTempFile(t, "dest", false)
	t.Run("FlashFileToBlockDevice executes correctly", func(t *testing.T) {
		err := FlashFileToBlockDevice(sample.Name(), dest.Name())
		if err != nil {
			t.Errorf("FlashFileToBlockDevice failed: %v", err)
		} else if checksum, err := ChecksumFile(t, dest.Name()); err != nil {
			t.Errorf("Failed to generate checksum for dest: %v", err)
		} else if !bytes.Equal(sampleSum, checksum) {
			t.Errorf("Checksum mismatch: expected %x, got %x", sampleSum, checksum)
		}
	})
	t.Run("ValidateBlockDeviceContent executes correctly", func(t *testing.T) {
		defer (func() {
			if err := recover(); err != nil {
				t.Errorf("Validation failed: %v", err)
			}
		})()
		err := ValidateBlockDeviceContent(sample.Name(), dest.Name())
		if err != nil {
			t.Errorf("Validation failed: %v", err)
		}
	})
	t.Run("ValidateBlockDeviceContent fails if data corrupted", func(t *testing.T) {
		// Simulate data corruption by truncating the destination file
		err := os.Truncate(dest.Name(), 127*1024*1024) // Truncate to 127MB
		if err != nil {
			t.Errorf("Failed to truncate dest file: %v", err)
		}
		// ValidateBlockDeviceContent should fail
		defer recover()
		err = ValidateBlockDeviceContent(sample.Name(), dest.Name())
		if err == nil {
			t.Errorf("Validation should have failed due to data corruption")
		} else if !errors.Is(err, ErrDeviceValidationFailed) {
			t.Errorf("Unexpected validation error: %v", err)
		}
	})
}

func TestHandleStopInput(t *testing.T) {
	t.Run("quit handling stop input with channel message", func(t *testing.T) {
		t.Parallel()
		quit := handleStopInput(io.NopCloser(bytes.NewBufferString("\n")), func() {
			t.Errorf("Cancel function should not be called")
		})
		quit <- true
		<-time.After(time.Second)
	})
	t.Run("quit handling stop input by closing channel", func(t *testing.T) {
		t.Parallel()
		quit := handleStopInput(io.NopCloser(bytes.NewBufferString("\n")), func() {
			t.Errorf("Cancel function should not be called")
		})
		close(quit)
		<-time.After(time.Second)
	})
	t.Run("cancel handling stop input", func(t *testing.T) {
		t.Parallel()
		var called atomic.Bool
		handleStopInput(io.NopCloser(bytes.NewBufferString("stop\n")), func() {
			called.Store(true)
		})
		<-time.After(time.Second)
		if !called.Load() {
			t.Errorf("Cancel function should have been called")
		}
	})
}
