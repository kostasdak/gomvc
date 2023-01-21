package gomvc

import (
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// ConnectDatabase
func ConnectDatabase(user string, pass string, dbname string) (*sql.DB, error) {
	cstring := user + ":" + pass + "@/" + dbname
	db, err := sql.Open("mysql", cstring) // "user:password@/dbname" user+":"+pass+"/"+dbname
	if err != nil {
		panic(err)
	}

	// See "Important settings" section.
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	return db, err
}
