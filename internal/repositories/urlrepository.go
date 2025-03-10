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
	SaveBatchURLs(ctx context.Context, urlObjs []*model.URLObject) error
	Ping(ctx context.Context) error
	GetShortURLByOrigin(ctx context.Context, originalURL string) (string, error)
}

type URLRepository struct {
	DB database.DBInterface
}

func NewURLRepository(db database.DBInterface) *URLRepository {
	return &URLRepository{DB: db}
}

func (r *URLRepository) SaveURL(ctx context.Context, urlObj *model.URLObject) error {
	query := `INSERT INTO urls (origin, shorten, created) 
              VALUES ($1, $2, $3) 
              ON CONFLICT (origin) DO NOTHING 
              RETURNING id`

	err := r.DB.(*database.DB).Pool.QueryRow(ctx, query, urlObj.Origin, urlObj.Shorten, time.Now()).Scan(&urlObj.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Если произошёл конфликт (уже есть такой origin), то получаем существующий shorten
			existingShortURL, lookupErr := r.GetShortURLByOrigin(ctx, urlObj.Origin)
			if lookupErr != nil {
				return fmt.Errorf("failed to fetch existing short URL: %w", lookupErr)
			}
			urlObj.Shorten = existingShortURL
			return pgx.ErrNoRows // Нам нужно дать понять обработчику, что это не новая запись
		}
		return fmt.Errorf("database insert error: %w", err)
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

func (r *URLRepository) SaveBatchURLs(ctx context.Context, urlObjs []*model.URLObject) error {
	tx, err := r.DB.(*database.DB).Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `INSERT INTO urls (origin, shorten, created) VALUES ($1, $2, $3) RETURNING id`
	for _, obj := range urlObjs {
		err := tx.QueryRow(ctx, query, obj.Origin, obj.Shorten, obj.Created).Scan(&obj.ID)
		if err != nil {
			return fmt.Errorf("failed to insert batch URLs: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *URLRepository) GetShortURLByOrigin(ctx context.Context, originalURL string) (string, error) {
	var shortURL string
	query := `SELECT shorten FROM urls WHERE origin = $1`
	err := r.DB.(*database.DB).Pool.QueryRow(ctx, query, originalURL).Scan(&shortURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("database query error: %w", err)
	}
	return shortURL, nil
}
