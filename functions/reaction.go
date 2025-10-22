package functions

import (
	"fmt"
	"net/http"
)

func (database Database) Reaction(w http.ResponseWriter, r *http.Request) {
	db := database.Db

	if r.URL.Path != "/reaction/" {
		RenderError(w, errPageNotFound, 404)
		return
	}

	if r.Method != http.MethodPost {
		RenderError(w, errMethodNotAllowed, 405)
		return
	}

	userID, err := database.authenticateUser(r)
	if err != nil {
		fmt.Println("User authentication failed", err)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		fmt.Println("Failed to parse form", err)
		RenderError(w, errPleaseTryLater, 500)
		return
	}

	target := r.FormValue("target")
	id := r.FormValue("id")
	reactionType := r.FormValue("type")

	if reactionType != "like" && reactionType != "dislike" {
		fmt.Println("unknown reaction", err)
		RenderError(w, "bad request", 400)
		return
	}

	targetId := getTargetId(target, id, w, db)
	if targetId < 0 {
		return
	}

	err = HandleReaction(db, userID, targetId, target, reactionType)
	if err != nil {
		RenderError(w, "please try later", 500)
	}

	Redirect(target, targetId, w, r, db)
}
