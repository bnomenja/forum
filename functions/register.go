package functions

import (
	"fmt"
	"net/http"
	"text/template"

	"golang.org/x/crypto/bcrypt"
)

func (database Database) Register(w http.ResponseWriter, r *http.Request) {
	db := database.Db

	if r.URL.Path != "/login/register" {
		RenderError(w, "Page not found", 404)
		return
	}

	switch r.Method {
	case http.MethodGet:
		tmpl, err := template.ParseFiles("templates/register.html")
		if err != nil {
			fmt.Println(err)
			return
		}

		err = tmpl.Execute(w, nil)
		if err != nil {
			fmt.Println(err)
			return
		}
	case http.MethodPost:
		r.ParseForm()

		name := r.FormValue("name")
		email := r.FormValue("email")
		password := r.FormValue("password")

		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			fmt.Println(err)
			RenderError(w, "Please try later", 500)
			return
		}

		err = IsValidCredential(name, email, password)
		if err != nil {
			fmt.Println(err)
			tmpl, err := template.ParseFiles("templates/login.html")
			if err != nil {
				fmt.Println(err)
				return
			}

			err = tmpl.Execute(w, err)
			if err != nil {
				fmt.Println(err)
				return
			}
			return
		}

		_, err1 := db.Exec(queryAddUser, name, email, string(hash))
		if err1 != nil {
			tmpl, err := template.ParseFiles("templates/register.html")
			if err != nil {
				fmt.Println(err)
				return
			}

			err = tmpl.Execute(w, err1)
			fmt.Println(err)
			if err != nil {
				fmt.Println(err)
				return
			}
			return
		}

		var Id int

		err = db.QueryRow(queryGetUserIDByEmail, email).Scan(&Id)
		if err != nil {
			fmt.Println(err)
			RenderError(w, "Please try later", 500)
			return
		}

		err = SetSession(Id, db, w)
		if err != nil {
			fmt.Println(err)
			RenderError(w, "Please try later", 500)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)

	default:
		RenderError(w, "Method not allowed", 405)
	}
}
