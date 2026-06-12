package checker

const (
	stepParticipants = "participants"
	stepRange        = "range"
	stepDuration     = "duration"
)

type State struct {
	Step   string   `json:"step"`
	Emails []string `json:"emails,omitempty"`
	Cands  []string `json:"cands,omitempty"`
	From   string   `json:"from,omitempty"`
	To     string   `json:"to,omitempty"`
}

type Button struct {
	Text string
	Data string
}

type Reply struct {
	Text     string
	Keyboard [][]Button
	Edit     bool
}
