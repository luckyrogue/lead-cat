package meetingedit

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/domain/meeting"
	"github.com/luckyrogue/lead-cat/internal/platform/boti18n"
)

type Backend interface {
	ListEditableMeetings(ctx context.Context, telegramID int64) ([]model.MeetingWithTZ, error)
	UpdateMeeting(ctx context.Context, organizationID, userID, meetingID uuid.UUID, in application.UpdateMeetingInput) (model.Meeting, error)
	UpdateSeries(ctx context.Context, organizationID, userID, meetingID uuid.UUID, in application.SeriesUpdateInput) (int, error)
	ListParticipants(ctx context.Context, meetingID uuid.UUID) ([]model.MeetingParticipant, error)
	SearchEmployees(ctx context.Context, organizationID uuid.UUID, query string) ([]model.Employee, error)
	AddParticipant(ctx context.Context, organizationID, userID, meetingID uuid.UUID, email string) error
	RemoveParticipant(ctx context.Context, organizationID, userID, meetingID uuid.UUID, email string) error
	CancelMeeting(ctx context.Context, organizationID, userID, meetingID uuid.UUID) error
	CancelSeries(ctx context.Context, organizationID, userID, meetingID uuid.UUID) (int, error)
	MeetingUpdateConflicts(ctx context.Context, organizationID, meetingID uuid.UUID, in application.UpdateMeetingInput) ([]application.Conflict, error)
}

type sessions interface {
	Get(ctx context.Context, telegramID int64) (*State, error)
	Set(ctx context.Context, telegramID int64, s State) error
	Del(ctx context.Context, telegramID int64) error
}

type Service struct {
	backend  Backend
	sessions sessions
}

func New(backend Backend, sess sessions) *Service {
	return &Service{backend: backend, sessions: sess}
}

func (s *Service) Start(ctx context.Context, telegramID int64, lang string) Reply {
	ms, err := s.backend.ListEditableMeetings(ctx, telegramID)
	if err != nil {
		return Reply{Text: boti18n.T(lang, "medit.list_failed")}
	}
	if len(ms) == 0 {
		return Reply{Text: boti18n.T(lang, "medit.none_editable")}
	}
	var rows [][]Button
	for _, m := range ms {
		rows = append(rows, []Button{{Text: m.Name, Data: "medit:pick:" + m.ID.String()}})
	}
	return Reply{Text: boti18n.T(lang, "medit.pick_meeting"), Keyboard: rows}
}

func (s *Service) OnCallback(ctx context.Context, telegramID int64, data, lang string) (Reply, bool) {
	switch {
	case strings.HasPrefix(data, "medit:pick:"):
		return s.pick(ctx, telegramID, strings.TrimPrefix(data, "medit:pick:"), lang), true
	case strings.HasPrefix(data, "medit:field:"):
		return s.field(ctx, telegramID, strings.TrimPrefix(data, "medit:field:"), lang), true
	case strings.HasPrefix(data, "medit:set:rec:"):
		return s.setRec(ctx, telegramID, strings.TrimPrefix(data, "medit:set:rec:"), lang), true
	case data == "medit:apply":
		return s.apply(ctx, telegramID, lang), true
	case data == "medit:applyforce":
		return s.applyForce(ctx, telegramID, lang), true
	case data == "medit:cancel":
		_ = s.sessions.Del(ctx, telegramID)
		return Reply{Text: boti18n.T(lang, "medit.cancelled"), Edit: true}, true
	case data == "medit:menu":
		return s.backToMenu(ctx, telegramID, lang), true
	case data == "medit:parts":
		return s.parts(ctx, telegramID, lang), true
	case data == "medit:padd":
		return s.padd(ctx, telegramID, lang), true
	case strings.HasPrefix(data, "medit:padd:"):
		return s.paddPick(ctx, telegramID, strings.TrimPrefix(data, "medit:padd:"), lang), true
	case data == "medit:premc":
		return s.premConfirm(ctx, telegramID, lang), true
	case strings.HasPrefix(data, "medit:prem:"):
		return s.prem(ctx, telegramID, strings.TrimPrefix(data, "medit:prem:"), lang), true
	case data == "medit:scope:one":
		return s.setScope(ctx, telegramID, "one", lang), true
	case data == "medit:scope:series":
		return s.setScope(ctx, telegramID, "series", lang), true
	case data == "medit:delete":
		return s.confirmDelete(ctx, telegramID, lang), true
	case data == "medit:delconf":
		return s.doDelete(ctx, telegramID, lang), true
	}
	return Reply{}, false
}

