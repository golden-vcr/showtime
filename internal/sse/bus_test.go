package sse

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_bus(t *testing.T) {
	xs := make([]int, 0)
	ys := make([]int, 0)
	zs := make([]int, 0)

	xsChan := make(chan int, 8)
	ysChan := make(chan int, 8)
	zsChan := make(chan int, 8)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for {
			select {
			case <-time.After(time.Millisecond):
				wg.Done()
				return
			case x := <-xsChan:
				xs = append(xs, x)
			case y := <-ysChan:
				ys = append(ys, y)
			case z := <-zsChan:
				zs = append(zs, z)
			}
		}
	}()

	b := bus[int]{
		chs: make(map[chan int]struct{}),
	}
	b.publish(100)
	b.register(xsChan)
	b.publish(200)
	b.register(ysChan)
	b.publish(300)
	b.register(zsChan)
	b.unregister(xsChan)
	b.unregister(xsChan) // no-op
	b.publish(400)
	b.clear()
	b.publish(500)
	wg.Wait()

	assert.Equal(t, []int{200, 300}, xs)
	assert.Equal(t, []int{300, 400}, ys)
	assert.Equal(t, []int{400}, zs)
}
