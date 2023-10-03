package sse

import "sync"

// bus keeps track of a channel for each HTTP client connection that needs to be
// notified when a relevant event occurs
type bus[T any] struct {
	chs map[chan T]struct{}
	mu  sync.RWMutex
}

// register adds a channel that will be notified when new messages are received
func (b *bus[T]) register(ch chan T) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.chs[ch] = struct{}{}
}

// unregister removes a previous-registered channel, if such a channel is registered
func (b *bus[T]) unregister(ch chan T) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.chs, ch)
}

// clear removes all channels from the bus
func (b *bus[T]) clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.chs = make(map[chan T]struct{})
}

// publish takes a message and fans it out to all currently-registered channels
func (b *bus[T]) publish(message T) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.chs {
		ch <- message
	}
}
