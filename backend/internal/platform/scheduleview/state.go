package scheduleview

// State is the per-user FSM state (stored in Redis between messages).
type State struct {
	Step          string   `json:"step"`
	EmployeeEmail string   `json:"employee_email,omitempty"`
	AwaitingKind  string   `json:"awaiting_kind,omitempty"` // search | date | range
	Cands         []string `json:"cands,omitempty"`         // candidate emails (index → email)
}

const (
	stepAwait   = "await"
	awaitSearch = "search"
	awaitDate   = "date"
	awaitRange  = "range"
)

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
