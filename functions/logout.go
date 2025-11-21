package functions

import (
	"fmt"
	"net/http"
)

func (database Database) Logout(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/logout" {
		RenderError(w, "Page not found", 404)
		return
	}

	if r.Method != http.MethodPost {
		RenderError(w, "Method not allowed", 405)
		return
	}

	// read cookie
	cookie, err := r.Cookie("session")
	if err != nil {
		// introuvable cookie
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// delete session in the database
	_, err = database.Db.Exec(Delete_Session_by_ID, cookie.Value)
	if err != nil {
		fmt.Println(err)
		RenderError(w, "please try later", 500)
		return
	}

	// delete cookie in the browser
	RemoveCookie(w)

	// redirect to home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
