package models

import (
	"time"
)

type TaskInfo struct {
	Name         string    `json:"Name"`
	Node         string    `json:"Node"`
	Status       string    `json:"Status"`
	StartTime    time.Time `json:"StartTime"`
	CompleteTime time.Time `json:"CompleteTime"`
}
