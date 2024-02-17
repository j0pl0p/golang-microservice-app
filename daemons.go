package main

type Daemon struct {
	Id     string
	Status string
}

func NewDaemon() *Daemon {
	return &Daemon{
		Id:     id, // TODO: запрос оркестратору на получение ID
		Status: "inactive",
	}
}

func (d *Daemon) UpdateStatus(newStatus string) {
	d.Status = newStatus
}
