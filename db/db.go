package db

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func InitDB(user, password, host, dbname string) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", user, password, host, dbname)

	var err error
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("sql.Open error: %w", err)
	}

	if err = DB.Ping(); err != nil {
		return fmt.Errorf("DB.Ping error: %w", err)
	}

	return nil
}
