package functions

import (
	"database/sql"
	"fmt"
	"net/http"

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
