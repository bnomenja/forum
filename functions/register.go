package functions

import (
	"net/http"
)

func (database Database) Register(w http.ResponseWriter, r *http.Request) {
	db := database.Db

	if r.URL.Path != "/register" {
		RenderError(w, "Page not found", 404)
		return
	}

	switch r.Method {

	case http.MethodGet:
		ExecuteTemplate(w, "register.html", nil, 200)

	case http.MethodPost:
		HandleRegister(w, r, db)

	default:
		RenderError(w, "Method not allowed", 405)
	}
}
