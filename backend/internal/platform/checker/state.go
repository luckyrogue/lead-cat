// Package checker drives the /checker common-free-time bot flow (§4.8).
package checker

// Steps for the checker FSM.
const (
	stepParticipants = "participants"
	stepRange        = "range"
	stepDuration     = "duration"
)

// State is the persisted /checker conversation state.
type State struct {
	Step   string   `json:"step"`
	Emails []string `json:"emails,omitempty"` // chosen participant emails
	Cands  []string `json:"cands,omitempty"`  // last search candidates (index → email)
	From   string   `json:"from,omitempty"`   // YYYY-MM-DD (inclusive)
	To     string   `json:"to,omitempty"`     // YYYY-MM-DD (inclusive)
}

// Button is one inline-keyboard button.
type Button struct {
	Text string
	Data string
}

// Reply is what the FSM returns for the handler to send.
type Reply struct {
	Text     string
	Keyboard [][]Button
	Edit     bool
}
