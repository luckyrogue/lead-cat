package emailtemplates

import (
	"bytes"
	"html/template"
	texttemplate "text/template"
)

type BookingConfirmationData struct {
	Language   string
	BookerName string
	EventTitle string
	Date       string
	Time       string
	Tz         string
	MeetLink   string
}

func bookingConfirmSubject(lang string) string {
	switch NormalizeLang(lang) {
	case "en":
		return "Your booking is confirmed 🐾"
	case "kk":
		return "Брондауыңыз расталды 🐾"
	default:
		return "Ваша бронь подтверждена 🐾"
	}
}

type bookingConfirmL struct{ Hi, Intro, EventLbl, WhenLbl, Cta, Foot, DefaultName string }

func bookingConfirmLabels(lang string) bookingConfirmL {
	switch NormalizeLang(lang) {
	case "en":
		return bookingConfirmL{Hi: "Hi", Intro: "Your meeting is booked. Here are the details:", EventLbl: "Meeting", WhenLbl: "When", Cta: "Join Google Meet", Foot: "You're receiving this because you booked a meeting via Lead Cat.", DefaultName: "there"}
	case "kk":
		return bookingConfirmL{Hi: "Сәлем", Intro: "Кездесуіңіз брондалды. Толығырақ:", EventLbl: "Кездесу", WhenLbl: "Қашан", Cta: "Google Meet-ке қосылу", Foot: "Бұл хатты Lead Cat арқылы кездесу брондағаныңыз үшін алдыңыз.", DefaultName: "дос"}
	default:
		return bookingConfirmL{Hi: "Привет", Intro: "Ваша встреча забронирована. Детали:", EventLbl: "Встреча", WhenLbl: "Когда", Cta: "Подключиться к Google Meet", Foot: "Вы получили это письмо, потому что забронировали встречу через Lead Cat.", DefaultName: "друг"}
	}
}

type BookingHostNotificationData struct {
	Language    string
	EventTitle  string
	BookerName  string
	BookerEmail string
	Date        string
	Time        string
	Tz          string
	MeetLink    string
}

func bookingHostSubject(lang, eventTitle string) string {
	switch NormalizeLang(lang) {
	case "en":
		return "New booking: " + eventTitle
	case "kk":
		return "Жаңа брондау: " + eventTitle
	default:
		return "Новая бронь: " + eventTitle
	}
}

type bookingHostL struct{ Hi, Intro, EventLbl, WhoLbl, WhenLbl, Cta, Foot string }

func bookingHostLabels(lang string) bookingHostL {
	switch NormalizeLang(lang) {
	case "en":
		return bookingHostL{Hi: "Hi", Intro: "You have a new booking.", EventLbl: "Meeting", WhoLbl: "Booked by", WhenLbl: "When", Cta: "Join Google Meet", Foot: "You're receiving this because someone booked time with you via Lead Cat."}
	case "kk":
		return bookingHostL{Hi: "Сәлем", Intro: "Сізде жаңа брондау бар.", EventLbl: "Кездесу", WhoLbl: "Брондаған", WhenLbl: "Қашан", Cta: "Google Meet-ке қосылу", Foot: "Бұл хатты біреу Lead Cat арқылы сізбен уақыт брондағандықтан алдыңыз."}
	default:
		return bookingHostL{Hi: "Привет", Intro: "У вас новая бронь.", EventLbl: "Встреча", WhoLbl: "Забронировал", WhenLbl: "Когда", Cta: "Подключиться к Google Meet", Foot: "Вы получили это письмо, потому что кто-то забронировал время через Lead Cat."}
	}
}

type bookingRow struct{ Label, Value string }

type bookingRenderData struct {
	Lang    string
	Hi      string
	Name    string // greeting target; empty for host email (generic intro)
	Intro   string
	Rows    []bookingRow
	Foot    string
	Button  template.HTML
	HasMeet bool
}

