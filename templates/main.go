package templates

import (
	"app/mq"
	"bytes"
	"database/sql"
	"embed"
	"fmt"
	"strings"
	tTemplate "text/template"
	"time"
)

const (
	TplAppError                 = "app_error"
	TplNewUserPasswordTokenMail = "new_user_password_token_mail"
	TplPasswordChangeTokenMail  = "password_change_token_mail"
	TplPasswordChanged          = "user_password_changed"
)

type templates map[string]*tTemplate.Template

var (
	textTemplates templates //fkey is the templatename
	htmlTemplates templates //fkey is the templatename

	//go:embed mail
	mailTemplates embed.FS

	//go:embed app
	appTemplates embed.FS
)

func InitTemplates() {
	textTemplates = make(templates)
	htmlTemplates = make(templates)
	htmlTemplates.addTemplate(TplAppError, true, appTemplates, "app/apperror.html")
	textTemplates.addTemplate(TplNewUserPasswordTokenMail, true, mailTemplates, "mail/user_password_token.txt")
	textTemplates.addTemplate(TplPasswordChangeTokenMail, true, mailTemplates, "mail/password_change_token.txt")
	textTemplates.addTemplate(TplPasswordChanged, true, mailTemplates, "mail/user_password_changed.txt")
}

func ExecuteTextTemplate(name string, data interface{}) (string, error) {
	tpl := GetTextTemplate(name)
	if tpl == nil {
		return "", fmt.Errorf("template %s does not exist", name)
	}

	var tplBuff bytes.Buffer
	if err := tpl.Execute(&tplBuff, data); err != nil {
		return "", fmt.Errorf("error executing template %s: %w", name, err)
	}
	return tplBuff.String(), nil
}

func ExecuteHTMLTemplate(name string, data interface{}) (string, error) {
	tpl := GetHTMLTemplate(name)
	if tpl == nil {
		return "", fmt.Errorf("template %s does not exist", name)
	}

	var tplBuff bytes.Buffer
	if err := tpl.Execute(&tplBuff, data); err != nil {
		return "", fmt.Errorf("error executing template %s: %w", name, err)
	}
	return tplBuff.String(), nil
}

func GetHTMLTemplate(name string) *tTemplate.Template {
	if t, ok := htmlTemplates[name]; ok {
		return t
	}
	return nil
}

func GetTextTemplate(name string) *tTemplate.Template {
	if t, ok := textTemplates[name]; ok {
		return t
	}
	return nil
}

func (t templates) addTemplate(templateName string, useDefaultDelim bool, box embed.FS, fileNames ...string) {
	delimOpen := "[#"
	delimClose := "#]"
	if useDefaultDelim {
		delimOpen = "{{"
		delimClose = "}}"
	}

	templateString := ""
	for _, file := range fileNames {
		tmpString, err := box.ReadFile(file)
		if err != nil {
			panic(err)
		}
		templateString += string(tmpString)
	}

	t[templateName] = tTemplate.Must(tTemplate.New(templateName).
		Funcs(tTemplate.FuncMap{
			"localTime": func(s interface{}) string {
				t, ok := s.(time.Time)
				if !ok {
					//check if null.Time
					tNull, okNull := s.(sql.NullTime)
					if okNull {
						ok = okNull
						t = tNull.Time
					}
				}
				if ok {
					loc, err := time.LoadLocation("Europe/Berlin")
					if err != nil {
						panic(err)
					}
					t = t.In(loc)
					if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 {
						return t.Format("02.01.2006")
					} else {
						return t.Format("02.01.2006 15:04")
					}
				}
				return fmt.Sprint(s)
			},
			"localDate": func(s interface{}) string {
				t, ok := s.(time.Time)
				if !ok {
					//check if null.Time
					tNull, okNull := s.(sql.NullTime)
					if okNull {
						ok = okNull
						t = tNull.Time
					}
				}
				if ok {
					loc, err := time.LoadLocation("Europe/Berlin")
					if err != nil {
						panic(err)
					}
					t = t.In(loc)
					return t.Format("02.01.2006")
				}
				return fmt.Sprint(s)
			},
			"euro": func(s float64) string {
				ss := strings.ReplaceAll(fmt.Sprintf("%.2f", s), ".", ",")

				return fmt.Sprint(ss)
			},
			"profileName": func(p *mq.UserProfile) string {
				if p.Forename != "" {
					return fmt.Sprintf("%s %s %s", p.Salutation, p.Forename, p.Lastname)
				} else {
					return fmt.Sprintf("%s %s", p.Salutation, p.Lastname)
				}
			},
			"centToEuro": func(p int) float32 {
				return float32(p) / 100
			},
		}).Delims(delimOpen, delimClose).Parse(templateString))
}
