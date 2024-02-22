package db

import (
	"time"
)

type User struct {
	Name                 string `json:"name"`
	Lastname             string `json:"lastname"`
	Username             string `json:"username"`
	password             string
	Alerts               []Alert
	SuscriptionActive    bool      `json:"suscriptionActive"`
	SuscriptionExpiresAt time.Time `json:"suscriptionExpiresAt"`
}
