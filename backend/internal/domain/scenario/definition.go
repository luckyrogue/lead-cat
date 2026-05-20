package scenario

import "encoding/json"

const (
	NodeTriggerCron           = "trigger.cron"
	NodeTriggerManual         = "trigger.manual"
	NodeActionTelegramMessage = "action.telegram.message"
	NodeActionTelegramCat     = "action.telegram.cat_photo"
	NodeActionVCSReport       = "action.vcs.commits_report"
)

type Definition struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

type Node struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Parameters json.RawMessage `json:"parameters"`
}

type Edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

func ParseDefinition(raw json.RawMessage) (Definition, error) {
	var d Definition
	if len(raw) == 0 {
		return d, nil
	}
	err := json.Unmarshal(raw, &d)
	return d, err
}

func (d Definition) Adjacency() map[string][]string {
	out := make(map[string][]string)
	for _, e := range d.Edges {
		out[e.Source] = append(out[e.Source], e.Target)
	}
	return out
}

func (d Definition) NodeByID(id string) (Node, bool) {
	for _, n := range d.Nodes {
		if n.ID == id {
			return n, true
		}
	}
	return Node{}, false
}

func (d Definition) TriggerNodes() []Node {
	var out []Node
	for _, n := range d.Nodes {
		if n.Type == NodeTriggerCron || n.Type == NodeTriggerManual {
			out = append(out, n)
		}
	}
	return out
}
