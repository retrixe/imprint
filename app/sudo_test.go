package app

import (
	"os"
	"testing"
)

func TestIsElevated(t *testing.T) {
	t.Run("works when elevated on Windows", func(t *testing.T) {
		runtimeGOOS = "windows"
		osOpen = func(name string) (*os.File, error) {
			return &os.File{}, nil
		}
		if !IsElevated() {
			t.Errorf("Expected IsElevated to return true")
		}
	})
	t.Run("works when not elevated on Windows", func(t *testing.T) {
		runtimeGOOS = "windows"
		osOpen = func(name string) (*os.File, error) {
			return nil, os.ErrNotExist
		}
		if IsElevated() {
			t.Errorf("Expected IsElevated to return false")
		}
	})
	t.Run("works when elevated on Linux", func(t *testing.T) {
		runtimeGOOS = "linux"
		osGeteuid = func() int { return 0 }
		if !IsElevated() {
			t.Errorf("Expected IsElevated to return true")
		}
	})
	t.Run("works when not elevated on Linux", func(t *testing.T) {
		runtimeGOOS = "linux"
		osGeteuid = func() int { return -1 }
		if IsElevated() {
			t.Errorf("Expected IsElevated to return false")
		}
	})
}
