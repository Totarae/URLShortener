package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Totarae/URLShortener/internal/database"
	"github.com/Totarae/URLShortener/internal/model"
	"github.com/jackc/pgx/v5"
)

// URLRepositoryInterface определяет методы репозитория и с хранилищем URL.
type URLRepositoryInterface interface {
	SaveURL(ctx context.Context, urlObj *model.URLObject) error
	GetURL(ctx context.Context, shorten string) (*model.URLObject, error)
	SaveBatchURLs(ctx context.Context, urlObjs []*model.URLObject) error
	Ping(ctx context.Context) error
	GetShortURLByOrigin(ctx context.Context, originalURL string) (string, error)
	GetURLsByUserID(ctx context.Context, userID string) ([]*model.URLObject, error)
	MarkURLsAsDeleted(ctx context.Context, ids []string, userID string) error
	CountURLs(ctx context.Context) (int, error)
	CountUsers(ctx context.Context) (int, error)
	GetStats(ctx context.Context) (urlCount int, userCount int, err error)
}

// URLRepository реализует URLRepositoryInterface с использованием PostgreSQL.
type URLRepository struct {
	DB database.DBInterface
}

func (r *URLRepository) GetStats(ctx context.Context) (urlCount int, userCount int, err error) {
	urls, err := r.CountURLs(ctx)
	if err != nil {
		return 0, 0, err
	}
	users, err := r.CountUsers(ctx)
	if err != nil {
		return 0, 0, err
	}
	return urls, users, nil
}

// NewURLRepository создаёт новый экземпляр URLRepository.
func NewURLRepository(db database.DBInterface) *URLRepository {
	return &URLRepository{DB: db}
}

// SaveURL сохраняет объект URL в базу данных.
// Если origin уже существует, возвращает существующий shorten.
func (r *URLRepository) SaveURL(ctx context.Context, urlObj *model.URLObject) error {
	query := `INSERT INTO urls (origin, shorten, created, user_id) 
              VALUES ($1, $2, $3, $4) 
              ON CONFLICT (origin) DO NOTHING 
              RETURNING id`

	err := r.DB.(*database.DB).Pool.QueryRow(ctx, query, urlObj.Origin, urlObj.Shorten, time.Now(), urlObj.UserID).Scan(&urlObj.ID)
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

// GetURL извлекает оригинальный URL по сокращённому идентификатору.
func (r *URLRepository) GetURL(ctx context.Context, shorten string) (*model.URLObject, error) {
	query := `SELECT id, origin, shorten, created, is_deleted  FROM urls WHERE shorten = $1`
	urlObj := &model.URLObject{}
	err := r.DB.(*database.DB).Pool.QueryRow(ctx, query, shorten).Scan(
		&urlObj.ID, &urlObj.Origin, &urlObj.Shorten, &urlObj.Created, &urlObj.IsDeleted,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("URL not found: %w", err)
		}
		return nil, fmt.Errorf("database error: %w", err)
	}
	return urlObj, nil
}

// Ping проверяет доступность базы данных.
func (r *URLRepository) Ping(ctx context.Context) error {
	_, err := r.DB.(*database.DB).Pool.Exec(ctx, "SELECT 1")
	return err
}

// SaveBatchURLs сохраняет список URL-объектов в базе данных в рамках транзакции.
func (r *URLRepository) SaveBatchURLs(ctx context.Context, urlObjs []*model.URLObject) error {
	tx, err := r.DB.(*database.DB).Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `INSERT INTO urls (origin, shorten, created, user_id) VALUES ($1, $2, $3, $4) RETURNING id`
	for _, obj := range urlObjs {
		err := tx.QueryRow(ctx, query, obj.Origin, obj.Shorten, obj.Created, obj.UserID).Scan(&obj.ID)
		if err != nil {
			return fmt.Errorf("failed to insert batch URLs: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetShortURLByOrigin возвращает сокращённый URL по оригинальному.
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

// GetURLsByUserID возвращает все сокращённые ссылки пользователя.
func (r *URLRepository) GetURLsByUserID(ctx context.Context, userID string) ([]*model.URLObject, error) {
	query := `SELECT id, origin, shorten, created FROM urls WHERE user_id = $1`
	rows, err := r.DB.(*database.DB).Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query URLs by user: %w", err)
	}
	defer rows.Close()

	var results []*model.URLObject
	for rows.Next() {
		obj := &model.URLObject{}
		err := rows.Scan(&obj.ID, &obj.Origin, &obj.Shorten, &obj.Created)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		results = append(results, obj)
	}

	return results, nil
}

// MarkURLsAsDeleted помечает указанные ссылки как удалённые.
func (r *URLRepository) MarkURLsAsDeleted(ctx context.Context, ids []string, userID string) error {
	if len(ids) == 0 {
		return nil
	}

	// Подготавливаем SQL для batch-обновления
	query := `
		UPDATE urls 
		SET is_deleted = TRUE 
		WHERE shorten = ANY($1) AND user_id = $2
	`
	_, err := r.DB.(*database.DB).Pool.Exec(ctx, query, ids, userID)
	if err != nil {
		return fmt.Errorf("failed to mark URLs as deleted: %w", err)
	}
	return nil
}

// CountURLs количество сокращенных ссылок
func (r *URLRepository) CountURLs(ctx context.Context) (int, error) {
	var count int
	err := r.DB.(*database.DB).Pool.QueryRow(ctx, "SELECT COUNT(*) FROM urls WHERE is_deleted = false").Scan(&count)
	return count, err
}

// CountUsers количество пользователей
func (r *URLRepository) CountUsers(ctx context.Context) (int, error) {
	var count int
	err := r.DB.(*database.DB).Pool.QueryRow(ctx, "SELECT COUNT(DISTINCT user_id) FROM urls WHERE user_id IS NOT NULL").Scan(&count)
	return count, err
}
