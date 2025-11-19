package functions

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func handleComment(w http.ResponseWriter, r *http.Request, post *Post, db *sql.DB, userID int) {
	if err := r.ParseForm(); err != nil {
		fmt.Println("Failed to parse comment form", err)
		RenderError(w, "please try later", 500)
		return
	}

	content := strings.TrimSpace(r.FormValue("content"))

	if err := isValidComment(content); err != nil {
		RenderError(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := db.Exec(queryInsertComment, post.Id, userID, content)
	if err != nil {
		fmt.Println("Failed to insert comment: %w", err)
		RenderError(w, errPleaseTryLater, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
}

// Insert the reaction in the reaction table and return an error if something wrong happened
func HandleReaction(db *sql.DB, userID, targetID int, target, reactionType string) error {
	isLike := (reactionType == "like") // check if we have a like or a dislike
	targetColumn := "post_id"          // if reacted on a post
	if target == "comment" {
		targetColumn = "comment_id" // if reacted on a comment
	}

	var reactionID int
	var existingLike bool

	query := "SELECT id, is_like FROM reaction WHERE user_id = ? AND " + targetColumn + " = ?" // get the reaction the user made before and his ID
	err := db.QueryRow(query, userID, targetID).Scan(&reactionID, &existingLike)

	switch {
	case err == sql.ErrNoRows: // the user never reacted before -> add a new reaction
		insert := "INSERT INTO reaction (user_id, " + targetColumn + ", is_like) VALUES (?, ?, ?)"
		_, err = db.Exec(insert, userID, targetID, isLike)
		return err

	case err != nil: // something wrong happened
		return err

	default: // we have the reaction type (like or dislike) and his ID

		if existingLike == isLike { // the new reaction is the same as the previous -> delete the old reaction
			_, err = db.Exec("DELETE FROM reaction WHERE id = ?", reactionID)
		} else { // the new reaction is different of the previous on -> update the reaction
			_, err = db.Exec("UPDATE reaction SET is_like = ? WHERE id = ?", isLike, reactionID)
		}

		return err
	}
}

// Get post based on the filter(created/liked/categories)
func GetFilteredPosts(db *sql.DB, categories []string, UserId int, filter, storedToken string) ([]Post, error) {
	posts := []Post{}
	var rows *sql.Rows
	var err error
	filter = strings.ToLower(strings.TrimSpace(filter))
	guest := false

	if UserId < 1 {
		guest = true
	}

	if guest && (filter == "mine" || filter == "liked") {
		return nil, errors.New("redirect")
	}

	switch filter {
	case "mine": // only get the post created by the user
		rows, err = db.Query("SELECT id FROM post WHERE user_id = ? ORDER BY created_at DESC", UserId)

	case "liked": // only get posts liked by the user. Disliked posts won't be retrieved (we can change this later if we decide to filter by reacted posts)
		rows, err = db.Query(`
			SELECT p.id
			FROM post p
			JOIN reaction r ON p.id = r.post_id
			WHERE r.user_id = ? AND r.is_like = true
			ORDER BY p.created_at DESC
		`, UserId)

	case "": // get all the posts
		rows, err = db.Query("SELECT id FROM post ORDER BY created_at DESC")
	default:
		return nil, errors.New("unknown filter")
	}

	if err != nil {
		return posts, fmt.Errorf("failed to int query for retrieving posts: %v", err)
	}

	defer rows.Close()
	allowed := map[string]bool{}

	for _, category := range categories {
		allowed[category] = true
	}

	for rows.Next() {
		var postId int

		err := rows.Scan(&postId)
		if err != nil {
			return nil, err
		}

		post, err := getPost(postId, db, UserId)
		if err != nil {
			return nil, err
		}

		if !guest {
			post.Token = storedToken
		}

		// if there is a category filter of the post is rejected ignore the post
		if len(categories) > 0 && !Wanted(allowed, post) {
			continue
		}

		posts = append(posts, *post)
	}

	return posts, nil
}

func Wanted(allowed map[string]bool, post *Post) bool {
	for _, postCategory := range post.Categories {
		if allowed[postCategory] {
			return true
		}
	}

	return false
}
