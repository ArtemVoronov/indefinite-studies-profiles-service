package entities

import "time"

type RegistrationToken struct {
	UserId         int
	Token          string
	ExpireAt       time.Time
	CreateDate     time.Time
	LastUpdateDate time.Time
}
