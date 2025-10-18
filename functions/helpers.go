package functions

import (
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"time"
	"unicode"
)

func SetSession(userID int, db *sql.DB, w http.ResponseWriter) error {
	sessionID, err := GenerateSessionID()
	if err != nil {
		return fmt.Errorf("failed to generate session id: %v", err)
	}

	expDate := time.Now().Add(24 * time.Hour)

	_, err = db.Exec(addCookie, sessionID, userID, expDate)
	if err != nil {
		return fmt.Errorf("failed to add the session in database: %v", err)
	}

	cookie := &http.Cookie{
		Name:     "session",
		Value:    sessionID,
		Path:     "/",
		Expires:  expDate,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	}

	http.SetCookie(w, cookie)
	return nil
}

func IsValidCredential(name, email, password string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

	if !emailRegex.MatchString(email) {
		return fmt.Errorf("indalid mail")
	}

	if len(name) > 20 {
		return fmt.Errorf("invalid name(too long)")
	}

	for _, ch := range name {
		if !unicode.IsPrint(ch) {
			return fmt.Errorf("invalid name(only printable caracters are allowed)")
		}
	}

	haveNumber := false
	haveUpper := false
	havelower := false

	for _, ch := range password {
		if unicode.IsNumber(ch) {
			haveNumber = true
		}
		if unicode.IsUpper(ch) {
			haveUpper = true
		}
		if unicode.IsLower(ch) {
			havelower = true
		}

		if haveNumber && haveUpper && havelower {
			break
		}
	}

	if !haveNumber && !haveUpper && !havelower {
		return fmt.Errorf("invalid Password (must contain number, upper case and lower case character)")
	}

	return nil
}
