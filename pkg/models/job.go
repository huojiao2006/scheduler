package models

import (
	"time"
)

type JobInfo struct {
	Name         string    `json:"Name"`
	Owner        string    `json:"Owner"`
	Cmd          []string  `json:"Cmd"`
	Status       string    `json:"Status"`
	StartTime    time.Time `json:"StartTime"`
	CompleteTime time.Time `json:"CompleteTime"`
}

type JobEvent struct {
	Event   string  `json:"Event"`
	JobInfo JobInfo `json:"JobInfo"`
}
