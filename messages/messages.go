package messages

import (
	"bytes"
	"encoding/json"
	"time"
)

type Message interface {
	Imp()
}

type Task struct {
	Id         string                   `json:"id"`
	Expression string                   `json:"expression"`
	Durations  map[string]time.Duration `json:"durations"`
}

type Result struct {
	Id  string  `json:"id"`
	Res float32 `json:"res"`
}

func ToBytes[T Message](messsage T) ([]byte, error) {
	var b bytes.Buffer
	err := json.NewEncoder(&b).Encode(messsage)
	return b.Bytes(), err
}

func FromBytes[T Message](b []byte) (T, error) {
	var message T
	err := json.NewDecoder(bytes.NewBuffer(b)).Decode(&message)
	return message, err
}

func (t Task) Imp()   {}
func (r Result) Imp() {}