func (s *Service) OnText(ctx context.Context, telegramID int64, text, lang string) (Reply, bool) {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil || st.Step != stepAwaiting {
		return Reply{}, false
	}
	text = strings.TrimSpace(text)
	if st.AwaitingField == "participant" {
		return s.searchParticipant(ctx, telegramID, st, text, lang), true
	}
	if st.AwaitingField == "datetime" {
		if st.Scope == "series" {
			start, end, perr := parseTimeRange(text)
			if perr != nil {
				return Reply{Text: boti18n.T(lang, "medit.bad_time_range")}, true
			}
			st.Overrides["start"] = start
			st.Overrides["end"] = end
		} else {
			d, start, end, perr := parseDateTime(text)
			if perr != nil {
				return Reply{Text: boti18n.T(lang, "medit.bad_datetime")}, true
			}
			st.Overrides["date"] = d
			st.Overrides["start"] = start
			st.Overrides["end"] = end
		}
	} else {
		if text == "" {
			return Reply{Text: boti18n.T(lang, "medit.empty_value")}, true
		}
		st.Overrides[st.AwaitingField] = text
	}
	st.Step = stepMenu
	st.AwaitingField = ""
	_ = s.sessions.Set(ctx, telegramID, *st)

	return menuReply(*st, false, lang), true
}

func (s *Service) pick(ctx context.Context, telegramID int64, idStr, lang string) Reply {
	mid, err := uuid.Parse(idStr)
	if err != nil {
		return Reply{Text: boti18n.T(lang, "medit.unknown_meeting")}
	}
	ms, err := s.backend.ListEditableMeetings(ctx, telegramID)
	if err != nil {
		return Reply{Text: boti18n.T(lang, "medit.get_meeting_failed")}
	}
	var found *model.MeetingWithTZ
	for i := range ms {
		if ms[i].ID == mid {
			found = &ms[i]
			break
		}
	}
	if found == nil || found.OrganizerUserID == nil {
		return Reply{Text: boti18n.T(lang, "medit.not_editable")}
	}
	loc := loadLoc(found.TZ)
	st := State{
		Step:           stepMenu,
		MeetingID:      mid.String(),
		OrganizationID: found.OrganizationID.String(),
		UserID:         found.OrganizerUserID.String(),
		Cur:            snapshot(found.Meeting, loc),
		Overrides:      map[string]string{},
	}
	if found.SeriesID != nil {
		st.SeriesID = found.SeriesID.String()
		_ = s.sessions.Set(ctx, telegramID, st)
		return scopeReply(lang)
	}
	st.Scope = "one"
	_ = s.sessions.Set(ctx, telegramID, st)
	return menuReply(st, true, lang)
}

func (s *Service) field(ctx context.Context, telegramID int64, f, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "medit.session_expired")}
	}
	if f == "rec" {
		if st.Scope == "series" {
			return menuReply(*st, true, lang)
		}
		return recReply(lang)
	}
	prompt, ok := fieldPrompt(f, lang)
	if !ok {
		return Reply{}
	}
	if f == "datetime" && st.Scope == "series" {
		prompt = boti18n.T(lang, "medit.prompt_time_series")
	}
	st.Step = stepAwaiting
	st.AwaitingField = f
	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{Text: prompt}
}

func (s *Service) setRec(ctx context.Context, telegramID int64, val, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "medit.session_expired")}
	}
	if !meeting.Recurrence(val).Valid() {
		return Reply{}
	}
	st.Overrides["recurrence"] = val
	st.Step = stepMenu
	st.AwaitingField = ""
	_ = s.sessions.Set(ctx, telegramID, *st)
	return menuReply(*st, true, lang)
}

