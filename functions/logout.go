package functions

import (
	"fmt"
	"net/http"
)

// Logout deletes the user's session and clears the session cookie.
func (database Database) Logout(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/logout" {
		RenderError(w, "Page not found", 404)
		return
	}

	if r.Method != http.MethodPost {
		RenderError(w, "Method not allowed", 405)
		return
	}

	
	cookie, err := r.Cookie("session")
	if err != nil {
		
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	
	_, err = database.Db.Exec(Delete_Session_by_ID, cookie.Value)
	if err != nil {
		fmt.Println(err)
		RenderError(w, "please try later", 500)
		return
	}

	RemoveCookie(w)


	http.Redirect(w, r, "/", http.StatusSeeOther)
}
