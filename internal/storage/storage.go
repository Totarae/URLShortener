package storage

import (
	"github.com/Totarae/URLShortener/internal/model"
)

type Storage interface {
	Save(short, original string)
	Get(short string) (string, bool)
	AppendToFile(entry model.Entry) error
}
