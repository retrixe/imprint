package app

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
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

func TestRunDd(t *testing.T) {
	sample, sampleSum := GenerateTempFile(t, "sample", true)
	dest, _ := GenerateTempFile(t, "dest", false)
	t.Run("dd executes correctly", func(t *testing.T) {
		defer (func() {
			if err := recover(); err != nil {
				t.Fatalf("RunDd failed: %v", err)
			}
		})()
		RunDd(sample.Name(), dest.Name())
		// Check if destination file checksum matches the source file
		destHash := sha256.New()
		destFile, err := os.Open(dest.Name())
		if err != nil {
			t.Fatalf("Failed to open dest file: %v", err)
		}
		defer destFile.Close()
		_, err = io.Copy(destHash, destFile)
		if err != nil {
			t.Fatalf("Failed to copy dest file: %v", err)
		}
		destSum := destHash.Sum(nil)
		if !bytes.Equal(sampleSum, destSum) {
			t.Fatalf("Checksum mismatch: expected %x, got %x", sampleSum, destSum)
		}
	})
}

func TestFlashAndValidation(t *testing.T) {
	sample, sampleSum := GenerateTempFile(t, "sample", true)
	dest, _ := GenerateTempFile(t, "dest", false)
	t.Run("FlashFileToBlockDevice executes correctly", func(t *testing.T) {
		defer (func() {
			if err := recover(); err != nil {
				t.Fatalf("FlashFileToBlockDevice failed: %v", err)
			}
		})()
		FlashFileToBlockDevice(sample.Name(), dest.Name())
		// Check if destination file checksum matches the source file
		destHash := sha256.New()
		destFile, err := os.Open(dest.Name())
		if err != nil {
			t.Fatalf("Failed to open dest file: %v", err)
		}
		defer destFile.Close()
		_, err = io.Copy(destHash, destFile)
		if err != nil {
			t.Fatalf("Failed to copy dest file: %v", err)
		}
		destSum := destHash.Sum(nil)
		if !bytes.Equal(sampleSum, destSum) {
			t.Fatalf("Checksum mismatch: expected %x, got %x", sampleSum, destSum)
		}
	})
	t.Run("ValidateBlockDeviceContent executes correctly", func(t *testing.T) {
		defer (func() {
			if err := recover(); err != nil {
				t.Fatalf("Validation failed: %v", err)
			}
		})()
		ValidateBlockDeviceContent(sample.Name(), dest.Name())
	})
	/* t.Run("ValidateBlockDeviceContent fails if data corrupted", func(t *testing.T) {
		// Simulate data corruption by truncating the destination file
		err := os.Truncate(dest.Name(), 127*1024*1024) // Truncate to 127MB
		if err != nil {
			t.Fatalf("Failed to truncate dest file: %v", err)
		}
		// ValidateBlockDeviceContent should fail
		defer recover()
		ValidateBlockDeviceContent(sample.Name(), dest.Name())
		t.Errorf("Validation should have failed due to data corruption")
	}) */
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
