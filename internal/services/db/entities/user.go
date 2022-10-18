package entities

import (
	"time"
)

type User struct {
	Id             int
	Uuid           string
	Login          string
	Email          string
	Password       string
	Role           string
	State          string
	CreateDate     time.Time
	LastUpdateDate time.Time
}

const (
	USER_STATE_NEW       string = "NEW"
	USER_STATE_CONFRIMED string = "CONFIRMED"
	USER_STATE_BLOCKED   string = "BLOCKED"
	USER_STATE_DELETED   string = "DELETED"
)

func GetPossibleUserStates() []string {
	return []string{USER_STATE_NEW, USER_STATE_CONFRIMED, USER_STATE_BLOCKED, USER_STATE_DELETED}
}
