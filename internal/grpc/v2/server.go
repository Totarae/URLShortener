package v2

import (
	"context"
	"github.com/Totarae/URLShortener/internal/model"
	"github.com/Totarae/URLShortener/internal/util"
	"go.uber.org/zap"
	"net/url"
	"time"

	pb "github.com/Totarae/URLShortener/cmd/proto_gen"
	"github.com/Totarae/URLShortener/internal/handlers"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCServer struct {
	pb.UnimplementedShortenerServiceServer
	Handler *handlers.Handler
}

func NewGRPCServer(handler *handlers.Handler) *GRPCServer {
	return &GRPCServer{Handler: handler}
}

func (s *GRPCServer) Shorten(ctx context.Context, req *pb.ShortenRequest) (*pb.ShortenResponse, error) {
	if req.GetUrl() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "URL is empty")
	}
	parsed, err := url.ParseRequestURI(req.GetUrl())
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid URL")
	}

	short := util.GenerateShortURL(req.GetUrl())

	urlObj := &model.URLObject{
		Origin:  req.GetUrl(),
		Shorten: short,
		Created: time.Now(),
		UserID:  req.GetUserId(),
	}

	if err := s.Handler.Repo.SaveURL(ctx, urlObj); err != nil {
		// Если такая URL уже существует
		return nil, status.Errorf(codes.AlreadyExists, "URL already exists")
	}

	return &pb.ShortenResponse{ShortUrl: short}, nil
}

func (s *GRPCServer) Resolve(ctx context.Context, req *pb.ResolveRequest) (*pb.ResolveResponse, error) {
	if req.ShortUrl == "" {
		return nil, status.Error(codes.InvalidArgument, "short_url is required")
	}

	h := s.Handler
	var origin string

	switch h.Mode {
	case "database":
		urlObj, err := h.Repo.GetURL(ctx, req.ShortUrl)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "db error: %v", err)
		}
		if urlObj == nil || urlObj.IsDeleted {
			return nil, status.Error(codes.NotFound, "not found")
		}
		origin = urlObj.Origin
	default:
		if val, ok := h.Store().Get(req.ShortUrl); ok {
			origin = val
		} else {
			return nil, status.Error(codes.NotFound, "not found")
		}
	}

	return &pb.ResolveResponse{OriginalUrl: origin}, nil
}

func (s *GRPCServer) BatchShorten(ctx context.Context, req *pb.BatchShortenRequest) (*pb.BatchShortenResponse, error) {
	if len(req.Urls) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no URLs provided")
	}

	results := make([]*pb.BatchShortenResult, 0, len(req.Urls))

	for _, item := range req.Urls {
		if item.OriginalUrl == "" || item.CorrelationId == "" {
			continue
		}

		short := util.GenerateShortURL(item.OriginalUrl)
		urlObj := &model.URLObject{
			Origin:  item.OriginalUrl,
			Shorten: short,
			Created: time.Now(),
			UserID:  req.UserId,
		}

		if err := s.Handler.Repo.SaveURL(ctx, urlObj); err != nil {
			continue
		}

		results = append(results, &pb.BatchShortenResult{
			CorrelationId: item.CorrelationId,
			ShortUrl:      short,
		})
	}

	return &pb.BatchShortenResponse{Items: results}, nil
}

func (s *GRPCServer) GetUserURLs(ctx context.Context, req *pb.GetUserURLsRequest) (*pb.GetUserURLsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	h := s.Handler
	var result []*pb.GetUserURLsResponseItem

	switch h.Mode {
	case "database":
		urlObjs, err := h.Repo.GetURLsByUserID(ctx, req.UserId)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to fetch URLs: %v", err)
		}

		for _, u := range urlObjs {
			if u.IsDeleted {
				continue
			}
			result = append(result, &pb.GetUserURLsResponseItem{
				ShortUrl:    u.Shorten,
				OriginalUrl: u.Origin,
			})
		}
	default:

		return nil, status.Error(codes.Unimplemented, "file mode not supported for this method")
	}

	return &pb.GetUserURLsResponse{Urls: result}, nil
}

func (s *GRPCServer) DeleteUserURLs(ctx context.Context, req *pb.DeleteUserURLsRequest) (*pb.DeleteUserURLsResponse, error) {
	if req.UserId == "" || len(req.ShortUrls) == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id and short_urls are required")
	}

	const batchSize = 100
	go func(ids []string, userID string) {
		ctx := context.Background()
		for i := 0; i < len(ids); i += batchSize {
			end := i + batchSize
			if end > len(ids) {
				end = len(ids)
			}
			batch := ids[i:end]

			if err := s.Handler.Repo.MarkURLsAsDeleted(ctx, batch, userID); err != nil {
				s.Handler.Logger.Error("Не щмогла", zap.Error(err))
			}
		}
	}(req.ShortUrls, req.UserId)

	return &pb.DeleteUserURLsResponse{Status: "accepted"}, nil
}
