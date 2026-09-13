package static

import (
	"html/template"
)

type StaticFiles struct {
	otpHTMLTemplate *template.Template
}

func InitStaticFiles() (*StaticFiles, error) {
	otpHTMLTemplate, err := template.ParseFiles("static/bmm-verification-email.html")
	if err != nil {
		return nil, err
	}

	return &StaticFiles{
		otpHTMLTemplate: otpHTMLTemplate,
	}, nil

}
