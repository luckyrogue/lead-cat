package emailtemplates

import (
	"bytes"
	"html/template"
	texttemplate "text/template"
)

// WelcomeData — Go port of docs/email-templates/welcome.{html,txt}.
type WelcomeData struct {
	Language         string
	FirstName        string
	AppURL           string
	UnsubscribeURL   string
}

type welcomeL struct {
	Pre, Hi, Intro, S1T, S1B, S2T, S2B, Cta, Help, Foot, Unsub, Made, DefaultName string
}

func welcomeLabels(lang string) welcomeL {
	switch NormalizeLang(lang) {
	case "en":
		return welcomeL{
			Pre: "Two taps to your first purrfectly-timed meeting.", Hi: "Hi",
			Intro: "Welcome aboard! Lead Cat sniffs out the purrfect time across everyone's calendars — no back-and-forth, no double-booking. Two quick steps and you're set.",
			S1T: "Connect your calendar", S1B: "Link Google in two taps so Lead Cat knows when you're free.",
			S2T: "Open the Mini App", S2B: "Everything lives inside Telegram — book, reschedule and confirm without leaving chat.",
			Cta: "Open Lead Cat", Help: "Works with Google Calendar & Telegram. No credit card, no claws out.",
			Foot: "You're receiving this because you signed up for Lead Cat.", Unsub: "Unsubscribe",
			Made: "Made with whiskers and warmth.", DefaultName: "there",
		}
	case "kk":
		return welcomeL{
			Pre: "Алғашқы кездесуіңе екі-ақ түрту қалды.", Hi: "Сәлем",
			Intro: "Қош келдің! Lead Cat бәрінің күнтізбесінен ортақ ыңғайлы уақытты тауып береді — артық хат алмасу да, қос брондау да жоқ. Екі қадам — дайынсың.",
			S1T: "Күнтізбеңді қос", S1B: "Google-ды екі түртумен қос, сонда Lead Cat бос уақытыңды біледі.",
			S2T: "Мини-қосымшаны аш", S2B: "Бәрі Telegram ішінде — чаттан шықпай жоспарла, ауыстыр, растай бер.",
			Cta: "Lead Cat-ты ашу", Help: "Google Calendar мен Telegram-мен жұмыс істейді. Карта да, тырнақ та керек емес.",
			Foot: "Бұл хатты Lead Cat-қа тіркелгендіктен алдың.", Unsub: "Жазылудан бас тарту",
			Made: "Мұрт пен жылумен жасалды.", DefaultName: "дос",
		}
	default:
		return welcomeL{
			Pre: "Два касания — и первая встреча в идеальное время.", Hi: "Привет",
			Intro: "Добро пожаловать! Lead Cat вынюхивает идеальное время по календарям всех участников — без переписок и накладок. Два шага — и всё готово.",
			S1T: "Подключи календарь", S1B: "Привяжи Google в два касания, чтобы Lead Cat знал, когда ты свободен.",
			S2T: "Открой мини-приложение", S2B: "Всё живёт внутри Telegram — планируй, переноси и подтверждай, не выходя из чата.",
			Cta: "Открыть Lead Cat", Help: "Работает с Google Calendar и Telegram. Без карты и без когтей.",
			Foot: "Ты получил это письмо, потому что зарегистрировался в Lead Cat.", Unsub: "Отписаться",
			Made: "Сделано с усами и теплом.", DefaultName: "друг",
		}
	}
}

func welcomeSubject(lang string) string {
	switch NormalizeLang(lang) {
	case "en":
		return "Welcome to Lead Cat 🐾"
	case "kk":
		return "Lead Cat-қа қош келдің 🐾"
	default:
		return "Добро пожаловать в Lead Cat 🐾"
	}
}

type welcomeRenderData struct {
	Lang             string
	Name             string
	AppURL           string
	UnsubscribeURL   string
	L                welcomeL
	Button           template.HTML
}