func (s *Service) apply(ctx context.Context, telegramID int64, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "medit.session_expired")}
	}
	if st.SeriesID != "" && st.Scope == "" {
		return Reply{Text: boti18n.T(lang, "medit.choose_scope_first"), Keyboard: scopeReply(lang).Keyboard, Edit: true}
	}
	if len(st.Overrides) == 0 {
		return Reply{Text: boti18n.T(lang, "medit.no_changes"), Keyboard: menuKeyboard(st.Scope, lang), Edit: true}
	}

	if st.Scope != "series" {
		if _, ok := st.Overrides["date"]; ok {
			orgID, _ := uuid.Parse(st.OrganizationID)
			mid, _ := uuid.Parse(st.MeetingID)
			conflicts, cerr := s.backend.MeetingUpdateConflicts(ctx, orgID, mid, toInput(st.Overrides))
			if cerr == nil && len(conflicts) > 0 {
				return Reply{Text: formatConflictWarning(conflicts, lang), Keyboard: conflictKeyboard(lang), Edit: true}
			}
		}
	}
	return s.doApply(ctx, telegramID, st, lang)
}

func (s *Service) applyForce(ctx context.Context, telegramID int64, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "medit.session_expired")}
	}
	return s.doApply(ctx, telegramID, st, lang)
}

func (s *Service) doApply(ctx context.Context, telegramID int64, st *State, lang string) Reply {
	orgID, _ := uuid.Parse(st.OrganizationID)
	uid, _ := uuid.Parse(st.UserID)
	mid, _ := uuid.Parse(st.MeetingID)
	if st.Scope == "series" {
		n, err := s.backend.UpdateSeries(ctx, orgID, uid, mid, seriesInput(st.Overrides))
		if err != nil {
			switch {
			case errors.Is(err, application.ErrInvalidInput):
				return Reply{Text: boti18n.T(lang, "medit.invalid_data")}
			case errors.Is(err, application.ErrForbidden):
				_ = s.sessions.Del(ctx, telegramID)
				return Reply{Text: boti18n.T(lang, "medit.forbidden")}
			case errors.Is(err, model.ErrMeetingNotEditable):
				_ = s.sessions.Del(ctx, telegramID)
				return Reply{Text: boti18n.T(lang, "medit.series_not_editable")}
			default:
				return Reply{Text: boti18n.T(lang, "medit.update_series_failed")}
			}
		}
		_ = s.sessions.Del(ctx, telegramID)
		return Reply{Text: boti18n.T(lang, "medit.series_updated", n)}
	}
	m, err := s.backend.UpdateMeeting(ctx, orgID, uid, mid, toInput(st.Overrides))
	if err != nil {
		switch {
		case errors.Is(err, application.ErrInvalidInput):
			return Reply{Text: boti18n.T(lang, "medit.invalid_data")}
		case errors.Is(err, application.ErrForbidden):
			_ = s.sessions.Del(ctx, telegramID)
			return Reply{Text: boti18n.T(lang, "medit.forbidden")}
		case errors.Is(err, model.ErrMeetingNotEditable):
			_ = s.sessions.Del(ctx, telegramID)
			return Reply{Text: boti18n.T(lang, "medit.meeting_not_editable")}
		default:
			return Reply{Text: boti18n.T(lang, "medit.update_failed")}
		}
	}
	_ = s.sessions.Del(ctx, telegramID)
	return Reply{Text: boti18n.T(lang, "medit.updated_done") + "\n" + summary(m)}
}

func formatConflictWarning(cs []application.Conflict, lang string) string {
	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		loc = time.FixedZone("Almaty", 5*60*60)
	}
	var b strings.Builder
	b.WriteString(boti18n.T(lang, "medit.conflict_header"))
	for _, c := range cs {
		fmt.Fprintf(&b, "- %s — «%s» (%s–%s)\n",
			c.PersonName, c.MeetingName, c.Start.In(loc).Format("15:04"), c.End.In(loc).Format("15:04"))
	}
	b.WriteString(boti18n.T(lang, "medit.conflict_apply_q"))
	return b.String()
}

func conflictKeyboard(lang string) [][]Button {
	return [][]Button{{
		{Text: boti18n.T(lang, "medit.btn_apply_yes"), Data: "medit:applyforce"},
		{Text: boti18n.T(lang, "medit.btn_change_time"), Data: "medit:field:datetime"},
	}}
}

func (s *Service) backToMenu(ctx context.Context, telegramID int64, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "medit.session_expired")}
	}
	return menuReply(*st, true, lang)
}

