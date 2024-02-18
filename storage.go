package main

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

type DatabaseHost interface {
	Add(s1, s2 string) (string, error)
	GetAll() ([]Expression, error)
	GetById(id string) (Expression, bool)
}

type Storage struct {
	db *sqlx.DB
}

var newDbInit = `
CREATE TABLE IF NOT EXISTS Expressions (
	id VARCHAR(256) PRIMARY KEY UNIQUE,
	expression VARCHAR(256),
	status VARCHAR(256) DEFAULT 'inactive',
    result FLOAT(32) DEFAULT 0.0
);

CREATE TABLE IF NOT EXISTS Daemons (
	id VARCHAR(256) PRIMARY KEY UNIQUE,
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

func (s *Storage) Add(id, exp string) (string, error) {
	addNewExpressionSQL := `INSERT INTO Expressions (id, expression) VALUES (?, ?)`
	_, err := s.db.Exec(addNewExpressionSQL, id, exp)
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
		var result float32
		if err = res.Scan(&id, &expression, &status, &result); err != nil {
			log.Println("ERROR: ", err)
			return nil, err
		}
		ans = append(ans, Expression{
			Id:     id,
			Exp:    expression,
			Status: status,
			Result: result,
		})
	}
	if err := res.Err(); err != nil {
		log.Println("ERROR: ", err)
		return nil, err
	}
	return ans, nil
}

func (s *Storage) GetById(id string) (Expression, bool) {
	getDataById := `SELECT id, expression, status, result FROM Expressions WHERE id=?`
	q, err := s.db.Prepare(getDataById)
	if err != nil {
		log.Println("ERROR: ", err.Error())
		return Expression{}, false
	}
	defer q.Close()
	var exp Expression
	err = q.QueryRow(id).Scan(&exp.Id, &exp.Exp, &exp.Status, &exp.Result)
	if err != nil {
		log.Println("ERROR: ", err.Error())
		return Expression{}, false
	}
	return exp, true
}

func (s *Storage) AddNewDaemon(id string) error {
	addNewDaemonSQL := `INSERT INTO Daemons (id, status) VALUES (?, 'inactive')`
	_, err := s.db.Exec(addNewDaemonSQL, id)
	if err != nil {
		return err
	}
	return nil

}

func (s *Storage) UpdateDaemon(id, newStatus string) error {
	updateDaemonSQL := `UPDATE Daemons SET status=? WHERE id=?`
	q, err := s.db.Prepare(updateDaemonSQL)
	if err != nil {
		return err
	}
	defer q.Close()
	_, err = q.Exec(newStatus, id)
	if err != nil {
		return err
	}
	return nil
}

// SaveResult Сохранение результата выражения по его ID, изменение статуса выражения
func (s *Storage) SaveResult(id string, v float32) error {
	saveResultSQL := `UPDATE Expressions SET result=?, status='done' WHERE id=?`
	q, err := s.db.Prepare(saveResultSQL)
	if err != nil {
		return err
	}
	defer q.Close()
	_, err = q.Exec(v, id)
	if err != nil {
		return err
	}
	return nil
}
