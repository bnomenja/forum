package functions

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"syscall"
)

// RenderError renders a custom HTML error page with the given message and status code.
func RenderError(w http.ResponseWriter, msg string, code int) {

	tmpl, err := template.ParseFiles("templates/error.html")
	if err != nil {
		http.Error(w, "Template parsing error", http.StatusInternalServerError)
		return
	}

	data := ErrorPage{
		Code:    code,
		Message: msg,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(code)

	if _, err := buf.WriteTo(w); err != nil {
		if !errors.Is(err, syscall.EPIPE) {
			fmt.Println("Failed to write buffer:", err)
		}
		return
	}
}
