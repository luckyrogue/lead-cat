package microsoft

import "strings"

type graphDateTime struct {
	DateTime string `json:"dateTime"`
	TimeZone string `json:"timeZone"`
}

type graphEmail struct {
	Address string `json:"address"`
}

type graphAttendee struct {
	EmailAddress graphEmail `json:"emailAddress"`
	Type         string     `json:"type"`
}

type graphBody struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

type graphEvent struct {
	ID                    string          `json:"id,omitempty"`
	Subject               string          `json:"subject"`
	Body                  *graphBody      `json:"body,omitempty"`
	Start                 graphDateTime   `json:"start"`
	End                   graphDateTime   `json:"end"`
	Attendees             []graphAttendee `json:"attendees,omitempty"`
	IsOnlineMeeting       bool            `json:"isOnlineMeeting,omitempty"`
	OnlineMeetingProvider string          `json:"onlineMeetingProvider,omitempty"`
	OnlineMeeting         *struct {
		JoinURL string `json:"joinUrl"`
	} `json:"onlineMeeting,omitempty"`
}

type graphScheduleResponse struct {
	Value []struct {
		ScheduleID    string `json:"scheduleId"`
		ScheduleItems []struct {
			Status string        `json:"status"`
			Start  graphDateTime `json:"start"`
			End    graphDateTime `json:"end"`
		} `json:"scheduleItems"`
	} `json:"value"`
}

type graphErrorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func trimFraction(s string) string {
	if idx := strings.IndexByte(s, '.'); idx != -1 {
		return s[:idx]
	}
	return s
}
