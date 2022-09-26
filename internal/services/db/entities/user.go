package entities

import (
	"time"

	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/db/entities"
)

type User struct {
	Id             int
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

func GetPossibleUserRoles() []string {
	return []string{entities.USER_ROLE_OWNER, entities.USER_ROLE_RESIDENT, entities.USER_ROLE_GI}
}
