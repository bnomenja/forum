package functions

import (
	"fmt"
	"net/http"
	"strings"
)

// Reaction handles like/dislike actions, validates user/session/CSRF, and redirects back to the source page.
func (database Database) Reaction(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/reaction/" {
		RenderError(w, errPageNotFound, 404)
		return
	}

	if r.Method != http.MethodPost {
		RenderError(w, errMethodNotAllowed, 405)
		return
	}

	storedToken, userID, err := authenticateUser(r, database.Db)
	if userID == -1 {
		RenderError(w, "please try later", 500)
		return
	}

	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if !ValidCSRF(r, storedToken) {
		RenderError(w, "Forbidden: CSRF Token Invalid", http.StatusForbidden)
		return
	}

	target := strings.TrimSpace(r.FormValue("target"))
	id := strings.TrimSpace(r.FormValue("id"))
	reactionType := strings.TrimSpace(r.FormValue("type"))

	if reactionType != "like" && reactionType != "dislike" {
		fmt.Println("unknown reaction", err)
		RenderError(w, "bad request", 400)
		return
	}

	targetId := getTargetId(target, id, w, database.Db)
	if targetId < 1 {
		return
	}

	err = HandleReaction(database.Db, userID, targetId, target, reactionType)
	if err != nil {
		RenderError(w, "please try later", 500)
	}

	Redirect(target, targetId, w, r, database.Db)
}
