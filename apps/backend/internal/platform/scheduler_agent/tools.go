package scheduler_agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
)

var almatyLoc = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		return time.FixedZone("Almaty", 5*60*60)
	}
	return loc
}()

type Backend interface {
	SearchEmployeesGlobal(ctx context.Context, query string) ([]model.Employee, error)
	FreeSlots(ctx context.Context, emails []string, from, to time.Time, durMins int) ([]application.FreeSlot, error)
	MeetingConflicts(ctx context.Context, emails []string, start, end time.Time, exclude uuid.UUID) ([]application.Conflict, error)
}

func ToolSpecs() []application.AgentTool {
	return []application.AgentTool{
		{
			Name:        "search_people",
			Description: "Find colleagues by name or email substring. Call this first to resolve any person the user names into an email address before scheduling.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string", "description": "Name or email fragment to search for."},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "find_free_slots",
			Description: "Find common free working-hours slots for the given participant emails across a date range. Use this to answer 'when is everyone free' questions.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"emails":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Participant emails (resolve names via search_people first)."},
					"from":          map[string]any{"type": "string", "description": "Inclusive start date, format YYYY-MM-DD."},
					"to":            map[string]any{"type": "string", "description": "Inclusive end date, format YYYY-MM-DD."},
					"duration_mins": map[string]any{"type": "integer", "description": "Required free-block length in minutes."},
				},
				"required": []string{"emails", "from", "to", "duration_mins"},
			},
		},
		{
			Name:        "check_conflicts",
			Description: "Check whether any of the given participants already have a meeting overlapping a specific time window. Call this before telling the user a time is free to book.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"emails": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Participant emails."},
					"start":  map[string]any{"type": "string", "description": "Window start, format 'YYYY-MM-DD HH:MM' (Almaty time)."},
					"end":    map[string]any{"type": "string", "description": "Window end, format 'YYYY-MM-DD HH:MM' (Almaty time)."},
				},
				"required": []string{"emails", "start", "end"},
			},
		},
	}
}

func Dispatch(ctx context.Context, be Backend, name string, args json.RawMessage) (string, error) {
	switch name {
	case "search_people":
		return dispatchSearch(ctx, be, args)
	case "find_free_slots":
		return dispatchFreeSlots(ctx, be, args)
	case "check_conflicts":
		return dispatchConflicts(ctx, be, args)
	default:
		return "", fmt.Errorf("unknown tool %q", name)
	}
}

func dispatchSearch(ctx context.Context, be Backend, args json.RawMessage) (string, error) {
	var in struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("bad arguments: %w", err)
	}
	emps, err := be.SearchEmployeesGlobal(ctx, in.Query)
	if err != nil {
		return "", err
	}
	seen := map[string]bool{}
	var b strings.Builder
	n := 0
	for _, e := range emps {
		if e.Email == "" || seen[e.Email] {
			continue
		}
		seen[e.Email] = true
		fmt.Fprintf(&b, "- %s <%s>\n", e.FullName, e.Email)
		n++
	}
	if n == 0 {
		return fmt.Sprintf("No people found matching %q.", in.Query), nil
	}
	return b.String(), nil
}

func dispatchFreeSlots(ctx context.Context, be Backend, args json.RawMessage) (string, error) {
	var in struct {
		Emails       []string `json:"emails"`
		From         string   `json:"from"`
		To           string   `json:"to"`
		DurationMins int      `json:"duration_mins"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("bad arguments: %w", err)
	}
	from, err := time.ParseInLocation("2006-01-02", in.From, almatyLoc)
	if err != nil {
		return "", fmt.Errorf("bad 'from' date (want YYYY-MM-DD): %w", err)
	}
	to, err := time.ParseInLocation("2006-01-02", in.To, almatyLoc)
	if err != nil {
		return "", fmt.Errorf("bad 'to' date (want YYYY-MM-DD): %w", err)
	}
	
	slots, err := be.FreeSlots(ctx, in.Emails, from, to.AddDate(0, 0, 1), in.DurationMins)
	if err != nil {
		return "", err
	}
	if len(slots) == 0 {
		return "No common free slots in that range. Suggest a wider range, a shorter duration, or fewer participants.", nil
	}
	var b strings.Builder
	for _, sl := range slots {
		fmt.Fprintf(&b, "- %s %s–%s (%d min free)\n",
			sl.Day.In(almatyLoc).Format("2006-01-02 Mon"),
			sl.Start.In(almatyLoc).Format("15:04"),
			sl.End.In(almatyLoc).Format("15:04"),
			sl.Mins)
	}
	return b.String(), nil
}

func dispatchConflicts(ctx context.Context, be Backend, args json.RawMessage) (string, error) {
	var in struct {
		Emails []string `json:"emails"`
		Start  string   `json:"start"`
		End    string   `json:"end"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("bad arguments: %w", err)
	}
	start, err := time.ParseInLocation("2006-01-02 15:04", in.Start, almatyLoc)
	if err != nil {
		return "", fmt.Errorf("bad 'start' (want 'YYYY-MM-DD HH:MM'): %w", err)
	}
	end, err := time.ParseInLocation("2006-01-02 15:04", in.End, almatyLoc)
	if err != nil {
		return "", fmt.Errorf("bad 'end' (want 'YYYY-MM-DD HH:MM'): %w", err)
	}
	conflicts, err := be.MeetingConflicts(ctx, in.Emails, start.UTC(), end.UTC(), uuid.Nil)
	if err != nil {
		return "", err
	}
	if len(conflicts) == 0 {
		return "No conflicts — everyone is free in that window.", nil
	}
	sort.Slice(conflicts, func(i, j int) bool { return conflicts[i].Start.Before(conflicts[j].Start) })
	var b strings.Builder
	b.WriteString("Conflicts found:\n")
	for _, c := range conflicts {
		fmt.Fprintf(&b, "- %s busy with %q at %s–%s\n",
			c.PersonName, c.MeetingName,
			c.Start.In(almatyLoc).Format("2006-01-02 15:04"),
			c.End.In(almatyLoc).Format("15:04"))
	}
	return b.String(), nil
}
