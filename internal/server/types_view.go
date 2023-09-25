package server

type State struct {
	// TapeId is the ID of the tape that's currently selected, or an empty string if no
	// tape is active
	TapeId string `json:"tapeId"`
}
