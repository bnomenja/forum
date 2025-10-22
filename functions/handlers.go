package functions

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func HandleRegister(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	r.ParseForm()

	name := r.FormValue("name")
	email := r.FormValue("email")
	password := r.FormValue("password")

	mistakes := IsValidCredential(name, email, password)
	if mistakes != "" {
		ExecuteTemplate(w, "register.html", mistakes, 400)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println(err)
		RenderError(w, "Please try later", 500)
		return
	}

	_, err = db.Exec(queryAddUser, name, email, string(hash))
	if err != nil {
		fmt.Println(err)
		switch err.Error() {
		case "UNIQUE constraint failed: User.Name":
			ExecuteTemplate(w, "register.html", "This name is already used, please select another one", 400)

		case "UNIQUE constraint failed: User.Email":
			ExecuteTemplate(w, "register.html", "This e-mail is already used, please go to login page to sign-in", 400)

		default:
			RenderError(w, "please try later", 500)
		}

		return
	}

	var user_id int

	err = db.QueryRow(queryGetUserIDByEmail, email).Scan(&user_id)
	if err != nil {
		fmt.Println(err)
		RenderError(w, "Please try later", 500)
		return
	}

	err = SetNewSession(w, db, user_id)
	if err != nil {
		fmt.Println(err)
		RenderError(w, "Please try later", 500)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func HandleLogin(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	r.ParseForm()

	email := r.FormValue("email")
	password := r.FormValue("password")

	var storedPassword string
	var User_id int

	err := db.QueryRow("SELECT Id, Password FROM User WHERE Email =?", email).Scan(&User_id, &storedPassword)
	if err != nil {

		if err == sql.ErrNoRows {
			ExecuteTemplate(w, "login.html", "invalid credential", 400)
			return
		}

		RenderError(w, "please try later", 500)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password))
	if err != nil {
		ExecuteTemplate(w, "login.html", "invalid credential", 400)
		return
	}

	var session_id string
	err = db.QueryRow("SELECT Id FROM Session WHERE User_id = ?", User_id).Scan(&session_id)

	switch err {
	case nil:
		err := SetNewExpireDate(w, db, User_id, session_id)
		if err != nil {
			fmt.Println(err)
			RenderError(w, "please try later", 500)
			return
		}

	case sql.ErrNoRows:
		err := SetNewSession(w, db, User_id)
		if err != nil {
			fmt.Println(err)
			RenderError(w, "please try later", 500)
			return
		}

	default:
		fmt.Println(err)
		RenderError(w, "please try later", 500)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (database Database) handleComment(w http.ResponseWriter, r *http.Request, post *Post) {
	userID, err := database.authenticateUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		fmt.Println("Failed to parse comment form", err)
		RenderError(w, "please try later", 500)
		return
	}

	content := strings.TrimSpace(r.FormValue("content"))
	if !isValidInput(content) {
		RenderError(w, "Invalid comment", http.StatusBadRequest)
		return
	}

	_, err = database.Db.Exec(queryInsertComment, post.Id, userID, content)
	if err != nil {
		fmt.Println("Failed to insert comment: %w", err)
		RenderError(w, errPleaseTryLater, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
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

func GetAllPosts(db *sql.DB, categories []string, UserId int, filter string) ([]Post, error) {
	posts := []Post{}
	var rows *sql.Rows
	var err error

	switch filter {
	case "mine":
		rows, err = db.Query("SELECT * FROM Post WHERE User_id = ? ORDER BY Created_at DESC", UserId)
	case "liked":
		rows, err = db.Query(`
			SELECT p.*
			FROM Post p
			INNER JOIN Reaction r ON p.Id = r.Post_id
			WHERE r.User_id = ? AND r.Is_like = ?
			ORDER BY p.Created_at DESC
		`, UserId, true)
	default:
		rows, err = db.Query("SELECT * FROM Post ORDER BY Created_at DESC")
	}

	if err != nil {
		return posts, fmt.Errorf("failed to query posts: %v", err)
	}

	defer rows.Close()

	for rows.Next() {
		var post Post
		err := rows.Scan(&post.Id, &post.AuthorId, &post.Title, &post.Content, &post.CreationDate)
		if err != nil {
			return posts, fmt.Errorf("failed to read rows: %v", err)
		}

		if err := GetAllCategories(db, &post); err != nil {
			return posts, fmt.Errorf("failed to get categories of post %v", post.Id)
		}

		if len(categories) > 0 {
			matches := false
			for _, postCategory := range post.Categories {
				for _, filterCategory := range categories {
					if postCategory == filterCategory {
						matches = true
						break
					}
				}
				if matches {
					break
				}
			}
			if !matches {
				continue
			}
		}

		if err := GetAuthorName(db, &post); err != nil {
			return posts, fmt.Errorf("failed to get author name for post %v", post.Id)
		}

		if err := GetCommentNumber(db, &post); err != nil {
			return posts, fmt.Errorf("failed to get comment number for post %v", post.Id)
		}

		if err := GetReactionNumber(db, &post); err != nil {
			return posts, fmt.Errorf("failed to get reactions for post %v", post.Id)
		}

		posts = append(posts, post)
	}

	return posts, nil
}
