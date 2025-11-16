package functions

import (
	"net/http"
)

func (database Database) Login(w http.ResponseWriter, r *http.Request) {
	db := database.Db
	if r.URL.Path != "/login" {
		RenderError(w, "Page not found", 404)
		return
	}

	userId, err := database.authenticateUser(r)
	if userId == -1 { // something wrong happened
		RenderError(w, "please try later", 500)
		return
	}
	if err == nil { // the user is  loged -> redirect him to home
		http.Redirect(w, r, "/", http.StatusSeeOther)
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
