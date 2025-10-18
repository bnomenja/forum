package functions

import (
	"fmt"
	"net/http"
	"time"
)

func (database Database) Logout(w http.ResponseWriter, r *http.Request) {
	db := database.Db

	if r.URL.Path != "/logout" {
		RenderError(w, "Page not found", 404)
		return
	}

	if r.Method != http.MethodGet {
		RenderError(w, "Method not allowed", 405)
		return
	}

	cookie, err := r.Cookie("session")
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	_, err = db.Exec(queryDeleteSession, cookie.Value)
	if err != nil {
		fmt.Println(err)
		RenderError(w, "please try later", 500)
		return
	}

	deleteCookie := &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		Expires:  time.Now().Add(-1 * time.Hour),
		HttpOnly: true,
		Secure:   false,
	}

	http.SetCookie(w, deleteCookie)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