var bookingHTMLTemplate = template.Must(template.New("booking").Parse(`<!DOCTYPE html>
<html lang="{{.Lang}}">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><meta name="color-scheme" content="light"><title>Lead Cat</title></head>
<body style="margin:0; padding:0; background-color:#F1EADF;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:#F1EADF;"><tr><td align="center" style="padding:28px 12px;">
<table role="presentation" width="600" cellpadding="0" cellspacing="0" border="0" style="width:600px; max-width:600px; background-color:#FBF7F0; border-radius:18px; overflow:hidden; font-family:Georgia,'Times New Roman',serif;">
<tr><td style="background-color:#E8714C; padding:26px 32px;"><span style="font-size:22px; font-weight:bold; color:#FFFFFF;">🐾&nbsp;Lead&nbsp;Cat</span></td></tr>
<tr><td style="padding:34px 32px 6px 32px;"><h1 style="margin:0; font-size:26px; line-height:1.2; font-weight:bold; color:#3A332E;">{{.Hi}}{{if .Name}}, {{.Name}}{{end}}!</h1></td></tr>
<tr><td style="padding:10px 32px 8px 32px;"><p style="margin:0; font-size:16px; line-height:1.6; color:#5C5249; font-family:Arial,Helvetica,sans-serif;">{{.Intro}}</p></td></tr>
{{range .Rows}}<tr><td style="padding:6px 32px;"><p style="margin:0; font-size:15px; line-height:1.5; color:#5C5249; font-family:Arial,Helvetica,sans-serif;"><strong style="color:#3A332E;">{{.Label}}:</strong> {{.Value}}</p></td></tr>
{{end}}{{if .HasMeet}}<tr><td align="center" style="padding:24px 32px 8px 32px;">
<table role="presentation" cellpadding="0" cellspacing="0" border="0"><tr><td align="center" bgcolor="#E8714C" style="border-radius:12px;">{{.Button}}</td></tr></table></td></tr>{{end}}
<tr><td style="background-color:#3A332E; padding:20px 32px;"><p style="margin:0; font-size:12px; line-height:1.5; color:#B7ACA0; font-family:Arial,Helvetica,sans-serif;">{{.Foot}}</p></td></tr>
</table></td></tr></table></body></html>`))

var bookingTextTemplate = texttemplate.Must(texttemplate.New("booking.txt").Parse(
	`{{.Hi}}{{if .Name}}, {{.Name}}{{end}}!

{{.Intro}}
{{range .Rows}}
{{.Label}}: {{.Value}}{{end}}
{{if .HasMeet}}
{{.MeetLine}}{{end}}

— Lead Cat
{{.Foot}}
`))

type textRenderData struct {
	bookingRenderData
	MeetLine string
}

func renderBooking(rd bookingRenderData, meetLink, cta string) (text, htmlOut string, err error) {
	var hb bytes.Buffer
	if err = bookingHTMLTemplate.Execute(&hb, rd); err != nil {
		return "", "", err
	}
	trd := textRenderData{bookingRenderData: rd}
	if rd.HasMeet {
		trd.MeetLine = cta + ": " + meetLink
	}
	var tb bytes.Buffer
	if err = bookingTextTemplate.Execute(&tb, trd); err != nil {
		return "", "", err
	}
	return tb.String(), hb.String(), nil
}

func RenderBookingConfirmation(d BookingConfirmationData) (subject, text, html string, err error) {
	lang := NormalizeLang(d.Language)
	l := bookingConfirmLabels(lang)
	name := d.BookerName
	if name == "" {
		name = l.DefaultName
	}
	rows := []bookingRow{
		{Label: l.EventLbl, Value: d.EventTitle},
		{Label: l.WhenLbl, Value: d.Date + ", " + d.Time + " (" + d.Tz + ")"},
	}
	rd := bookingRenderData{
		Lang: lang, Hi: l.Hi, Name: name, Intro: l.Intro, Rows: rows, Foot: l.Foot,
		HasMeet: d.MeetLink != "",
	}
	if rd.HasMeet {
		rd.Button = BulletproofButton(d.MeetLink, l.Cta)
	}
	text, html, err = renderBooking(rd, d.MeetLink, l.Cta)
	if err != nil {
		return "", "", "", err
	}
	return bookingConfirmSubject(lang), text, html, nil
}

func RenderBookingHostNotification(d BookingHostNotificationData) (subject, text, html string, err error) {
	lang := NormalizeLang(d.Language)
	l := bookingHostLabels(lang)
	rows := []bookingRow{
		{Label: l.EventLbl, Value: d.EventTitle},
		{Label: l.WhoLbl, Value: d.BookerName + " <" + d.BookerEmail + ">"},
		{Label: l.WhenLbl, Value: d.Date + ", " + d.Time + " (" + d.Tz + ")"},
	}
	rd := bookingRenderData{
		Lang: lang, Hi: l.Hi, Name: "", Intro: l.Intro, Rows: rows, Foot: l.Foot,
		HasMeet: d.MeetLink != "",
	}
	if rd.HasMeet {
		rd.Button = BulletproofButton(d.MeetLink, l.Cta)
	}
	text, html, err = renderBooking(rd, d.MeetLink, l.Cta)
	if err != nil {
		return "", "", "", err
	}
	return bookingHostSubject(lang, d.EventTitle), text, html, nil
}
