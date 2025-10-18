package functions

import (
	"database/sql"
	"net/http"
	"strconv"
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
		logError("User authentication failed", err)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		logError("Failed to parse form", err)
		RenderError(w, errPleaseTryLater, 500)
		return
	}

	target := r.FormValue("target")
	id := r.FormValue("id")
	reactionType := r.FormValue("type")

	if reactionType != "like" && reactionType != "dislike" {
		logError("unknown reaction", err)
		RenderError(w, "bad request", 400)
		return
	}

	targetId := 0

	switch target {
	case "comment":
		commentId, err := strconv.Atoi(id)
		if err != nil {
			logError("not an id", err)
			RenderError(w, errPageNotFound, 404)
			return
		}

		verification := ""
		err = db.QueryRow("SELECT Content FROM Comment WHERE ID =?", commentId).Scan(&verification)
		if err != nil {
			if err == sql.ErrNoRows {
				RenderError(w, "this comment doesn't exist", 404)
				return
			}

			logError("error while confirming comment existance", err)
			RenderError(w, errPageNotFound, 404)
			return
		}

		targetId = commentId

	case "post":
		postId, err := strconv.Atoi(id)
		if err != nil {
			logError("not an id", err)
			RenderError(w, errPageNotFound, 404)
			return
		}

		verification := ""
		err = db.QueryRow("SELECT Title FROM Post WHERE ID =?", postId).Scan(&verification)
		if err != nil {
			if err == sql.ErrNoRows {
				RenderError(w, "this post doesn't exist", 404)
				return
			}

			logError("error while confirming post existance", err)
			RenderError(w, errPageNotFound, 404)
			return
		}

		targetId = postId

	default:
		logError("react to unknown", err)
		RenderError(w, "bad request", 400)
		return
	}

	err = HandleReaction(db, userID, targetId, target, reactionType)
	if err != nil {
		RenderError(w, "please try later", 500)
	}

	if target == "comment" {
		postId := 0
		db.QueryRow("SELECT Post_id FROM Comment WHERE Id = ?", targetId).Scan(&postId)
		Id := strconv.Itoa(postId)

		http.Redirect(w, r, "/posts/"+Id, http.StatusSeeOther)

		return
	} else {
		to := r.FormValue("redirect")

		if to == "home" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		} else {
			Id := strconv.Itoa(targetId)
			http.Redirect(w, r, "/posts/"+Id, http.StatusSeeOther)
		}

	}
}

func HandleReaction(db *sql.DB, userID, targetID int, target, reactionType string) error {
	isLike := (reactionType == "like")
	targetColumn := "Post_id"
	if target == "comment" {
		targetColumn = "Comment_id"
	}

	var reactionID int
	var existingLike bool

	query := "SELECT Id, Is_like FROM Reaction WHERE User_id = ? AND " + targetColumn + " = ?"
	err := db.QueryRow(query, userID, targetID).Scan(&reactionID, &existingLike)

	switch {
	case err == sql.ErrNoRows:
		insert := "INSERT INTO Reaction (User_id, " + targetColumn + ", Is_like) VALUES (?, ?, ?)"
		_, err = db.Exec(insert, userID, targetID, isLike)
		return err

	case err != nil:
		return err

	default:
		if existingLike == isLike {
			_, err = db.Exec("DELETE FROM Reaction WHERE Id = ?", reactionID)
		} else {
			_, err = db.Exec("UPDATE Reaction SET Is_like = ? WHERE Id = ?", isLike, reactionID)
		}
		return err
	}
}
