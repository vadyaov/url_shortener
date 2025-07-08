package grpc

import (
	"context"
	"errors"

	shortener_v0 "github.com/vadyaov/url_shortener/internal/app/grpc/pkg/shortener_v0"
	"github.com/vadyaov/url_shortener/internal/service"
	"github.com/vadyaov/url_shortener/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	shortener_v0.UnimplementedShortenerV0Server
	service service.URLShortenerService
}

func NewServer(svc service.URLShortenerService) *Server {
	return &Server{
		service: svc,
	}
}

func (s *Server) GetShortUrl(ctx context.Context, req *shortener_v0.Request) (*shortener_v0.Response, error) {
	if req.GetUrl() == "" {
		return nil, status.Error(codes.InvalidArgument, "URL is required")
	}

	short, err := s.service.GetShortUrl(req.GetUrl())
	if err != nil {
		if errors.Is(err, storage.ErrDuplicateShortCode) {
			return nil, status.Error(codes.AlreadyExists, "Failed to create short URL due to conflict")
		}
		return nil, status.Error(codes.Internal, "Failed to create short URL")
	}

	return &shortener_v0.Response{Url: short}, nil
}

func (s *Server) GetOriginUrl(ctx context.Context, req *shortener_v0.Request) (*shortener_v0.Response, error) {
	if req.GetUrl() == "" {
		return nil, status.Error(codes.InvalidArgument, "URL is required")
	}

	orig, err := s.service.GetOriginUrl(req.GetUrl())
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "Short Url not found.")
		} else {
			return nil, status.Error(codes.Internal, "Failed to get original URL")
		}
	}

	return &shortener_v0.Response{Url: orig}, nil
}
