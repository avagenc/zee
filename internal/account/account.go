package account

import (
	"errors"
	"time"
)

var ErrAccountNotLinked = errors.New("no tuya account linked to user")

type Account struct {
	OwnerID   string
	TuyaUID   string
	CreatedAt time.Time
	UpdatedAt time.Time
}
