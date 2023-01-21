package gomvc

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// AuthObject is a struct that holds all the information to perform a correct authentication against the user table in the database.
type AuthObject struct {
	Model             Model
	UsernameFieldName string
	PasswordFieldName string
	HashCodeFieldName string
	ExpTimeFieldName  string
	SessionKey        string
	ExpireAfterIdle   time.Duration
	ExtraConditions   []AuthCondition
	authURL           string
	LoggedInMessage   string
	LoginFailMessage  string
}

// AuthCondition is the struct for the ExtraConditions field in the AuthObject struct.
type AuthCondition struct {
	Field    string
	Operator string
	Value    string
}

// GetExpirationFromNow returns the expiration time from now.
func (a *AuthObject) GetExpirationFromNow() time.Time {
	return time.Now().UTC().Add(a.ExpireAfterIdle)
}

// IsSessionExpired checks authentication, get cookie value and check against user record in database
func (a *AuthObject) IsSessionExpired(r *http.Request) (bool, error) {
	if len(a.SessionKey) > 0 {
		if !Session.Exists(r.Context(), a.SessionKey) {
			// Info log
			InfoMessage("Auth Key [" + a.SessionKey + "] not exist or expired.")

			// Return [true] -> Redirect
			return true, nil
		} else {
			// Cookie is still alive
			// Info log
			InfoMessage("Auth Key [" + a.SessionKey + "] is active")
			token := Session.Get(r.Context(), a.SessionKey).(string)

			f := make([]Filter, 0)
			f = append(f, Filter{Field: a.HashCodeFieldName, Operator: "=", Value: token})
			if len(a.ExtraConditions) > 0 {
				for _, v := range a.ExtraConditions {
					f = append(f, Filter{Field: v.Field, Operator: v.Operator, Value: v.Value, Logic: "AND"})
				}
			}
			user_rr, err := a.Model.GetRecords(f, 1)
			if err != nil {
				// Return [true] + error
				return true, err
			}
			//User exists
			if len(user_rr) > 0 {
				t1 := time.Now().UTC()
				t2_indx := user_rr[0].GetFieldIndex(a.ExpTimeFieldName)
				id_indx := user_rr[0].GetFieldIndex(a.Model.PKField)
				t2 := user_rr[0].Values[t2_indx].(time.Time)
				userId := user_rr[0].Values[id_indx]

				// Compare UTC time with time in database
				if t1.After(t2) {
					// idle limit expired -> login again
					InfoMessage("Idle time expired, please sign in again")

					// Return [true] -> Redirect
					return true, nil
				}

				// Update idle value, session is not expired, user is still authenticated
				fld := make([]SQLField, 0)
				fld = append(fld, SQLField{FieldName: a.ExpTimeFieldName, Value: a.GetExpirationFromNow()})
				a.Model.Update(fld, fmt.Sprint(userId))
				return false, nil
			} else {
				InfoMessage("User not found in database, cookie value not match")
				return true, nil
			}
		}
	}

	// Info log
	InfoMessage("Auth Key not defined.")

	// Return [true] -> Redirect
	return true, nil
}

// KillAuthSession kills the auth session by reseting the expiration time in user record in database
func (a *AuthObject) KillAuthSession(w http.ResponseWriter, r *http.Request) error {
	if len(a.SessionKey) > 0 {
		if !Session.Exists(r.Context(), a.SessionKey) {
			// Info log
			InfoMessage("Auth Key [" + a.SessionKey + "] not exist or expired.")

			// Return nil
			return nil
		} else {
			token := Session.Get(r.Context(), a.SessionKey).(string)
			f := make([]Filter, 0)
			f = append(f, Filter{Field: a.HashCodeFieldName, Operator: "=", Value: token})
			user_rr, err := a.Model.GetRecords(f, 1)
			if err != nil {
				// Return error
				return err
			}
			t1 := time.Now().UTC().Add(-1)
			id_indx := user_rr[0].GetFieldIndex(a.Model.PKField)
			userId := user_rr[0].Values[id_indx]

			fld := make([]SQLField, 0)
			fld = append(fld, SQLField{FieldName: a.ExpTimeFieldName, Value: t1})
			a.Model.Update(fld, fmt.Sprint(userId))

			return nil
		}
	}
	return nil
}

// HashPassword create a password hash
func (a *AuthObject) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 8)
	return string(bytes), err
}

// CheckPasswordHash compares password and hash for Authentication using bcrypt.
func (a *AuthObject) CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// TokenGenerator is the random token generator
func (a *AuthObject) TokenGenerator() string {
	b := make([]byte, 64)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
