package reminder_scheduler

import (
	"bytes"
	"fmt"
	"html/template"
	texttemplate "text/template"
)

// Reminder email — Go port of docs/email-templates/meeting-reminder.html.
// Keep the two in sync. Localized ru/en/kk; default ru. User-supplied fields
// (Name, Title, Participants) are auto-escaped by html/template.

type remLabels struct {
	Hi, Lead, When, Who, Cta, Foot, Unsub, Made, DefaultName, DefaultTitle string
}

func reminderLabels(lang string) remLabels {
	switch lang {
	case "en":
		return remLabels{
			Hi: "Hi", Lead: "A gentle nudge — you have a meeting coming up:",
			When: "When", Who: "Who", Cta: "Join Google Meet",
			Foot:  "You’re receiving this reminder from Lead Cat.",
			Unsub: "Manage reminders", Made: "See you there 🐾",
			DefaultName: "there", DefaultTitle: "Your meeting",
		}
	case "kk":
		return remLabels{
			Hi: "Сәлем", Lead: "Жайлап еске саламыз — жақында кездесуің бар:",
			When: "Қашан", Who: "Кім", Cta: "Google Meet-ке қосылу",
			Foot:  "Бұл еске салуды Lead Cat-тан алдың.",
			Unsub: "Еске салуларды басқару", Made: "Сонда көріскенше 🐾",
			DefaultName: "дос", DefaultTitle: "Кездесу",
		}
	default: // ru
		return remLabels{
			Hi: "Привет", Lead: "Мягко напоминаем — скоро у тебя встреча:",
			When: "Когда", Who: "Кто", Cta: "Подключиться к Google Meet",
			Foot:  "Ты получаешь это напоминание от Lead Cat.",
			Unsub: "Управление напоминаниями", Made: "До встречи 🐾",
			DefaultName: "друг", DefaultTitle: "Встреча",
		}
	}
}

func reminderEmailSubject(lang, title, timeStr string) string {
	switch lang {
	case "en":
		return fmt.Sprintf(`Reminder: "%s" at %s`, title, timeStr)
	case "kk":
		return fmt.Sprintf("Еске салу: «%s» %s", title, timeStr)
	default:
		return fmt.Sprintf("Напоминание: «%s» в %s", title, timeStr)
	}
}

type reminderEmailData struct {
	Lang           string
	Name           string
	Title          string
	Date           string
	Time           string
	Tz             string
	Participants   string
	MeetLink       string
	UnsubscribeURL string
	L              remLabels
	// Button is the pre-rendered, Outlook-hardened CTA. It carries
	// conditional comments (VML), which html/template would otherwise strip,
	// so it is built in Go and injected as trusted HTML.
	Button template.HTML
}

// reminderButton renders the bulletproof CTA: a VML roundrect for Outlook and
// a styled anchor for everyone else. href/label are HTML-escaped (Meet links
// are Google-generated, but escaping keeps the attribute context safe).
func reminderButton(meetLink, cta string) template.HTML {
	href := template.HTMLEscapeString(meetLink)
	label := template.HTMLEscapeString(cta)
	return template.HTML(`<!--[if mso]><v:roundrect xmlns:v="urn:schemas-microsoft-com:vml" xmlns:w="urn:schemas-microsoft-com:office:word" href="` + href + `" style="height:48px;v-text-anchor:middle;width:260px;" arcsize="25%" stroke="f" fillcolor="#E8714C"><w:anchorlock/><center style="color:#FFFFFF;font-family:Arial,Helvetica,sans-serif;font-size:16px;font-weight:bold;">` + label + `</center></v:roundrect><![endif]-->` +
		`<!--[if !mso]><!-- --><a href="` + href + `" target="_blank" style="display:inline-block; padding:15px 34px; font-family:Arial,Helvetica,sans-serif; font-size:16px; font-weight:bold; color:#FFFFFF; text-decoration:none; border-radius:12px; mso-padding-alt:0;">` + label + `</a><!--<![endif]-->`)
}

