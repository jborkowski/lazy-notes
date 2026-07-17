package watch

import (
	"sync"
	"time"
)

// Debouncer coalesces rapid trigger calls into a single delayed callback.
type Debouncer struct {
	delay time.Duration
	mu    sync.Mutex
	timer *time.Timer
}

func NewDebouncer(delay time.Duration) *Debouncer {
	if delay <= 0 {
		delay = 1500 * time.Millisecond
	}
	return &Debouncer{delay: delay}
}

// Trigger schedules fn after delay, resetting the timer on each call.
func (d *Debouncer) Trigger(fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(d.delay, fn)
}

// Stop cancels any pending callback.
func (d *Debouncer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
}
