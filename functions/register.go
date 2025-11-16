package functions

import (
	"net/http"
)

func (database Database) Register(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/register" {
		RenderError(w, "Page not found", 404)
		return
	}

	userId, err := authenticateUser(r, database.Db)
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
		ExecuteTemplate(w, "register.html", nil, 200)

	case http.MethodPost:
		HandleRegister(w, r, database.Db)

	default:
		RenderError(w, "Method not allowed", 405)
	}
}
