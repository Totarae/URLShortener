package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Totarae/URLShortener/internal/model"
	"github.com/Totarae/URLShortener/internal/util"
	"go.uber.org/zap"
)

type Repository interface {
	SaveURL(ctx context.Context, urlObj *model.URLObject) error
	GetURL(ctx context.Context, short string) (*model.URLObject, error)
	MarkURLsAsDeleted(ctx context.Context, ids []string, userID string) error
	GetURLsByUserID(ctx context.Context, userID string) ([]*model.URLObject, error)
	GetStats(ctx context.Context) (urlCount int, userCount int, err error)
	Ping(ctx context.Context) error
	SaveBatchURLs(ctx context.Context, urls []*model.URLObject) error
}

type Store interface {
	Save(short, original, userID string)
	Get(short string) (string, bool)
	GetByUser(userID string) map[string]string
	MarkDeleted(shortenIDs []string, userID string)
}

type ShortenerService struct {
	Repo    Repository
	Store   Store
	Logger  *zap.Logger
	Mode    string
	BaseURL string
}

func NewShortenerService(repo Repository, store Store, logger *zap.Logger, mode, baseURL string) *ShortenerService {
	return &ShortenerService{
		Repo:    repo,
		Store:   store,
		Logger:  logger,
		Mode:    mode,
		BaseURL: baseURL,
	}
}

func (s *ShortenerService) ShortenURL(ctx context.Context, userID, originalURL string) (string, error) {
	short := util.GenerateShortURL(originalURL)
	urlObj := &model.URLObject{
		Origin:  originalURL,
		Shorten: short,
		Created: time.Now(),
		UserID:  userID,
	}

	if s.Mode == "database" {
		err := s.Repo.SaveURL(ctx, urlObj)
		return short, err
	}
	s.Store.Save(short, originalURL, userID)
	return short, nil
}

func (s *ShortenerService) ResolveURL(ctx context.Context, id string) (*model.URLObject, error) {
	if s.Mode == "memory" || s.Mode == "file" {
		original, ok := s.Store.Get(id)
		if !ok {
			return nil, nil
		}
		return &model.URLObject{Origin: original}, nil
	}

	urlObj, err := s.Repo.GetURL(ctx, id)
	if err != nil {
		return nil, err
	}
	return urlObj, nil
}

func (s *ShortenerService) BatchShorten(ctx context.Context, userID string, items []model.BatchItem) ([]model.BatchResult, error) {
	results := make([]model.BatchResult, 0, len(items))
	for _, item := range items {
		short, err := s.ShortenURL(ctx, userID, item.OriginalURL)
		if err != nil {
			return nil, err
		}
		results = append(results, model.BatchResult{
			CorrelationID: item.CorrelationID,
			ShortURL:      short,
		})
	}
	return results, nil
}

func (s *ShortenerService) DeleteURLs(ctx context.Context, userID string, ids []string) {
	if s.Mode == "database" {
		s.Repo.MarkURLsAsDeleted(ctx, ids, userID)
		return
	}
	s.Store.MarkDeleted(ids, userID)
}

func (s *ShortenerService) GetUserURLs(ctx context.Context, userID string) ([]model.BatchResult, error) {
	var results []model.BatchResult
	if s.Mode == "database" {
		urls, err := s.Repo.GetURLsByUserID(ctx, userID)
		if err != nil {
			return nil, err
		}
		for _, u := range urls {
			results = append(results, model.BatchResult{
				ShortURL:    u.Shorten,
				OriginalURL: u.Origin,
			})
		}
		return results, nil
	}
	m := s.Store.GetByUser(userID)
	for short, origin := range m {
		results = append(results, model.BatchResult{
			ShortURL:    short,
			OriginalURL: origin,
		})
	}
	return results, nil
}

func (s *ShortenerService) GetStats(ctx context.Context) (int, int, error) {
	if s.Mode != "database" {
		return 0, 0, nil // В file/memory режимах статистика недоступна
	}
	urls, users, err := s.Repo.GetStats(ctx)
	if err != nil {
		s.Logger.Error("Failed to retrieve stats", zap.Error(err))
		return 0, 0, err
	}
	return urls, users, nil
}

func (s *ShortenerService) Ping(ctx context.Context) error {
	if s.Mode != "database" {
		return nil // Ping актуален только для database
	}
	return s.Repo.Ping(ctx)
}
func (s *ShortenerService) CreateBatchShortURLs(ctx context.Context, userID string, items []model.BatchItem) ([]model.BatchResult, error) {
	results := make([]model.BatchResult, 0, len(items))
	urlObjs := make([]*model.URLObject, 0, len(items))

	for _, item := range items {
		short := util.GenerateShortURL(item.OriginalURL)
		shortURL := fmt.Sprintf("%s/%s", s.BaseURL, short)

		urlObj := &model.URLObject{
			Origin:  item.OriginalURL,
			Shorten: short,
			Created: time.Now(),
			UserID:  userID,
		}
		urlObjs = append(urlObjs, urlObj)

		results = append(results, model.BatchResult{
			CorrelationID: item.CorrelationID,
			ShortURL:      shortURL,
		})
	}

	if s.Mode == "database" {
		if err := s.Repo.SaveBatchURLs(ctx, urlObjs); err != nil {
			s.Logger.Error("failed to save batch URLs", zap.Error(err))
			return nil, err
		}
	} else {
		for _, u := range urlObjs {
			s.Store.Save(u.Shorten, u.Origin, u.UserID)
		}
	}

	return results, nil
}
