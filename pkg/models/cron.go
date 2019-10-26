package models

import (
	"time"
)

type CronInfo struct {
	Name             string    `json:"Name"`
	Owner            string    `json:"Owner"`
	Script           string    `json:"Script"`
	Cmd              []string  `json:"Cmd"`
	Status           string    `json:"Status"`
	LastScheduleTime time.Time `json:"LastScheduleTime"`
}

type CronEvent struct {
	Event    string   `json:"Event"`
	CronInfo CronInfo `json:"CronInfo"`
}
