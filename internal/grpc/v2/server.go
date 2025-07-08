package v2

import (
	"context"
	"github.com/Totarae/URLShortener/internal/model"
	pb "github.com/Totarae/URLShortener/internal/pkg/proto_gen"
	"github.com/Totarae/URLShortener/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/url"
)

type GRPCServer struct {
	pb.UnimplementedShortenerServiceServer
	Service *service.ShortenerService
}

func NewGRPCServer(svc *service.ShortenerService) *GRPCServer {
	return &GRPCServer{Service: svc}
}

func (s *GRPCServer) Shorten(ctx context.Context, req *pb.ShortenRequest) (*pb.ShortenResponse, error) {
	if req.GetUrl() == "" {

		return nil, status.Errorf(codes.InvalidArgument, "URL is empty")
	}
	parsed, err := url.ParseRequestURI(req.GetUrl())
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid URL")
	}

	short, err := s.Service.ShortenURL(ctx, req.GetUserId(), req.GetUrl())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "shorten failed: %v", err)
	}

	return &pb.ShortenResponse{ShortUrl: short}, nil
}

func (s *GRPCServer) Resolve(ctx context.Context, req *pb.ResolveRequest) (*pb.ResolveResponse, error) {
	if req.ShortUrl == "" {
		return nil, status.Error(codes.InvalidArgument, "short_url is required")
	}

	urlObj, err := s.Service.ResolveURL(ctx, req.ShortUrl)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "resolve failed: %v", err)
	}
	if urlObj == nil || urlObj.IsDeleted {
		return nil, status.Error(codes.NotFound, "not found")
	}
	return &pb.ResolveResponse{OriginalUrl: urlObj.Origin}, nil
}

func (s *GRPCServer) BatchShorten(ctx context.Context, req *pb.BatchShortenRequest) (*pb.BatchShortenResponse, error) {
	if len(req.Urls) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no URLs provided")
	}

	items := make([]model.BatchItem, 0, len(req.Urls))
	for _, item := range req.Urls {
		if item.OriginalUrl == "" || item.CorrelationId == "" {
			continue
		}
		items = append(items, model.BatchItem{
			CorrelationID: item.CorrelationId,
			OriginalURL:   item.OriginalUrl,
		})
	}
	results, err := s.Service.BatchShorten(ctx, req.UserId, items)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "batch shorten failed: %v", err)
	}
	resp := make([]*pb.BatchShortenResult, 0, len(results))
	for _, r := range results {
		resp = append(resp, &pb.BatchShortenResult{
			CorrelationId: r.CorrelationID,
			ShortUrl:      r.ShortURL,
		})
	}
	return &pb.BatchShortenResponse{Items: resp}, nil
}

func (s *GRPCServer) GetUserURLs(ctx context.Context, req *pb.GetUserURLsRequest) (*pb.GetUserURLsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	results, err := s.Service.GetUserURLs(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get user urls failed: %v", err)
	}
	items := make([]*pb.GetUserURLsResponseItem, 0, len(results))
	for _, r := range results {
		items = append(items, &pb.GetUserURLsResponseItem{
			ShortUrl:    r.ShortURL,
			OriginalUrl: r.OriginalURL,
		})
	}
	return &pb.GetUserURLsResponse{Urls: items}, nil
}

func (s *GRPCServer) DeleteUserURLs(ctx context.Context, req *pb.DeleteUserURLsRequest) (*pb.DeleteUserURLsResponse, error) {
	if req.UserId == "" || len(req.ShortUrls) == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id and short_urls are required")
	}

	go s.Service.DeleteURLs(context.Background(), req.UserId, req.ShortUrls)
	return &pb.DeleteUserURLsResponse{Status: "accepted"}, nil
}
