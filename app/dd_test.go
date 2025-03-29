package app

import (
	"bytes"
	"io"
	"sync/atomic"
	"testing"
	"time"
)

func TestHandleStopInput(t *testing.T) {
	t.Run("quit handling stop input with channel message", func(t *testing.T) {
		t.Parallel()
		quit := handleStopInput(func() {
			t.Errorf("Cancel function should not be called")
		})
		quit <- true
		<-time.After(time.Second)
	})
	t.Run("quit handling stop input by closing channel", func(t *testing.T) {
		t.Parallel()
		quit := handleStopInput(func() {
			t.Errorf("Cancel function should not be called")
		})
		close(quit)
		<-time.After(time.Second)
	})
	t.Run("cancel handling stop input", func(t *testing.T) {
		var called atomic.Bool
		handleStopInput(func() {
			called.Store(true)
		})
		stdin = io.NopCloser(bytes.NewBufferString("stop\n"))
		<-time.After(time.Second)
		if !called.Load() {
			t.Errorf("Cancel function should have been called")
		}
	})
}
