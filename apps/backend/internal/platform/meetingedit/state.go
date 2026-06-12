package meetingedit

type State struct {
	Step           string            `json:"step"`
	MeetingID      string            `json:"meeting_id"`
	OrganizationID string            `json:"organization_id"`
	UserID         string            `json:"user_id"`
	AwaitingField  string            `json:"awaiting_field,omitempty"`
	Cur            map[string]string `json:"cur"`
	Overrides      map[string]string `json:"overrides"`
	PartList       []string          `json:"part_list,omitempty"`
	PartCands      []string          `json:"part_cands,omitempty"`
	PendingRemove  string            `json:"pending_remove,omitempty"`
	SeriesID       string            `json:"series_id,omitempty"`
	Scope          string            `json:"scope,omitempty"`
}

const (
	stepMenu     = "menu"
	stepAwaiting = "awaiting"
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