func (s *Service) parts(ctx context.Context, telegramID int64, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "medit.session_expired")}
	}
	mid, _ := uuid.Parse(st.MeetingID)
	ps, err := s.backend.ListParticipants(ctx, mid)
	if err != nil {
		return Reply{Text: boti18n.T(lang, "medit.get_parts_failed")}
	}
	var emails []string
	for _, p := range ps {
		if p.Email != "" {
			emails = append(emails, p.Email)
		}
	}
	st.PartList = emails
	st.Step = stepMenu
	st.AwaitingField = ""
	_ = s.sessions.Set(ctx, telegramID, *st)
	return partsReply(emails, true, lang)
}

func partsReply(emails []string, edit bool, lang string) Reply {
	var rows [][]Button
	for i, e := range emails {
		rows = append(rows, []Button{{Text: "✖ " + e, Data: fmt.Sprintf("medit:prem:%d", i)}})
	}
	rows = append(rows, []Button{{Text: boti18n.T(lang, "medit.btn_add"), Data: "medit:padd"}})
	rows = append(rows, []Button{{Text: boti18n.T(lang, "medit.btn_back"), Data: "medit:menu"}})
	text := boti18n.T(lang, "medit.parts_title")
	if len(emails) == 0 {
		text = boti18n.T(lang, "medit.parts_empty")
	}
	return Reply{Text: text, Keyboard: rows, Edit: edit}
}

func (s *Service) padd(ctx context.Context, telegramID int64, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "medit.session_expired")}
	}
	st.Step = stepAwaiting
	st.AwaitingField = "participant"
	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{Text: boti18n.T(lang, "medit.padd_prompt")}
}

func (s *Service) searchParticipant(ctx context.Context, telegramID int64, st *State, query, lang string) Reply {
	orgID, _ := uuid.Parse(st.OrganizationID)
	emps, err := s.backend.SearchEmployees(ctx, orgID, query)
	if err != nil {
		return Reply{Text: boti18n.T(lang, "medit.search_failed")}
	}
	var cands []string
	var rows [][]Button
	seen := map[string]bool{}
	for _, e := range emps {
		if e.Email == "" || seen[e.Email] {
			continue
		}
		if len(cands) >= 10 {
			break
		}
		seen[e.Email] = true
		rows = append(rows, []Button{{Text: e.FullName + " — " + e.Email, Data: fmt.Sprintf("medit:padd:%d", len(cands))}})
		cands = append(cands, e.Email)
	}
	if addr, perr := mail.ParseAddress(strings.TrimSpace(query)); perr == nil {
		email := strings.ToLower(addr.Address)
		if !seen[email] {
			rows = append(rows, []Button{{Text: boti18n.T(lang, "medit.btn_add_email", email), Data: fmt.Sprintf("medit:padd:%d", len(cands))}})
			cands = append(cands, email)
		}
	}
	if len(cands) == 0 {
		return Reply{Text: boti18n.T(lang, "medit.search_none")}
	}
	st.PartCands = cands
	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{Text: boti18n.T(lang, "medit.padd_pick"), Keyboard: rows}
}

func (s *Service) paddPick(ctx context.Context, telegramID int64, idxStr, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "medit.session_expired")}
	}
	email, ok := indexInto(st.PartCands, idxStr)
	if !ok {
		return Reply{Text: boti18n.T(lang, "medit.cand_not_found")}
	}
	orgID, _ := uuid.Parse(st.OrganizationID)
	uid, _ := uuid.Parse(st.UserID)
	mid, _ := uuid.Parse(st.MeetingID)
	if err := s.backend.AddParticipant(ctx, orgID, uid, mid, email); err != nil {
		switch {
		case errors.Is(err, application.ErrInvalidInput):
			return Reply{Text: boti18n.T(lang, "medit.already_or_invalid")}
		case errors.Is(err, application.ErrForbidden):
			_ = s.sessions.Del(ctx, telegramID)
			return Reply{Text: boti18n.T(lang, "medit.forbidden")}
		default:
			return Reply{Text: boti18n.T(lang, "medit.add_failed")}
		}
	}
	return s.parts(ctx, telegramID, lang)
}