var reminderEmailTemplate = template.Must(template.New("reminder").Parse(`<!DOCTYPE html>
<html lang="{{.Lang}}">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><meta name="x-apple-disable-message-reformatting"><meta name="color-scheme" content="light"><meta name="supported-color-schemes" content="light"><title>Lead Cat</title></head>
<body style="margin:0; padding:0; background-color:#F1EADF;">
<div style="display:none; max-height:0; overflow:hidden; opacity:0; mso-hide:all; font-size:1px; line-height:1px; color:#F1EADF;">{{.Title}} — {{.Time}}</div>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:#F1EADF;"><tr><td align="center" style="padding:28px 12px;">
<table role="presentation" width="600" cellpadding="0" cellspacing="0" border="0" style="width:600px; max-width:600px; background-color:#FBF7F0; border-radius:18px; overflow:hidden; font-family:Georgia,'Times New Roman',serif;">
<tr><td style="padding:24px 32px 8px 32px;"><span style="font-size:18px; font-weight:bold; color:#E8714C;">🐾&nbsp;Lead&nbsp;Cat</span></td></tr>
<tr><td style="padding:8px 32px 4px 32px;"><h1 style="margin:0; font-size:24px; line-height:1.25; font-weight:bold; color:#3A332E;">{{.L.Hi}}, {{.Name}}!</h1></td></tr>
<tr><td style="padding:8px 32px 0 32px;"><p style="margin:0; font-size:16px; line-height:1.6; color:#5C5249; font-family:Arial,Helvetica,sans-serif;">{{.L.Lead}}</p></td></tr>
<tr><td style="padding:20px 32px 4px 32px;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:#FFFFFF; border-left:4px solid #E8714C; border-radius:10px;"><tr><td style="padding:22px 24px;">
<p style="margin:0 0 14px 0; font-size:21px; line-height:1.25; font-weight:bold; color:#3A332E;">«{{.Title}}»</p>
<table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%">
<tr><td valign="top" style="padding:4px 0; font-family:Arial,Helvetica,sans-serif; font-size:14px; color:#A89C90; width:96px;">🗓&nbsp;{{.L.When}}</td>
<td valign="top" style="padding:4px 0; font-family:Arial,Helvetica,sans-serif; font-size:15px; font-weight:bold; color:#3A332E;">{{.Date}}, {{.Time}}{{if .Tz}} <span style="font-weight:normal; color:#8A7F75;">({{.Tz}})</span>{{end}}</td></tr>
{{if .Participants}}<tr><td valign="top" style="padding:4px 0; font-family:Arial,Helvetica,sans-serif; font-size:14px; color:#A89C90;">👥&nbsp;{{.L.Who}}</td>
<td valign="top" style="padding:4px 0; font-family:Arial,Helvetica,sans-serif; font-size:15px; color:#5C5249;">{{.Participants}}</td></tr>{{end}}
</table></td></tr></table></td></tr>
{{if .MeetLink}}<tr><td align="center" style="padding:24px 32px 8px 32px;">
<table role="presentation" cellpadding="0" cellspacing="0" border="0"><tr><td align="center" bgcolor="#E8714C" style="border-radius:12px;">
{{.Button}}
</td></tr></table></td></tr>{{end}}
<tr><td align="center" style="padding:10px 32px 34px 32px;"><p style="margin:0; font-size:14px; color:#A89C90; font-family:Arial,Helvetica,sans-serif;">{{.L.Made}}</p></td></tr>
<tr><td style="background-color:#3A332E; padding:22px 32px;">
<p style="margin:0 0 4px 0; font-size:14px; font-weight:bold; color:#FBF7F0; font-family:Arial,Helvetica,sans-serif;">🐾 Lead Cat</p>
<p style="margin:0 0 10px 0; font-size:12px; line-height:1.5; color:#B7ACA0; font-family:Arial,Helvetica,sans-serif;">{{.L.Foot}}</p>
<p style="margin:0; font-size:12px; color:#B7ACA0; font-family:Arial,Helvetica,sans-serif;"><a href="{{.UnsubscribeURL}}" target="_blank" style="color:#F6C453; text-decoration:underline;">{{.L.Unsub}}</a></p>
</td></tr>
</table></td></tr></table></body></html>`))

func renderReminderEmail(d reminderEmailData) (string, error) {
	d = applyReminderDefaults(d)
	var b bytes.Buffer
	if err := reminderEmailTemplate.Execute(&b, d); err != nil {
		return "", err
	}
	return b.String(), nil
}

// reminderTextTemplate is the plain-text alternative — Go port of
// docs/email-templates/meeting-reminder.txt. text/template does not escape,
// which is correct for a text/plain part.
var reminderTextTemplate = texttemplate.Must(texttemplate.New("reminder.txt").Parse(
	`{{.L.Hi}}, {{.Name}}!

{{.L.Lead}}

«{{.Title}}»
{{.L.When}}: {{.Date}}, {{.Time}}{{if .Tz}} ({{.Tz}}){{end}}{{if .Participants}}
{{.L.Who}}: {{.Participants}}{{end}}{{if .MeetLink}}

{{.L.Cta}}: {{.MeetLink}}{{end}}

{{.L.Made}}
{{.L.Unsub}}: {{.UnsubscribeURL}}
`))

func renderReminderText(d reminderEmailData) (string, error) {
	d = applyReminderDefaults(d)
	var b bytes.Buffer
	if err := reminderTextTemplate.Execute(&b, d); err != nil {
		return "", err
	}
	return b.String(), nil
}

func applyReminderDefaults(d reminderEmailData) reminderEmailData {
	d.L = reminderLabels(d.Lang)
	if d.Name == "" {
		d.Name = d.L.DefaultName
	}
	if d.Title == "" {
		d.Title = d.L.DefaultTitle
	}
	if d.UnsubscribeURL == "" {
		d.UnsubscribeURL = "#"
	}
	if d.MeetLink != "" {
		d.Button = reminderButton(d.MeetLink, d.L.Cta)
	}
	return d
}
