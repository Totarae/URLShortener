package storage

type Storage interface {
	Save(short, original string)
	Get(short string) (string, bool)
}
