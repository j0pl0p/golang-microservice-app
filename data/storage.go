package data

import (
	"fmt"
	"github.com/j0pl0p/final-task-GO-YL/structures"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"time"
)

type DatabaseHost interface {
	Add(s1, s2 string) (string, error)
	GetAll() ([]structures.Expression, error)
	GetById(id string) (structures.Expression, bool)
}

// Storage Структура хранилища
type Storage struct {
	Db *sqlx.DB
}

var newDbInit = `
CREATE TABLE IF NOT EXISTS Expressions (
	id VARCHAR(256) PRIMARY KEY UNIQUE,
	expression VARCHAR(256),
	status VARCHAR(256) DEFAULT 'active',
    result FLOAT(32) DEFAULT 0.0
);

CREATE TABLE IF NOT EXISTS Daemons (
	id VARCHAR(256) PRIMARY KEY UNIQUE,
	status VARCHAR(256) DEFAULT 'active',
    last_response DATETIME
);
`

// NewStorage Создание нового хранилища
func NewStorage(path string) (*Storage, error) {
	db, err := sqlx.Connect("sqlite3", path)

	db.MustExec(newDbInit)
	if err != nil {
		return nil, fmt.Errorf("cant open database: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("cant connect to database: %w", err)
	}
	return &Storage{Db: db}, nil
}

// AddExpression Добавление выражения
func (s *Storage) AddExpression(id, exp string) (string, error) {
	addNewExpressionSQL := `INSERT INTO Expressions (id, expression) VALUES (?, ?)`
	_, err := s.Db.Exec(addNewExpressionSQL, id, exp)
	if err != nil {
		log.Println("ERROR: ", err)
		return "", err
	}
	return id, nil
}

// GetAllExpressions Получение всех выражений
func (s *Storage) GetAllExpressions() ([]structures.Expression, error) {
	var ans []structures.Expression
	getAllExpressionsSQL := `SELECT * FROM Expressions`
	res, err := s.Db.Query(getAllExpressionsSQL)
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
		ans = append(ans, structures.Expression{
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

// GetExpressionById Получение выражения по его ID
func (s *Storage) GetExpressionById(id string) (structures.Expression, bool) {
	getDataById := `SELECT id, expression, status, result FROM Expressions WHERE id=?`
	q, err := s.Db.Prepare(getDataById)
	if err != nil {
		log.Println("ERROR: ", err.Error())
		return structures.Expression{}, false
	}
	defer q.Close()
	var exp structures.Expression
	err = q.QueryRow(id).Scan(&exp.Id, &exp.Exp, &exp.Status, &exp.Result)
	if err != nil {
		log.Println("ERROR: ", err.Error())
		return structures.Expression{}, false
	}
	return exp, true
}

// AddNewDaemon Добавление нового демона
func (s *Storage) AddNewDaemon(id string) error {
	addNewDaemonSQL := `INSERT INTO Daemons (id, status, last_response) VALUES (?, 'active', ?)`
	_, err := s.Db.Exec(addNewDaemonSQL, id, time.Now())
	if err != nil {
		return err
	}
	return nil
}

// UpdateDaemonStatus Обновление статуса демона
func (s *Storage) UpdateDaemonStatus(id, newStatus string) error {
	updateDaemonSQL := `UPDATE Daemons SET status=? WHERE id=?`
	q, err := s.Db.Prepare(updateDaemonSQL)
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

// UpdateDaemonLastResponse Обновление времени последнего ответа демона
func (s *Storage) UpdateDaemonLastResponse(id string) error {
	updateDaemonSQL := `UPDATE Daemons SET last_response=? WHERE id=?`
	q, err := s.Db.Prepare(updateDaemonSQL)
	if err != nil {
		return err
	}
	defer q.Close()
	_, err = q.Exec(time.Now(), id)
	if err != nil {
		return err
	}
	return nil
}

// SaveResult Сохранение результата выражения по его ID, изменение статуса выражения
func (s *Storage) SaveResult(id string, v float32) error {
	saveResultSQL := `UPDATE Expressions SET result=?, status='done' WHERE id=?`
	q, err := s.Db.Prepare(saveResultSQL)
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

// GetDaemonsResponses Возвращает мапу Id-LastResponse демонов
func (s Storage) GetDaemonsResponses() (map[string]time.Time, error) {
	getDataSQL := `SELECT id, last_response FROM Daemons`
	q, err := s.Db.Query(getDataSQL)
	if err != nil {
		return nil, err
	}
	defer q.Close()
	ans := make(map[string]time.Time)
	for q.Next() {
		var id string
		var lp time.Time
		err := q.Scan(&id, &lp)
		if err != nil {
			return nil, err
		}
		ans[id] = lp
	}
	return ans, nil
}
