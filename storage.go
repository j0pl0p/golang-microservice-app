package main

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

type Database interface {
	Add(s1, s2 string) (string, error)
}

type Storage struct {
	db *sqlx.DB
}

var newDbInit = `
CREATE TABLE IF NOT EXISTS Expressions (
	id VARCHAR(256) PRIMARY KEY,
	expression VARCHAR(256),
	status VARCHAR(256) DEFAULT 'inactive'
);

CREATE TABLE IF NOT EXISTS Agents (
	id VARCHAR(256) PRIMARY KEY,
	status VARCHAR(256)
);
`

func NewStorage(path string) (*Storage, error) {
	db, err := sqlx.Connect("sqlite3", path)

	db.MustExec(newDbInit)
	if err != nil {
		return nil, fmt.Errorf("cant open database: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("cant connect to database: %w", err)
	}
	return &Storage{db: db}, nil
}

func (s *Storage) Add(exp, id string) (string, error) {
	addNewExpressionSQL := `INSERT INTO Expressions (id, expression) VALUES (?, ?)`
	_, err := s.db.Exec(addNewExpressionSQL, exp, id)
	if err != nil {
		log.Println("ERROR: ", err)
		return "", err
	}
	return id, nil
}

func (s *Storage) GetAll() ([]Expression, error) {
	var ans []Expression
	getAllExpressionsSQL := `SELECT * FROM Expressions`
	res, err := s.db.Query(getAllExpressionsSQL)
	if err != nil {
		log.Println("ERROR: ", err)
		return nil, err
	}
	defer res.Close()
	for res.Next() {
		var id string
		var expression string
		var status string
		if err = res.Scan(&id, &expression, &status); err != nil {
			log.Println("ERROR: ", err)
			return nil, err
		}
		ans = append(ans, Expression{
			Exp:    expression,
			Id:     id,
			Status: status,
		})
	}
	if err := res.Err(); err != nil {
		log.Println("ERROR: ", err)
		return nil, err
	}
	return ans, nil
}
