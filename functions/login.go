package functions

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type LoginData struct {
	Message  string
	Username string
}

func (database Database) Login(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/login" {
		RenderError(w, "Page not found", 404)
		return
	}

	switch r.Method {
	case http.MethodGet:
		ExecuteTemplate(w, "login.html", nil, 200)

	case http.MethodPost:
		HandleLogin(w, r, database.Db)

	default:
		RenderError(w, "Method not allowed", 405)
		return
	}
}

func HandleLogin(w http.ResponseWriter, r *http.Request, DB *sql.DB) {
	username := strings.TrimSpace(r.FormValue("username"))
	password := strings.TrimSpace(r.FormValue("password"))
	var data LoginData

	if username == "" || password == "" {
		data.Message = "⚠️ plz commplete your identification"
		ExecuteTemplate(w, "login.html", data, 400)
		return
	}

	// Retrieve password from database by username
	var hashedPassword string
	var userID int

	err := DB.QueryRow("SELECT id, password FROM user WHERE name = ? ", username).Scan(&userID, &hashedPassword)

	if err == sql.ErrNoRows {
		data.Message = "❌ user unavailable"
		ExecuteTemplate(w, "login.html", data, http.StatusUnauthorized)
		return

	} else if err != nil {
		fmt.Println("DB query error:", err)
		RenderError(w, "something wrong happened, please try again later", 500)
		return
	}

	// check password
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		data.Username = username
		data.Message = "❌ invalid password"
		ExecuteTemplate(w, "login.html", data, http.StatusUnauthorized)
		return
	}

	// delete any sesion after now
	_, err = DB.Exec("DELETE FROM session where user_id=? ", userID)
	if err != nil {
		fmt.Println(err)
		RenderError(w, "please try later", 500)
		return
	}

	var sessionID string
	err = DB.QueryRow("SELECT id FROM session WHERE user_id = ?", userID).Scan(&sessionID)

	// we deleted all the user's previous sessions so there isn't any
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

	// 5️⃣ Redirect to Home
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
