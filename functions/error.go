package functions

import (
	"html/template"
	"net/http"
)

type ErrorPage struct {
	Code    int
	Message string
}

func RenderError(w http.ResponseWriter, msg string, code int) {
	w.WriteHeader(code)

	tmpl, err := template.ParseFiles("templates/error.html")
	if err != nil {
		http.Error(w, "Internal server error", 500)
		return
	}

	tmpl.Execute(w, ErrorPage{
		Code:    code,
		Message: msg,
	})
}
