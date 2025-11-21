package functions

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Login handles login requests and renders the login page or processes credentials.
func (database Database) Login(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/login" {
		RenderError(w, "Page not found", 404)
		return
	}

	switch r.Method {
	case http.MethodGet:
		if len(r.URL.RawQuery) > 0 {
			RenderError(w, "Method not allowed", 405)
			return
		}

		ExecuteTemplate(w, "login.html", nil, 200)

	case http.MethodPost:
		HandleLogin(w, r, database.Db)

	default:
		RenderError(w, "Method not allowed", 405)
		return
	}
}

// HandleLogin validates user credentials, manages sessions, and logs the user in.
func HandleLogin(w http.ResponseWriter, r *http.Request, DB *sql.DB) {
	username := strings.TrimSpace(r.FormValue("username"))
	password := strings.TrimSpace(r.FormValue("password"))
	var data LoginData

	if username == "" || password == "" {
		data.Message = "⚠️ plz commplete your identification"
		ExecuteTemplate(w, "login.html", data, http.StatusBadRequest)
		return
	}


	var hashedPassword string
	var userID int

	err := DB.QueryRow(Select_UserID_and_Pw, username).Scan(&userID, &hashedPassword)

	if err == sql.ErrNoRows {
		data.Message = "❌ Invalid user or password"
		ExecuteTemplate(w, "login.html", data, http.StatusUnauthorized)
		return

	} else if err != nil {
		fmt.Println("DB query error:", err)
		RenderError(w, "something wrong happened, please try again later", 500)
		return
	}

	

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		data.Username = username
		data.Message = "❌ Invalid user or password"
		ExecuteTemplate(w, "login.html", data, http.StatusUnauthorized)
		return
	}

	
	_, err = DB.Exec(Delete_User_Session, userID)
	if err != nil {
		fmt.Println(err)
		RenderError(w, "please try later", 500)
		return
	}

	var sessionID string
	err = DB.QueryRow(Select_SessionID, userID).Scan(&sessionID)

	
	if err == sql.ErrNoRows {
		err := SetNewSession(w, DB, userID)
		if err != nil {
			fmt.Println(err)
			RenderError(w, "please try later", 500)
			return
		}

	} else {
		fmt.Println(err)
		RenderError(w, "please try later", 500)
		return
	}


	http.Redirect(w, r, "/", http.StatusSeeOther)
}
