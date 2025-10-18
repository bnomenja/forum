package functions

import (
	"net/http"
)

func (databse Database) Login(w http.ResponseWriter, r *http.Request) {
	db := databse.Db
	if r.URL.Path != "/login" {
		RenderError(w, "Page not found", 404)
		return
	}

	switch r.Method {
	case http.MethodGet:
		ExecuteTemplate(w, "login.html", nil, 200)

	case http.MethodPost:
		HandleLogin(w, r, db)

	default:
		RenderError(w, "Method not allowed", 405)
		return
	}
}
