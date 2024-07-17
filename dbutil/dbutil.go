package dbutil

import (
	"database/sql"

	// _ "github.com/go-sql-driver/mysql"
	// _ "github.com/mattn/go-sqlite3"
	_ "modernc.org/sqlite"
)

type MySQLDB struct {
	*sql.DB
}

func (mdb *MySQLDB) CreateTable() error {
	createUserTableSQL := `CREATE TABLE IF NOT EXISTS user (
		"iduser" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,		
		"name" TEXT,
		"userid" TEXT UNIQUE,
		"password" TEXT,
		"account" INTEGER DEFAULT 10000,
		"age" INTEGER,
		"purpose" TEXT,
		"created" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		"updated" TIMESTAMP,
		"reward_info" TEXT,
		"myconfig" TEXT DEFAULT '{"pronunciation_correction", "grammar_correction", "listening_practice", "free_talk"}',
		"device_ids" TEXT
	);`

	stmt, err := mdb.Prepare(createUserTableSQL)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec()
	if err != nil {
		return err
	}
	return nil
}

func NewMySQLDB(dsn string) (*MySQLDB, error) {
	// db, err := sql.Open("mysql", dsn)
	db, err := sql.Open("sqlite", "file:user_biz.db")
	if err != nil {
		return nil, err
	}
	return &MySQLDB{db}, nil
}

func (mdb *MySQLDB) Close() error {
	return mdb.DB.Close()
}

func (mdb *MySQLDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return mdb.DB.Query(query, args...)
}
func (mdb *MySQLDB) QueryNoParams(query string) (*sql.Rows, error) {
	return mdb.DB.Query(query)
}

func (mdb *MySQLDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return mdb.DB.QueryRow(query, args...)
}

func (mdb *MySQLDB) Insert(query string, args ...interface{}) (int64, error) {
	result, err := mdb.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (mdb *MySQLDB) Update(query string, args ...interface{}) (int64, error) {
	result, err := mdb.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (mdb *MySQLDB) UpdateNoParams(sqlString string) (int64, error) {
	result, err := mdb.Exec(sqlString)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (mdb *MySQLDB) Delete(query string, args ...interface{}) (int64, error) {
	result, err := mdb.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (mdb *MySQLDB) Begin() (*sql.Tx, error) {
	return mdb.DB.Begin()
}
