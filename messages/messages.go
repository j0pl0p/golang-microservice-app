package messages

import (
	"bytes"
	"encoding/json"
	"time"
)

// Message Интерфейс сообщения
type Message interface {
	Imp()
}

// Task Структура задания
type Task struct {
	Id         string                   `json:"id"`
	Expression string                   `json:"expression"`
	Durations  map[string]time.Duration `json:"durations"`
}

// Beat Структура хертбита
type Beat struct {
	Id string `json:"id"`
}

// Result Структура результата
type Result struct {
	Id  string  `json:"id"`
	Res float32 `json:"res"`
}

// ToBytes Конвертация сообщения в байты
func ToBytes[T Message](message T) ([]byte, error) {
	var b bytes.Buffer
	err := json.NewEncoder(&b).Encode(message)
	return b.Bytes(), err
}

// FromBytes Конвертация байтов в сообщение
func FromBytes[T Message](b []byte) (T, error) {
	var message T
	err := json.NewDecoder(bytes.NewBuffer(b)).Decode(&message)
	return message, err
}

func (t Task) Imp()   {}
func (r Result) Imp() {}
func (b Beat) Imp()   {}
