package emailtemplates

import (
	"bytes"
	"fmt"
	"html/template"
	texttemplate "text/template"
)

type MagicLinkData struct {
	Language       string
	SignInURL      string
	ExpiresMinutes int
	FirstName      string
}

type magicL struct {
	Pre, Hi, Lead, Expires, Cta, Foot, DefaultName string
}

func magicLabels(lang string) magicL {
	switch NormalizeLang(lang) {
	case "en":
		return magicL{
			Pre: "Your sign-in link for Lead Cat", Hi: "Hi", Lead: "Tap the button below to sign in to Lead Cat. This link is for you only — don't share it.",
			Expires: "This link expires in %d minutes.", Cta: "Sign in to Lead Cat",
			Foot: "If you didn't request this email, you can safely ignore it.", DefaultName: "there",
		}
	case "kk":
		return magicL{
			Pre: "Lead Cat-қа кіру сілтемесі", Hi: "Сәлем", Lead: "Lead Cat-қа кіру үшін төмендегі батырманы басың. Бұл сілтеме тек сенің — басқалармен бөліспе.",
			Expires: "Сілтеме %d минуттан кейін жарамсыз болады.", Cta: "Lead Cat-қа кіру",
			Foot: "Егер сен бұл хатты сұрамаған болсаң, елеме.", DefaultName: "дос",
		}
	default:
		return magicL{
			Pre: "Ссылка для входа в Lead Cat", Hi: "Привет", Lead: "Нажми кнопку ниже, чтобы войти в Lead Cat. Эта ссылка только для тебя — не пересылай её.",
			Expires: "Ссылка действует %d минут.", Cta: "Войти в Lead Cat",
			Foot: "Если ты не запрашивал это письмо, просто проигнорируй его.", DefaultName: "друг",
		}
	}
}

func magicSubject(lang string) string {
	switch NormalizeLang(lang) {
	case "en":
		return "Sign in to Lead Cat"
	case "kk":
		return "Lead Cat-қа кіру"
	default:
		return "Войти в Lead Cat"
	}
}

type magicRenderData struct {
	Lang           string
	Name           string
	SignInURL      string
	ExpiresMinutes int
	ExpiresLine    string
	L              magicL
	Button         template.HTML
}

var magicHTMLTemplate = template.Must(template.New("magic").Parse(`<!DOCTYPE html>
<html lang="{{.Lang}}">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><meta name="x-apple-disable-message-reformatting"><meta name="color-scheme" content="light"><meta name="supported-color-schemes" content="light"><title>Lead Cat</title></head>
<body style="margin:0; padding:0; background-color:#F1EADF;">
<div style="display:none; max-height:0; overflow:hidden; opacity:0; mso-hide:all; font-size:1px; line-height:1px; color:#F1EADF;">{{.L.Pre}}</div>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:#F1EADF;"><tr><td align="center" style="padding:28px 12px;">
<table role="presentation" width="600" cellpadding="0" cellspacing="0" border="0" style="width:600px; max-width:600px; background-color:#FBF7F0; border-radius:18px; overflow:hidden; font-family:Georgia,'Times New Roman',serif;">
<tr><td style="background-color:#E8714C; padding:26px 32px;"><span style="font-size:22px; font-weight:bold; color:#FFFFFF;">🐾&nbsp;Lead&nbsp;Cat</span></td></tr>
<tr><td style="padding:32px 32px 8px 32px;"><h1 style="margin:0; font-size:28px; line-height:1.25; font-weight:bold; color:#3A332E;">{{.L.Hi}}, {{.Name}}!</h1></td></tr>
<tr><td style="padding:8px 32px 0 32px;"><p style="margin:0; font-size:16px; line-height:1.6; color:#5C5249; font-family:Arial,Helvetica,sans-serif;">{{.L.Lead}}</p></td></tr>
<tr><td align="center" style="padding:28px 32px 8px 32px;">
<table role="presentation" cellpadding="0" cellspacing="0" border="0"><tr><td align="center" bgcolor="#E8714C" style="border-radius:12px;">
{{.Button}}
</td></tr></table></td></tr>
<tr><td align="center" style="padding:8px 32px 28px 32px;"><p style="margin:0; font-size:14px; line-height:1.5; color:#8A7F75; font-family:Arial,Helvetica,sans-serif;">{{.ExpiresLine}}</p></td></tr>
<tr><td style="background-color:#3A332E; padding:22px 32px;">
<p style="margin:0 0 4px 0; font-size:14px; font-weight:bold; color:#FBF7F0; font-family:Arial,Helvetica,sans-serif;">🐾 Lead Cat</p>
<p style="margin:0; font-size:12px; line-height:1.5; color:#B7ACA0; font-family:Arial,Helvetica,sans-serif;">{{.L.Foot}}</p>
</td></tr>
</table></td></tr></table></body></html>`))

var magicTextTemplate = texttemplate.Must(texttemplate.New("magic.txt").Parse(
	`{{.L.Hi}}, {{.Name}}!

{{.L.Lead}}

{{.L.Cta}}: {{.SignInURL}}

{{.ExpiresLine}}

{{.L.Foot}}
`))

func RenderMagicLink(d MagicLinkData) (subject, text, html string, err error) {
	lang := NormalizeLang(d.Language)
	l := magicLabels(lang)
	name := d.FirstName
	if name == "" {
		name = l.DefaultName
	}
	rd := magicRenderData{
		Lang:           lang,
		Name:           name,
		SignInURL:      d.SignInURL,
		ExpiresMinutes: d.ExpiresMinutes,
		ExpiresLine:    fmt.Sprintf(l.Expires, d.ExpiresMinutes),
		L:              l,
		Button:         BulletproofButton(d.SignInURL, l.Cta),
	}
	subject = magicSubject(lang)
	var tb, hb bytes.Buffer
	if err = magicTextTemplate.Execute(&tb, rd); err != nil {
		return "", "", "", err
	}
	if err = magicHTMLTemplate.Execute(&hb, rd); err != nil {
		return "", "", "", err
	}
	return subject, tb.String(), hb.String(), nil
}
