package models

import ()

type Info struct {
	Key            string `json:"Key"`
	Value          []byte `json:"Value"`
	CreateRevision int64  `json:"CreateRevision"`
	ModRevision    int64  `json:"ModRevision"`
	Version        int64  `json:"Version"`
}

type Event struct {
	Event string `json:"Event"`
	Data  Info   `json:"Data"`
}
