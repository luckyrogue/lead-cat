package emailtemplates

import (
	"bytes"
	"fmt"
	"html/template"
	texttemplate "text/template"
)

type InviteData struct {
	Language    string
	OrgName     string
	RoleLabel   string
	LoginURL    string
	InviterName string
}

type inviteL struct {
	Pre, Hi, Lead, Role, Cta, Foot, DefaultIntro string
}

func inviteLabels(lang string) inviteL {
	switch NormalizeLang(lang) {
	case "en":
		return inviteL{
			Pre: "You've been invited to a Lead Cat organization", Hi: "Hi", Lead: "You've been invited to join an organization on Lead Cat.",
			Role: "Your role", Cta: "Sign in to accept", Foot: "If you weren't expecting this invite, you can ignore this email.",
			DefaultIntro: "Someone invited you",
		}
	case "kk":
		return inviteL{
			Pre: "Lead Cat ұйымына шақыру", Hi: "Сәлем", Lead: "Lead Cat-тағы ұйымға қосылуға шақырылдың.",
			Role: "Рөлің", Cta: "Кіру және қабылдау", Foot: "Егер бұл шақыруды күтпеген болсаң, хатты елеме.",
			DefaultIntro: "Сені шақырды",
		}
	default:
		return inviteL{
			Pre: "Приглашение в организацию Lead Cat", Hi: "Привет", Lead: "Тебя пригласили в организацию в Lead Cat.",
			Role: "Твоя роль", Cta: "Войти и принять", Foot: "Если ты не ждал это приглашение, просто проигнорируй письмо.",
			DefaultIntro: "Тебя пригласили",
		}
	}
}

func inviteSubject(lang, orgName string) string {
	switch NormalizeLang(lang) {
	case "en":
		return fmt.Sprintf("Invitation to %s", orgName)
	case "kk":
		return fmt.Sprintf("%s ұйымына шақыру", orgName)
	default:
		return fmt.Sprintf("Приглашение в %s", orgName)
	}
}

type inviteRenderData struct {
	Lang        string
	OrgName     string
	RoleLabel   string
	LoginURL    string
	InviterLine string
	L           inviteL
	Button      template.HTML
}

var inviteHTMLTemplate = template.Must(template.New("invite").Parse(`<!DOCTYPE html>
<html lang="{{.Lang}}">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><meta name="x-apple-disable-message-reformatting"><meta name="color-scheme" content="light"><meta name="supported-color-schemes" content="light"><title>Lead Cat</title></head>
<body style="margin:0; padding:0; background-color:#F1EADF;">
<div style="display:none; max-height:0; overflow:hidden; opacity:0; mso-hide:all; font-size:1px; line-height:1px; color:#F1EADF;">{{.L.Pre}}</div>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:#F1EADF;"><tr><td align="center" style="padding:28px 12px;">
<table role="presentation" width="600" cellpadding="0" cellspacing="0" border="0" style="width:600px; max-width:600px; background-color:#FBF7F0; border-radius:18px; overflow:hidden; font-family:Georgia,'Times New Roman',serif;">
<tr><td style="padding:24px 32px 8px 32px;"><span style="font-size:18px; font-weight:bold; color:#E8714C;">🐾&nbsp;Lead&nbsp;Cat</span></td></tr>
<tr><td style="padding:8px 32px 4px 32px;"><h1 style="margin:0; font-size:26px; line-height:1.25; font-weight:bold; color:#3A332E;">{{.L.Hi}}!</h1></td></tr>
<tr><td style="padding:8px 32px 0 32px;"><p style="margin:0; font-size:16px; line-height:1.6; color:#5C5249; font-family:Arial,Helvetica,sans-serif;">{{.InviterLine}}</p></td></tr>
<tr><td style="padding:20px 32px 4px 32px;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:#FFFFFF; border-left:4px solid #E8714C; border-radius:10px;"><tr><td style="padding:22px 24px;">
<p style="margin:0 0 14px 0; font-size:21px; line-height:1.25; font-weight:bold; color:#3A332E;">{{.OrgName}}</p>
<table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%">
<tr><td valign="top" style="padding:4px 0; font-family:Arial,Helvetica,sans-serif; font-size:14px; color:#A89C90; width:96px;">👤&nbsp;{{.L.Role}}</td>
<td valign="top" style="padding:4px 0; font-family:Arial,Helvetica,sans-serif; font-size:15px; font-weight:bold; color:#3A332E;">{{.RoleLabel}}</td></tr>
</table></td></tr></table></td></tr>
<tr><td align="center" style="padding:24px 32px 8px 32px;">
<table role="presentation" cellpadding="0" cellspacing="0" border="0"><tr><td align="center" bgcolor="#E8714C" style="border-radius:12px;">
{{.Button}}
</td></tr></table></td></tr>
<tr><td style="background-color:#3A332E; padding:22px 32px;">
<p style="margin:0 0 4px 0; font-size:14px; font-weight:bold; color:#FBF7F0; font-family:Arial,Helvetica,sans-serif;">🐾 Lead Cat</p>
<p style="margin:0; font-size:12px; line-height:1.5; color:#B7ACA0; font-family:Arial,Helvetica,sans-serif;">{{.L.Foot}}</p>
</td></tr>
</table></td></tr></table></body></html>`))

var inviteTextTemplate = texttemplate.Must(texttemplate.New("invite.txt").Parse(
	`{{.L.Hi}}!

{{.InviterLine}}

{{.OrgName}}
{{.L.Role}}: {{.RoleLabel}}

{{.L.Cta}}: {{.LoginURL}}

{{.L.Foot}}
`))

func RoleLabelLocal(lang, role string) string {
	switch NormalizeLang(lang) {
	case "en":
		if role == "admin" {
			return "Admin"
		}
		return "Member"
	case "kk":
		if role == "admin" {
			return "Әкімші"
		}
		return "Мүше"
	default:
		if role == "admin" {
			return "Администратор"
		}
		return "Участник"
	}
}

func RenderOrgInvite(d InviteData) (subject, text, html string, err error) {
	lang := NormalizeLang(d.Language)
	l := inviteLabels(lang)
	inviterLine := l.Lead
	if d.InviterName != "" {
		switch lang {
		case "en":
			inviterLine = fmt.Sprintf("%s invited you to join %s on Lead Cat.", d.InviterName, d.OrgName)
		case "kk":
			inviterLine = fmt.Sprintf("%s сені Lead Cat-тағы «%s» ұйымына шақырды.", d.InviterName, d.OrgName)
		default:
			inviterLine = fmt.Sprintf("%s приглашает тебя в «%s» в Lead Cat.", d.InviterName, d.OrgName)
		}
	}
	rd := inviteRenderData{
		Lang:        lang,
		OrgName:     d.OrgName,
		RoleLabel:   d.RoleLabel,
		LoginURL:    d.LoginURL,
		InviterLine: inviterLine,
		L:           l,
		Button:      BulletproofButton(d.LoginURL, l.Cta),
	}
	subject = inviteSubject(lang, d.OrgName)
	var tb, hb bytes.Buffer
	if err = inviteTextTemplate.Execute(&tb, rd); err != nil {
		return "", "", "", err
	}
	if err = inviteHTMLTemplate.Execute(&hb, rd); err != nil {
		return "", "", "", err
	}
	return subject, tb.String(), hb.String(), nil
}
