package model

import "time"

type URLObject struct {
	tableName struct{}  `pg:"urls"`
	ID        uint      `pg:"id,notnull,pk"`
	Origin    string    `pg:"origin,notnull"`
	Shorten   string    `pg:"shorten,notnull,unique"`
	Created   time.Time `pg:"created,default:now()"`
	UserID    string    `pg:"user_id"`
}
