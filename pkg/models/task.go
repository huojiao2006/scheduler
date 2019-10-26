package models

import (
	"time"
)

type TaskInfo struct {
	Name         string    `json:"Name"`
	Owner        string    `json:"Owner"`
	Node         string    `json:"Node"`
	Cmd          []string  `json:"Cmd"`
	Status       string    `json:"Status"`
	StartTime    time.Time `json:"StartTime"`
	CompleteTime time.Time `json:"CompleteTime"`
}
