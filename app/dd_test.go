package app

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
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

func GenerateTempFolder(t *testing.T, suffix string) string {
	t.Helper()
	sample, err := os.MkdirTemp(t.TempDir(), t.Name()+"_"+suffix)
	if err != nil {
		t.Fatalf("Failed to create temp folder: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(sample); err != nil {
			t.Fatalf("Failed to remove temp folder: %v", err)
		}
	})
	return sample
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
	sampleDir := GenerateTempFolder(t, "sample")
	dest, _ := GenerateTempFile(t, "dest", false)
	t.Run("FlashFileToBlockDevice fails when either file is folder", func(t *testing.T) {
		var errIsDir *IsDirectoryError
		err := FlashFileToBlockDevice(sampleDir, dest.Name())
		if !errors.As(err, &errIsDir) {
			t.Errorf("Expected IsDirectoryError, got: %v", err)
		}
		err = FlashFileToBlockDevice(sample.Name(), sampleDir)
		if !errors.As(err, &errIsDir) {
			t.Errorf("Expected IsDirectoryError, got: %v", err)
		}
	})
	t.Run("FlashFileToBlockDevice fails when either file does not exist", func(t *testing.T) {
		var errNotExists *NotExistsError
		err := FlashFileToBlockDevice(sample.Name(), filepath.Join(sampleDir, "nonexistent"))
		if !errors.As(err, &errNotExists) {
			t.Errorf("Expected NotExistsError, got: %v", err)
		}
		err = FlashFileToBlockDevice(filepath.Join(sampleDir, "nonexistent"), dest.Name())
		if !errors.As(err, &errNotExists) {
			t.Errorf("Expected NotExistsError, got: %v", err)
		}
	})
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

func TestFormatProgress(t *testing.T) {
	testCases := []struct {
		name       string
		totalBytes int
		delta      int64
		action     string
		floatPrec  bool
		expected   string
	}{
		{
			name:       "zero bytes, zero delta, float",
			totalBytes: 0,
			delta:      0,
			action:     "verified",
			floatPrec:  true,
			expected:   "0 bytes (0 B, 0 B) verified, 0.000 s, 0 B/s",
		},
		{
			name:       "zero bytes, zero delta, int",
			totalBytes: 0,
			delta:      0,
			action:     "verified",
			floatPrec:  false,
			expected:   "0 bytes (0 B, 0 B) verified, 0 s, 0 B/s",
		},
		{
			name:       "zero bytes, short delta, float",
			totalBytes: 0,
			delta:      500,
			action:     "verified",
			floatPrec:  true,
			expected:   "0 bytes (0 B, 0 B) verified, 0.500 s, 0 B/s",
		},
		{
			name:       "zero bytes, short delta, int",
			totalBytes: 0,
			delta:      500,
			action:     "verified",
			floatPrec:  false,
			expected:   "0 bytes (0 B, 0 B) verified, 0 s, 0 B/s",
		},
		{
			name:       "small bytes, simple delta, float",
			totalBytes: 512,
			delta:      2000,
			action:     "copied",
			floatPrec:  true,
			expected:   "512 bytes (512 B, 512 B) copied, 2.000 s, 256 B/s",
		},
		{
			name:       "small bytes, simple delta, int",
			totalBytes: 512,
			delta:      2000,
			action:     "copied",
			floatPrec:  false,
			expected:   "512 bytes (512 B, 512 B) copied, 2 s, 256 B/s",
		},
		{
			name:       "KiB size, fractional delta, float",
			totalBytes: 1536,
			delta:      3500,
			action:     "read",
			floatPrec:  true,
			expected:   "1536 bytes (1.5 KB, 1.5 KiB) read, 3.500 s, 438 B/s",
		},
		{
			name:       "KiB size, fractional delta, int",
			totalBytes: 1536,
			delta:      3500,
			action:     "read",
			floatPrec:  false,
			expected:   "1536 bytes (1.5 KB, 1.5 KiB) read, 3 s, 512 B/s",
		},
		{
			name:       "MiB size, longer delta, float",
			totalBytes: 2 * 1024 * 1024,
			delta:      4876,
			action:     "written",
			floatPrec:  true,
			expected:   "2097152 bytes (2.1 MB, 2.0 MiB) written, 4.876 s, 430.1 KB/s",
		},
		{
			name:       "MiB size, longer delta, int",
			totalBytes: 2 * 1024 * 1024,
			delta:      4876,
			action:     "written",
			floatPrec:  false,
			expected:   "2097152 bytes (2.1 MB, 2.0 MiB) written, 4 s, 524.3 KB/s",
		},
		{
			name:       "Just under 1 second, int",
			totalBytes: 1000,
			delta:      999,
			action:     "transferred",
			floatPrec:  false,
			expected:   "1000 bytes (1.0 KB, 1000 B) transferred, 0 s, 0 B/s",
		},
		{
			name:       "Exactly 1 second, int",
			totalBytes: 1000,
			delta:      1000,
			action:     "transferred",
			floatPrec:  false,
			expected:   "1000 bytes (1.0 KB, 1000 B) transferred, 1 s, 1.0 KB/s",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := FormatProgress(testCase.totalBytes, testCase.delta, testCase.action, testCase.floatPrec)
			if result != testCase.expected {
				t.Errorf("expected %s, got %s", testCase.expected, result)
			}
		})
	}
}
