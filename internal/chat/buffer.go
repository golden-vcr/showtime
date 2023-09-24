package chat

// bufferedMessage records the user ID associated with a recent message ID, so we we
// can identify all relevant message IDs given an offending user ID
type bufferedMessage struct {
	userId    string
	messageId string
}

// messageBuffer is a fixed-size ring buffer recording the user IDs associated with the
// N most recent messages
type messageBuffer struct {
	messages  []bufferedMessage
	capacity  int
	size      int
	headIndex int
}

// newMessageBuffer initializes an empty messageBuffer that will hold userId:messageId
// pairs up to the given capacity
func newMessageBuffer(capacity int) *messageBuffer {
	return &messageBuffer{
		messages:  make([]bufferedMessage, capacity, capacity),
		capacity:  capacity,
		size:      0,
		headIndex: 0,
	}
}

// add registers a new item recording the fact that the given user sent a message with
// the given ID, potentially ejecting the oldest item from the buffer in the process
func (b *messageBuffer) add(userId string, messageId string) {
	b.messages[b.headIndex].userId = userId
	b.messages[b.headIndex].messageId = messageId
	b.headIndex = (b.headIndex + 1) % b.capacity
	b.size = min(b.size+1, b.capacity)
}

// resolveMessageIds searches the buffer for all recent messages sent by the user with
// the given ID, and returns a list of all message IDs associated with that user
func (b *messageBuffer) resolveMessageIds(userId string) []string {
	results := make([]string, 0, 8)
	for i := 0; i < b.size; i++ {
		if b.messages[i].userId == userId {
			results = append(results, b.messages[i].messageId)
		}
	}
	return results
}
