package functions

import (
	"database/sql"
	"fmt"
	"net/http"
	"text/template"

	"golang.org/x/crypto/bcrypt"
)

func (databse Database) Login(w http.ResponseWriter, r *http.Request) {
	db := databse.Db
	if r.URL.Path != "/login" {
		RenderError(w, "Page not found", 404)
		return
	}

	switch r.Method {
	case http.MethodPost:
		r.ParseForm()

		email := r.FormValue("email")
		password := r.FormValue("password")

		var storedPassword string
		var User_id int

		err := db.QueryRow("SELECT Id, Password FROM User WHERE Email =?", email).Scan(&User_id, &storedPassword)
		if err != nil {
			if err == sql.ErrNoRows {
				tmpl, err := template.ParseFiles("templates/login.html")
				if err != nil {
					fmt.Println(err)
					return
				}

				err = tmpl.Execute(w, "invalid credential")
				if err != nil {
					fmt.Println(err)
					return
				}
				return
			}

			RenderError(w, "please try later", 500)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password))
		if err != nil {
			tmpl, err := template.ParseFiles("templates/login.html")
			if err != nil {
				fmt.Println(err)
				return
			}

			err = tmpl.Execute(w, "invalid credential")
			if err != nil {
				fmt.Println(err)
				return
			}
			return

		}

		var Session_ID string
		err = db.QueryRow("SELECT Id FROM Session WHERE User_id = ?", User_id).Scan(&Session_ID)

		switch err {
		case nil:
			newExp, err := SetNewExpireDate(db, User_id)
			if err != nil {
				fmt.Println(err)
				RenderError(w, "please try later", 500)
				return
			}

			cookie := &http.Cookie{
				Name:     "session",
				Value:    Session_ID,
				Path:     "/",
				Expires:  newExp,
				HttpOnly: true,
				Secure:   false,
			}

			http.SetCookie(w, cookie)

		case sql.ErrNoRows:
			err := SetSession(User_id, db, w)
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
	case http.MethodGet:
		tmpl, err := template.ParseFiles("templates/login.html")
		if err != nil {
			fmt.Println(err)
			return
		}

		err = tmpl.Execute(w, nil)
		if err != nil {
			fmt.Println(err)
			return
		}

		return
	default:
		RenderError(w, "Method not allowed", 405)
		return
	}
}
