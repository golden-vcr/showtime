package chat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessageBuffer(t *testing.T) {
	b := newMessageBuffer(4)

	assert.Len(t, b.messages, 4)
	assert.Equal(t, 4, b.capacity)
	assert.Equal(t, 0, b.size)
	assert.Equal(t, 0, b.headIndex)

	b.add("alice", "1")
	b.add("bob", "2")
	b.add("alice", "3")

	assert.Len(t, b.messages, 4)
	assert.Equal(t, 4, b.capacity)
	assert.Equal(t, 3, b.size)
	assert.Equal(t, 3, b.headIndex)

	assert.ElementsMatch(t, b.resolveMessageIds("alice"), []string{"1", "3"})
	assert.ElementsMatch(t, b.resolveMessageIds("bob"), []string{"2"})
	assert.Len(t, b.resolveMessageIds("charlie"), 0)

	b.add("charlie", "4")
	b.add("bob", "5")
	b.add("alice", "6")

	assert.Len(t, b.messages, 4)
	assert.Equal(t, 4, b.capacity)
	assert.Equal(t, 4, b.size)
	assert.Equal(t, 2, b.headIndex)

	assert.ElementsMatch(t, b.resolveMessageIds("alice"), []string{"3", "6"})
	assert.ElementsMatch(t, b.resolveMessageIds("bob"), []string{"5"})
	assert.ElementsMatch(t, b.resolveMessageIds("charlie"), []string{"4"})
}
