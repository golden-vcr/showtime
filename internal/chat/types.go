package chat

// LogEventType is an abstraction on top of IRC messages, presenting the frontend with a
// simplified set of chat events that are germane to rendering the chat log
type LogEventType string

const (
	// LogEventTypeMessage indicates that a new chat line should be displayed
	LogEventTypeMessage LogEventType = "message"
	// LogEventTypeDeletion indicates that one or more previous lines should be deleted
	LogEventTypeDeletion LogEventType = "deletion"
	// LogEventTypeClear indicates that all lines should be deleted from the log
	LogEventTypeClear LogEventType = "clear"
)

// LogEvent is an event in Twitch chat that the chat log UI needs to know about
type LogEvent struct {
	Type     LogEventType `json:"type"`
	Message  *LogMessage  `json:"message,omitempty"`
	Deletion *LogDeletion `json:"deletion,omitempty"`
}

// LogMessage is the payload for an event with type 'message'
type LogMessage struct {
	ID       string         `json:"id"`
	Username string         `json:"username"`
	Color    string         `json:"color"`
	Text     string         `json:"text"`
	Emotes   []EmoteDetails `json:"emotes"`
}

type EmoteDetails struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}

// LogDeletion is the payload for an event with type 'deletion'
type LogDeletion struct {
	MessageIDs []string `json:"messageIds"`
}