func (s *Service) prem(ctx context.Context, telegramID int64, idxStr, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "medit.session_expired")}
	}
	email, ok := indexInto(st.PartList, idxStr)
	if !ok {
		return Reply{Text: boti18n.T(lang, "medit.part_not_found")}
	}
	st.PendingRemove = email
	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{
		Text: boti18n.T(lang, "medit.remove_confirm", email),
		Edit: true,
		Keyboard: [][]Button{
			{{Text: boti18n.T(lang, "medit.btn_yes"), Data: "medit:premc"}},
			{{Text: boti18n.T(lang, "medit.btn_cancel_back"), Data: "medit:parts"}},
		},
	}
}

func (s *Service) premConfirm(ctx context.Context, telegramID int64, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "medit.session_expired")}
	}
	email := st.PendingRemove
	if email == "" {
		return Reply{Text: boti18n.T(lang, "medit.nothing_to_remove")}
	}
	orgID, _ := uuid.Parse(st.OrganizationID)
	uid, _ := uuid.Parse(st.UserID)
	mid, _ := uuid.Parse(st.MeetingID)
	if err := s.backend.RemoveParticipant(ctx, orgID, uid, mid, email); err != nil {
		if errors.Is(err, application.ErrForbidden) {
			_ = s.sessions.Del(ctx, telegramID)
			return Reply{Text: boti18n.T(lang, "medit.forbidden")}
		}
		return Reply{Text: boti18n.T(lang, "medit.remove_failed")}
	}
	st.PendingRemove = ""
	_ = s.sessions.Set(ctx, telegramID, *st)
	return s.parts(ctx, telegramID, lang)
}

func indexInto(list []string, idxStr string) (string, bool) {
	i, err := strconv.Atoi(idxStr)
	if err != nil || i < 0 || i >= len(list) {
		return "", false
	}
	return list[i], true
}

func fieldPrompt(f, lang string) (string, bool) {
	switch f {
	case "dept":
		return boti18n.T(lang, "medit.prompt_dept"), true
	case "type":
		return boti18n.T(lang, "medit.prompt_type"), true
	case "host":
		return boti18n.T(lang, "medit.prompt_host"), true
	case "description":
		return boti18n.T(lang, "medit.prompt_description"), true
	case "datetime":
		return boti18n.T(lang, "medit.prompt_datetime"), true
	}
	return "", false
}

func menuKeyboard(scope, lang string) [][]Button {
	if scope == "series" {
		return [][]Button{
			{{Text: boti18n.T(lang, "medit.btn_time"), Data: "medit:field:datetime"}},
			{{Text: boti18n.T(lang, "medit.btn_dept"), Data: "medit:field:dept"}, {Text: boti18n.T(lang, "medit.btn_type"), Data: "medit:field:type"}},
			{{Text: boti18n.T(lang, "medit.btn_host"), Data: "medit:field:host"}, {Text: boti18n.T(lang, "medit.btn_description"), Data: "medit:field:description"}},
			{{Text: boti18n.T(lang, "medit.btn_delete"), Data: "medit:delete"}},
			{{Text: boti18n.T(lang, "medit.btn_apply"), Data: "medit:apply"}, {Text: boti18n.T(lang, "medit.btn_cancel"), Data: "medit:cancel"}},
		}
	}
	return [][]Button{
		{{Text: boti18n.T(lang, "medit.btn_datetime"), Data: "medit:field:datetime"}},
		{{Text: boti18n.T(lang, "medit.btn_dept"), Data: "medit:field:dept"}, {Text: boti18n.T(lang, "medit.btn_type"), Data: "medit:field:type"}},
		{{Text: boti18n.T(lang, "medit.btn_host"), Data: "medit:field:host"}, {Text: boti18n.T(lang, "medit.btn_description"), Data: "medit:field:description"}},
		{{Text: boti18n.T(lang, "medit.btn_recurrence"), Data: "medit:field:rec"}},
		{{Text: boti18n.T(lang, "medit.btn_participants"), Data: "medit:parts"}},
		{{Text: boti18n.T(lang, "medit.btn_delete"), Data: "medit:delete"}},
		{{Text: boti18n.T(lang, "medit.btn_apply"), Data: "medit:apply"}, {Text: boti18n.T(lang, "medit.btn_cancel"), Data: "medit:cancel"}},
	}
}

