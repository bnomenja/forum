package functions

import (
	"fmt"
	"net/http"
	"strings"
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

	userID, err := database.authenticateUser(r) // only authenticated user can react
	if userID == -1 {                           // something wrong happened
		RenderError(w, "please try later", 500)
		return
	}

	if err != nil { // the user is not loged
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		fmt.Println("Failed to parse form", err)
		RenderError(w, errPleaseTryLater, 500)
		return
	}

	target := strings.TrimSpace(r.FormValue("target"))
	id := strings.TrimSpace(r.FormValue("id"))
	reactionType := strings.TrimSpace(r.FormValue("type"))

	if reactionType != "like" && reactionType != "dislike" { // only like and dislike are allowed
		fmt.Println("unknown reaction", err)
		RenderError(w, "bad request", 400)
		return
	}

	targetId := getTargetId(target, id, w, db) // ID start at one
	if targetId < 1 {
		return
	}

	err = HandleReaction(db, userID, targetId, target, reactionType)
	if err != nil {
		RenderError(w, "please try later", 500)
	}

	// we redirect the user in the same page were he reacted (home page / comment page)
	Redirect(target, targetId, w, r, db)
}
