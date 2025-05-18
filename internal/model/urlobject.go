package model

import "time"

// URLObject представляет зпись по короткой ссылке.
// Используется для хранения оригинального URL, сокращённого идентификатора,
// времени создания, принадлежности пользователю и флага удаления.
type URLObject struct {
	tableName struct{}  `pg:"urls"`
	ID        uint      `pg:"id,notnull,pk"`
	Origin    string    `pg:"origin,notnull"`
	Shorten   string    `pg:"shorten,notnull,unique"`
	Created   time.Time `pg:"created,default:now()"`
	UserID    string    `pg:"user_id"`
	IsDeleted bool      `pg:"is_deleted,default:false"`
}