func scopeReply(lang string) Reply {
	return Reply{
		Text: boti18n.T(lang, "medit.scope_q"),
		Edit: true,
		Keyboard: [][]Button{
			{{Text: boti18n.T(lang, "medit.btn_scope_one"), Data: "medit:scope:one"}},
			{{Text: boti18n.T(lang, "medit.btn_scope_series"), Data: "medit:scope:series"}},
		},
	}
}

func (s *Service) confirmDelete(ctx context.Context, telegramID int64, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "medit.session_expired")}
	}
	text := boti18n.T(lang, "medit.delete_one_q")
	if st.Scope == "series" {
		text = boti18n.T(lang, "medit.delete_series_q")
	}
	return Reply{
		Text: text,
		Edit: true,
		Keyboard: [][]Button{
			{{Text: boti18n.T(lang, "medit.btn_delete_yes"), Data: "medit:delconf"}},
			{{Text: boti18n.T(lang, "medit.btn_cancel_back"), Data: "medit:menu"}},
		},
	}
}

func (s *Service) doDelete(ctx context.Context, telegramID int64, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "medit.session_expired")}
	}
	orgID, _ := uuid.Parse(st.OrganizationID)
	uid, _ := uuid.Parse(st.UserID)
	mid, _ := uuid.Parse(st.MeetingID)

	if st.Scope == "series" {
		n, err := s.backend.CancelSeries(ctx, orgID, uid, mid)
		if err != nil {
			return s.deleteErrReply(ctx, telegramID, err, lang)
		}
		_ = s.sessions.Del(ctx, telegramID)
		return Reply{Text: boti18n.T(lang, "medit.series_deleted", n)}
	}
	if err := s.backend.CancelMeeting(ctx, orgID, uid, mid); err != nil {
		return s.deleteErrReply(ctx, telegramID, err, lang)
	}
	_ = s.sessions.Del(ctx, telegramID)
	return Reply{Text: boti18n.T(lang, "medit.meeting_deleted")}
}

func (s *Service) deleteErrReply(ctx context.Context, telegramID int64, err error, lang string) Reply {
	switch {
	case errors.Is(err, application.ErrForbidden):
		_ = s.sessions.Del(ctx, telegramID)
		return Reply{Text: boti18n.T(lang, "medit.forbidden")}
	case errors.Is(err, model.ErrMeetingNotEditable):
		_ = s.sessions.Del(ctx, telegramID)
		return Reply{Text: boti18n.T(lang, "medit.meeting_unavailable")}
	default:
		return Reply{Text: boti18n.T(lang, "medit.delete_failed")}
	}
}

func (s *Service) setScope(ctx context.Context, telegramID int64, scope, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "medit.session_expired")}
	}
	st.Scope = scope
	_ = s.sessions.Set(ctx, telegramID, *st)
	return menuReply(*st, true, lang)
}

func recReply(lang string) Reply {
	return Reply{Text: boti18n.T(lang, "medit.pick_recurrence"), Edit: true, Keyboard: [][]Button{
		{{Text: boti18n.T(lang, "medit.rec.once"), Data: "medit:set:rec:once"}},
		{{Text: boti18n.T(lang, "medit.rec.daily"), Data: "medit:set:rec:daily"}},
		{{Text: boti18n.T(lang, "medit.rec.weekly"), Data: "medit:set:rec:weekly"}},
		{{Text: boti18n.T(lang, "medit.rec.biweekly"), Data: "medit:set:rec:biweekly"}},
		{{Text: boti18n.T(lang, "medit.rec.monthly"), Data: "medit:set:rec:monthly"}},
	}}
}

func menuReply(st State, edit bool, lang string) Reply {
	return Reply{Text: menuText(st, lang), Keyboard: menuKeyboard(st.Scope, lang), Edit: edit}
}

