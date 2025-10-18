package functions

import (
	"net/http"
	"strings"
	"text/template"
)

func (database Database) CreateComment(w http.ResponseWriter, r *http.Request) {
	postID, err := extractPostID(r.URL.Path)
	if err != nil {
		RenderError(w, errPageNotFound, http.StatusNotFound)
		return
	}

	post, err := database.getPostWithDetails(postID)
	if err != nil {
		database.handlePostRetrievalError(w, err)
		return
	}

	switch r.Method {
	case http.MethodGet:
		database.handlePostGet(w, *post)
	case http.MethodPost:
		database.handleCommentPost(w, r, post)
	default:
		RenderError(w, errMethodNotAllowed, http.StatusMethodNotAllowed)
	}
}

func (database Database) handlePostGet(w http.ResponseWriter, post Post) {
	tmpl, err := template.ParseFiles("templates/comments.html")
	if err != nil {
		logError("failed to parse template", err)
		RenderError(w, errPleaseTryLater, 500)
		return
	}

	err = tmpl.Execute(w, post)
	if err != nil {
		logError("failed to execute template", err)
		RenderError(w, errPleaseTryLater, 500)
		return
	}
}

func (database Database) handleCommentPost(w http.ResponseWriter, r *http.Request, post *Post) {
	userID, err := database.authenticateUser(r)
	if err != nil {
		logError("User authentication failed for comment", err)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		logError("Failed to parse comment form", err)
		RenderError(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	content := strings.TrimSpace(r.FormValue("content"))
	if content == "" {
		RenderError(w, "Comment cannot be empty", http.StatusBadRequest)
		return
	}

	if err := database.insertComment(post.Id, userID, content); err != nil {
		logError("Failed to insert comment", err)
		RenderError(w, errPleaseTryLater, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
}
