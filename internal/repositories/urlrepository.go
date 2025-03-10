package repositories

import (
	"context"
	"errors"
	"fmt"
	"github.com/Totarae/URLShortener/internal/database"
	"github.com/Totarae/URLShortener/internal/model"
	"github.com/jackc/pgx/v5"
	"time"
)

// URLRepositoryInterface определяет методы репозитория
type URLRepositoryInterface interface {
	SaveURL(ctx context.Context, urlObj *model.URLObject) error
	GetURL(ctx context.Context, shorten string) (*model.URLObject, error)
	Ping(ctx context.Context) error
}

type URLRepository struct {
	DB database.DBInterface
}

func NewURLRepository(db database.DBInterface) *URLRepository {
	return &URLRepository{DB: db}
}

func (r *URLRepository) SaveURL(ctx context.Context, urlObj *model.URLObject) error {
	query := `INSERT INTO urls (origin, shorten, created) VALUES ($1, $2, $3) RETURNING id`
	err := r.DB.(*database.DB).Pool.QueryRow(ctx, query, urlObj.Origin, urlObj.Shorten, time.Now()).Scan(&urlObj.ID)
	if err != nil {
		return err
	}
	return nil
}

func (r *URLRepository) GetURL(ctx context.Context, shorten string) (*model.URLObject, error) {
	query := `SELECT id, origin, shorten, created FROM urls WHERE shorten = $1`
	urlObj := &model.URLObject{}
	err := r.DB.(*database.DB).Pool.QueryRow(ctx, query, shorten).Scan(
		&urlObj.ID, &urlObj.Origin, &urlObj.Shorten, &urlObj.Created,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("URL not found: %w", err)
		}
		return nil, fmt.Errorf("database error: %w", err)
	}
	return urlObj, nil
}

func (r *URLRepository) Ping(ctx context.Context) error {
	_, err := r.DB.(*database.DB).Pool.Exec(ctx, "SELECT 1")
	return err
}