func menuText(st State, lang string) string {
	eff := func(k string) string {
		if v, ok := st.Overrides[k]; ok {
			return v
		}
		return st.Cur[k]
	}
	mark := func(k string) string {
		if _, ok := st.Overrides[k]; ok {
			return " ★"
		}
		return ""
	}
	var b strings.Builder
	if st.Scope == "series" {
		b.WriteString(boti18n.T(lang, "medit.menu_series_header", st.Cur["date"]))
		tmark := ""
		if _, ok := st.Overrides["start"]; ok {
			tmark = " ★"
		}
		fmt.Fprintf(&b, "• %s: %s–%s%s\n", boti18n.T(lang, "medit.lbl_time"), eff("start"), eff("end"), tmark)
		fmt.Fprintf(&b, "• %s: %s%s\n", boti18n.T(lang, "medit.lbl_dept"), eff("dept"), mark("dept"))
		fmt.Fprintf(&b, "• %s: %s%s\n", boti18n.T(lang, "medit.lbl_type"), eff("type"), mark("type"))
		fmt.Fprintf(&b, "• %s: %s%s\n", boti18n.T(lang, "medit.lbl_host"), eff("host"), mark("host"))
		fmt.Fprintf(&b, "• %s: %s%s\n", boti18n.T(lang, "medit.lbl_description"), eff("description"), mark("description"))
		return b.String()
	}
	b.WriteString(boti18n.T(lang, "medit.menu_one_header"))
	dmark := ""
	if _, ok := st.Overrides["date"]; ok {
		dmark = " ★"
	}
	fmt.Fprintf(&b, "• %s: %s %s–%s%s\n", boti18n.T(lang, "medit.lbl_datetime"), eff("date"), eff("start"), eff("end"), dmark)
	fmt.Fprintf(&b, "• %s: %s%s\n", boti18n.T(lang, "medit.lbl_dept"), eff("dept"), mark("dept"))
	fmt.Fprintf(&b, "• %s: %s%s\n", boti18n.T(lang, "medit.lbl_type"), eff("type"), mark("type"))
	fmt.Fprintf(&b, "• %s: %s%s\n", boti18n.T(lang, "medit.lbl_host"), eff("host"), mark("host"))
	fmt.Fprintf(&b, "• %s: %s%s\n", boti18n.T(lang, "medit.lbl_description"), eff("description"), mark("description"))
	fmt.Fprintf(&b, "• %s: %s%s\n", boti18n.T(lang, "medit.lbl_recurrence"), recLabel(eff("recurrence"), lang), mark("recurrence"))
	return b.String()
}

func recLabel(v, lang string) string {
	switch v {
	case "", "once":
		return boti18n.T(lang, "medit.rec.once")
	case "daily":
		return boti18n.T(lang, "medit.rec.daily")
	case "weekly":
		return boti18n.T(lang, "medit.rec.weekly")
	case "biweekly":
		return boti18n.T(lang, "medit.rec.biweekly")
	case "monthly":
		return boti18n.T(lang, "medit.rec.monthly")
	default:
		return meeting.Recurrence(v).Label()
	}
}

func snapshot(m model.Meeting, loc *time.Location) map[string]string {
	s := m.StartsAt.In(loc)
	e := m.EndsAt.In(loc)
	return map[string]string{
		"dept": m.Dept, "type": m.Type, "host": m.Host,
		"description": m.Description, "recurrence": m.Recurrence,
		"date": s.Format("2006-01-02"), "start": s.Format("15:04"), "end": e.Format("15:04"),
	}
}

func toInput(ov map[string]string) application.UpdateMeetingInput {
	var in application.UpdateMeetingInput
	set := func(p **string, k string) {
		if v, ok := ov[k]; ok {
			vv := v
			*p = &vv
		}
	}
	set(&in.Dept, "dept")
	set(&in.Type, "type")
	set(&in.Host, "host")
	set(&in.Description, "description")
	set(&in.Recurrence, "recurrence")
	set(&in.Date, "date")
	set(&in.Start, "start")
	set(&in.End, "end")
	return in
}

func seriesInput(ov map[string]string) application.SeriesUpdateInput {
	var in application.SeriesUpdateInput
	set := func(p **string, k string) {
		if v, ok := ov[k]; ok {
			vv := v
			*p = &vv
		}
	}
	set(&in.Dept, "dept")
	set(&in.Type, "type")
	set(&in.Host, "host")
	set(&in.Description, "description")
	set(&in.Start, "start")
	set(&in.End, "end")
	return in
}

func summary(m model.Meeting) string {
	s := "«" + m.Name + "»"
	if m.MeetLink != "" {
		s += "\n🔗 " + m.MeetLink
	}
	return s
}

func loadLoc(tz string) *time.Location {
	if tz == "" {
		tz = "Asia/Almaty"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.UTC
	}
	return loc
}
