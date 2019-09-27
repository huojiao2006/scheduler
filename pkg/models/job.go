package models

import (
	"time"
)

type JobInfo struct {
	Name         string    `json:"Name"`
	Owner        string    `json:"Owner"`
	Status       string    `json:"Status"`
	StartTime    time.Time `json:"StartTime"`
	CompleteTime time.Time `json:"CompleteTime"`
}
