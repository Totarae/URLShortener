package v2

import (
	"context"
	"github.com/Totarae/URLShortener/internal/model"
	"github.com/Totarae/URLShortener/internal/util"
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