var welcomeHTMLTemplate = template.Must(template.New("welcome").Parse(`<!DOCTYPE html>
<html lang="{{.Lang}}">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><meta name="x-apple-disable-message-reformatting"><meta name="color-scheme" content="light"><meta name="supported-color-schemes" content="light"><title>Lead Cat</title></head>
<body style="margin:0; padding:0; background-color:#F1EADF;">
<div style="display:none; max-height:0; overflow:hidden; opacity:0; mso-hide:all; font-size:1px; line-height:1px; color:#F1EADF;">{{.L.Pre}}</div>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:#F1EADF;"><tr><td align="center" style="padding:28px 12px;">
<table role="presentation" width="600" cellpadding="0" cellspacing="0" border="0" style="width:600px; max-width:600px; background-color:#FBF7F0; border-radius:18px; overflow:hidden; font-family:Georgia,'Times New Roman',serif;">
<tr><td style="background-color:#E8714C; padding:26px 32px;"><span style="font-size:22px; font-weight:bold; color:#FFFFFF; letter-spacing:0.2px;">🐾&nbsp;Lead&nbsp;Cat</span></td></tr>
<tr><td style="padding:36px 32px 8px 32px;"><h1 style="margin:0; font-size:30px; line-height:1.2; font-weight:bold; color:#3A332E;">{{.L.Hi}}, {{.Name}}!</h1></td></tr>
<tr><td style="padding:12px 32px 8px 32px;"><p style="margin:0; font-size:16px; line-height:1.6; color:#5C5249; font-family:Arial,Helvetica,sans-serif;">{{.L.Intro}}</p></td></tr>
<tr><td style="padding:18px 32px 0 32px;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0"><tr>
<td width="44" valign="top" style="padding-top:2px;"><table role="presentation" cellpadding="0" cellspacing="0" border="0"><tr><td width="34" height="34" align="center" valign="middle" style="background-color:#F6C453; border-radius:17px; font-family:Arial,Helvetica,sans-serif; font-size:16px; font-weight:bold; color:#3A332E;">1</td></tr></table></td>
<td valign="top"><p style="margin:0 0 3px 0; font-size:17px; font-weight:bold; color:#3A332E;">{{.L.S1T}}</p><p style="margin:0; font-size:15px; line-height:1.55; color:#8A7F75; font-family:Arial,Helvetica,sans-serif;">{{.L.S1B}}</p></td>
</tr></table></td></tr>
<tr><td style="padding:18px 32px 6px 32px;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0"><tr>
<td width="44" valign="top" style="padding-top:2px;"><table role="presentation" cellpadding="0" cellspacing="0" border="0"><tr><td width="34" height="34" align="center" valign="middle" style="background-color:#F6C453; border-radius:17px; font-family:Arial,Helvetica,sans-serif; font-size:16px; font-weight:bold; color:#3A332E;">2</td></tr></table></td>
<td valign="top"><p style="margin:0 0 3px 0; font-size:17px; font-weight:bold; color:#3A332E;">{{.L.S2T}}</p><p style="margin:0; font-size:15px; line-height:1.55; color:#8A7F75; font-family:Arial,Helvetica,sans-serif;">{{.L.S2B}}</p></td>
</tr></table></td></tr>
<tr><td align="center" style="padding:28px 32px 8px 32px;">
<table role="presentation" cellpadding="0" cellspacing="0" border="0"><tr><td align="center" bgcolor="#E8714C" style="border-radius:12px;">
{{.Button}}
</td></tr></table></td></tr>
<tr><td align="center" style="padding:6px 32px 36px 32px;"><p style="margin:0; font-size:13px; line-height:1.5; color:#A89C90; font-family:Arial,Helvetica,sans-serif;">{{.L.Help}}</p></td></tr>
<tr><td style="background-color:#3A332E; padding:22px 32px;">
<p style="margin:0 0 4px 0; font-size:14px; font-weight:bold; color:#FBF7F0; font-family:Arial,Helvetica,sans-serif;">🐾 Lead Cat</p>
<p style="margin:0 0 10px 0; font-size:12px; line-height:1.5; color:#B7ACA0; font-family:Arial,Helvetica,sans-serif;">{{.L.Foot}} {{.L.Made}}</p>
<p style="margin:0; font-size:12px; color:#B7ACA0; font-family:Arial,Helvetica,sans-serif;"><a href="{{.UnsubscribeURL}}" target="_blank" style="color:#F6C453; text-decoration:underline;">{{.L.Unsub}}</a></p>
</td></tr>
</table></td></tr></table></body></html>`))

var welcomeTextTemplate = texttemplate.Must(texttemplate.New("welcome.txt").Parse(
	`{{.L.Hi}}, {{.Name}}!

{{.L.Intro}}

1. {{.L.S1T}} — {{.L.S1B}}
2. {{.L.S2T}} — {{.L.S2B}}

{{.L.Cta}}: {{.AppURL}}

{{.L.Help}}

— Lead Cat. {{.L.Made}}
{{.L.Unsub}}: {{.UnsubscribeURL}}
`))

// RenderWelcome returns localized subject, plain text, and HTML bodies.
func RenderWelcome(d WelcomeData) (subject, text, html string, err error) {
	lang := NormalizeLang(d.Language)
	l := welcomeLabels(lang)
	name := d.FirstName
	if name == "" {
		name = l.DefaultName
	}
	appURL := d.AppURL
	if appURL == "" {
		appURL = "#"
	}
	unsub := d.UnsubscribeURL
	if unsub == "" {
		unsub = "#"
	}
	cta := l.Cta + " 🐾"
	rd := welcomeRenderData{
		Lang:           lang,
		Name:           name,
		AppURL:         appURL,
		UnsubscribeURL: unsub,
		L:              l,
		Button:         BulletproofButton(appURL, cta),
	}
	subject = welcomeSubject(lang)
	var tb, hb bytes.Buffer
	if err = welcomeTextTemplate.Execute(&tb, rd); err != nil {
		return "", "", "", err
	}
	if err = welcomeHTMLTemplate.Execute(&hb, rd); err != nil {
		return "", "", "", err
	}
	return subject, tb.String(), hb.String(), nil
}

// FirstNameFromDisplay extracts a greeting name from display name or email local-part.
func FirstNameFromDisplay(displayName, email string) string {
	if displayName != "" {
		if i := indexSpace(displayName); i > 0 {
			return displayName[:i]
		}
		return displayName
	}
	if email == "" {
		return ""
	}
	local := email
	if at := indexAt(email); at > 0 {
		local = email[:at]
	}
	if dot := indexDot(local); dot > 0 {
		return local[:dot]
	}
	return local
}

func indexSpace(s string) int {
	for i, r := range s {
		if r == ' ' {
			return i
		}
	}
	return -1
}

func indexAt(s string) int {
	for i, r := range s {
		if r == '@' {
			return i
		}
	}
	return -1
}

func indexDot(s string) int {
	for i, r := range s {
		if r == '.' {
			return i
		}
	}
	return -1
}
