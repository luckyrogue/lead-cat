package scenario

import (
	"encoding/json"
	"fmt"
)

var knownNodeTypes = map[string]bool{
	NodeTriggerCron:           true,
	NodeTriggerManual:         true,
	NodeActionTelegramMessage: true,
	NodeActionTelegramCat:     true,
	NodeActionVCSReport:       true,
}

func ValidateDefinition(raw json.RawMessage) error {
	d, err := ParseDefinition(raw)
	if err != nil {
		return err
	}
	if len(d.Nodes) == 0 {
		return fmt.Errorf("scenario must have at least one node")
	}
	ids := make(map[string]bool)
	cronTriggers := 0
	for _, n := range d.Nodes {
		if n.ID == "" {
			return fmt.Errorf("node id required")
		}
		if ids[n.ID] {
			return fmt.Errorf("duplicate node id %s", n.ID)
		}
		ids[n.ID] = true
		if !knownNodeTypes[n.Type] {
			return fmt.Errorf("unknown node type %s", n.Type)
		}
		if n.Type == NodeTriggerCron {
			cronTriggers++
		}
	}
	for _, e := range d.Edges {
		if !ids[e.Source] || !ids[e.Target] {
			return fmt.Errorf("edge references unknown node")
		}
	}
	if hasCycle(d) {
		return fmt.Errorf("scenario graph must not contain cycles")
	}
	return nil
}

func hasCycle(d Definition) bool {
	adj := d.Adjacency()
	visiting := make(map[string]bool)
	done := make(map[string]bool)
	var dfs func(string) bool
	dfs = func(id string) bool {
		if done[id] {
			return false
		}
		if visiting[id] {
			return true
		}
		visiting[id] = true
		for _, next := range adj[id] {
			if dfs(next) {
				return true
			}
		}
		delete(visiting, id)
		done[id] = true
		return false
	}
	for _, n := range d.Nodes {
		if dfs(n.ID) {
			return true
		}
	}
	return false
}
