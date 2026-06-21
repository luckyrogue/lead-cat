package emailtemplates

import "html/template"

func BulletproofButton(href, label string) template.HTML {
	h := template.HTMLEscapeString(href)
	l := template.HTMLEscapeString(label)
	return template.HTML(`<!--[if mso]><v:roundrect xmlns:v="urn:schemas-microsoft-com:vml" xmlns:w="urn:schemas-microsoft-com:office:word" href="` + h + `" style="height:48px;v-text-anchor:middle;width:260px;" arcsize="25%" stroke="f" fillcolor="#E8714C"><w:anchorlock/><center style="color:#FFFFFF;font-family:Arial,Helvetica,sans-serif;font-size:16px;font-weight:bold;">` + l + `</center></v:roundrect><![endif]-->` +
		`<!--[if !mso]><!-- --><a href="` + h + `" target="_blank" style="display:inline-block; padding:15px 34px; font-family:Arial,Helvetica,sans-serif; font-size:16px; font-weight:bold; color:#FFFFFF; text-decoration:none; border-radius:12px; mso-padding-alt:0;">` + l + `</a><!--<![endif]-->`)
}
