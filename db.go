package gomvc

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// ConnectDatabase
func ConnectDatabase(cfg DatabaseConf) (*sql.DB, error) {
	tlsParam := ""
	if cfg.UseTLS {
		tlsParam = "&tls=true"
	}

	port := cfg.Port
	if port == 0 {
		port = 3306 // default
	}

	cstring := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true%s",
		cfg.Dbuser,
		cfg.Dbpass,
		cfg.Server,
		port,
		cfg.Dbname,
		tlsParam,
	)

	db, err := sql.Open("mysql", cstring) // "user:password@/dbname"
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// See "Important settings" section.
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	// Test connection
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	return db, nil
}

// ConnectDatabase
func ConnectDatabaseSQLite(dbname string) (*sql.DB, error) {
	//cstring := user + ":" + pass + "@/" + dbname
	db, err := sql.Open("sqlite3", dbname) // "user:password@/dbname"
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// Test the connection
	if err = db.Ping(); err != nil {
		db.Close() // Clean up on failure
		return nil, fmt.Errorf("SQLite ping failed: %w", err)
	}

	return db, err
}
