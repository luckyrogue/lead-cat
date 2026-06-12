package scheduleview

type State struct {
	Step          string   `json:"step"`
	EmployeeEmail string   `json:"employee_email,omitempty"`
	AwaitingKind  string   `json:"awaiting_kind,omitempty"`
	Cands         []string `json:"cands,omitempty"`
}

const (
	stepAwait   = "await"
	awaitSearch = "search"
	awaitDate   = "date"
	awaitRange  = "range"
)

type Button struct {
	Text string
	Data string
}

type Reply struct {
	Text     string
	Keyboard [][]Button
	Edit     bool
}
