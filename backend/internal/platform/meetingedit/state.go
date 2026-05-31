package meetingedit

// State is the per-user FSM state (stored in Redis between messages).
type State struct {
	Step          string            `json:"step"` // menu | awaiting
	MeetingID     string            `json:"meeting_id"`
	WorkspaceID   string            `json:"workspace_id"`
	UserID        string            `json:"user_id"`
	AwaitingField string            `json:"awaiting_field,omitempty"`
	Cur           map[string]string `json:"cur"`       // current display values
	Overrides     map[string]string `json:"overrides"` // pending edits
	PartList      []string          `json:"part_list,omitempty"`  // emails shown in the remove list (index → email)
	PartCands     []string          `json:"part_cands,omitempty"` // emails shown as add candidates (index → email)
}

const (
	stepMenu     = "menu"
	stepAwaiting = "awaiting"
)

// Button is one inline-keyboard button.
type Button struct {
	Text string
	Data string
}

// Reply is what the FSM returns for the handler to send. Edit=true means edit
// the existing message instead of sending a new one.
type Reply struct {
	Text     string
	Keyboard [][]Button
	Edit     bool
}
