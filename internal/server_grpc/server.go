package servergrpc

import (
	"context"
	"errors"
	"strings"

	"github.com/besapuz/urlshort/github.com/besapuz/urlshort/api/proto"
	"github.com/besapuz/urlshort/internal/app"
	"github.com/besapuz/urlshort/internal/config/db"
	"github.com/besapuz/urlshort/internal/router"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ShortenerServer реализует gRPC сервис
type ShortenerServer struct {
	proto.UnimplementedShortenerServiceServer
	shortener *router.URLShortener
	baseURL   string
}

// NewShortenerServer создает новый gRPC сервер
func NewShortenerServer(shortener *router.URLShortener, baseURL string) *ShortenerServer {
	return &ShortenerServer{
		shortener: shortener,
		baseURL:   baseURL,
	}
}

// getUserID извлекает userID из metadata (authorization header)
func (s *ShortenerServer) getUserID(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	// Ищем authorization header
	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing authorization header")
	}

	// Ожидаем формат: "Bearer <user_id>"
	parts := strings.SplitN(authHeaders[0], " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", status.Error(codes.Unauthenticated, "invalid authorization format")
	}

	return parts[1], nil
}

// ShortenURL реализует rpc ShortenURL (соответствует POST /api/shorten)
func (s *ShortenerServer) ShortenURL(ctx context.Context, req *proto.URLShortenRequest) (*proto.URLShortenResponse, error) {
	// Получаем userID из metadata
	userID, err := s.getUserID(ctx)
	if err != nil {
		return nil, err
	}

	// Валидация URL
	url := req.Url
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return nil, status.Error(codes.InvalidArgument, "invalid URL format")
	}

	// Генерируем короткий ID
	shortID := app.GenerateShortID(8)
	newUUID := uuid.New().String()

	// Сохраняем URL в зависимости от типа хранилища
	if s.shortener.UseDB {
		// Сохранение в БД
		savedShortID, err := s.shortener.DBStorage.SaveURLWithConflictCheck(ctx, newUUID, shortID, url, userID)
		if err != nil {
			if errors.Is(err, db.ErrURLConflict) {
				// URL уже существует - возвращаем существующий
				result := s.baseURL
				if !strings.HasSuffix(result, "/") {
					result += "/"
				}
				result += savedShortID
				return &proto.URLShortenResponse{Result: result}, nil
			}
			return nil, status.Error(codes.Internal, "failed to save URL")
		}
		shortID = savedShortID
	} else if s.shortener.FileStoragePath != "" {
		// Сохранение в файл с использованием пула
		mapping := router.GetURLMappingFromPool()
		defer router.PutURLMappingToPool(mapping)

		mapping.UUID = newUUID
		mapping.ShortURL = shortID
		mapping.OriginalURL = url
		mapping.UserID = userID
		mapping.DeletedFlag = false

		// Используем методы Storages
		s.shortener.MemoryStorage.SaveURLWithMapping(mapping)

		// Сохраняем в файл
		if err := s.shortener.MemoryStorage.SaveToFile(s.shortener.FileStoragePath); err != nil {
			return nil, status.Error(codes.Internal, "failed to save to file")
		}
	} else {
		// Сохранение только в память
		s.shortener.MemoryStorage.SaveURL(shortID, url, userID)
	}

	// Формируем полный URL
	result := s.baseURL
	if !strings.HasSuffix(result, "/") {
		result += "/"
	}
	result += shortID

	return &proto.URLShortenResponse{Result: result}, nil
}

// ExpandURL реализует rpc ExpandURL (соответствует GET /{id})
func (s *ShortenerServer) ExpandURL(ctx context.Context, req *proto.URLExpandRequest) (*proto.URLExpandResponse, error) {
	id := req.Id
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "empty id")
	}

	var originalURL string
	var err error

	if s.shortener.UseDB {
		// Получение из БД
		originalURL, err = s.shortener.DBStorage.GetURL(ctx, id)
		if err != nil {
			if err.Error() == "URL was deleted" {
				return nil, status.Error(codes.NotFound, "URL was deleted")
			}
			return nil, status.Error(codes.NotFound, "URL not found")
		}
	} else {
		// Получение из памяти/файла через метод Storages
		url, exists := s.shortener.MemoryStorage.GetURL(id)
		if !exists {
			return nil, status.Error(codes.NotFound, "URL not found")
		}
		originalURL = url
	}

	return &proto.URLExpandResponse{Result: originalURL}, nil
}

// ListUserURLs реализует rpc ListUserURLs (соответствует GET /api/user/urls)
func (s *ShortenerServer) ListUserURLs(ctx context.Context, _ *emptypb.Empty) (*proto.UserURLsResponse, error) {
	// Получаем userID из metadata
	userID, err := s.getUserID(ctx)
	if err != nil {
		return nil, err
	}

	response := &proto.UserURLsResponse{
		Urls: []*proto.URLData{},
	}

	if s.shortener.UseDB {
		// Получение из БД
		urls, err := s.shortener.DBStorage.GetUserURLs(ctx, userID)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to get user URLs")
		}

		for _, url := range urls {
			shortURL := s.baseURL
			if !strings.HasSuffix(shortURL, "/") {
				shortURL += "/"
			}
			shortURL += url.ShortURL

			response.Urls = append(response.Urls, &proto.URLData{
				ShortUrl:    shortURL,
				OriginalUrl: url.OriginalURL,
			})
		}
	} else if s.shortener.FileStoragePath != "" {
		// Получение из файлового хранилища
		s.shortener.MemoryStorage.MutexLock()
		defer s.shortener.MemoryStorage.MutexUnlock()

		for _, mapping := range s.shortener.MemoryStorage.GetURLMappings() {
			if mapping.UserID == userID && !mapping.DeletedFlag {
				shortURL := s.baseURL
				if !strings.HasSuffix(shortURL, "/") {
					shortURL += "/"
				}
				shortURL += mapping.ShortURL

				response.Urls = append(response.Urls, &proto.URLData{
					ShortUrl:    shortURL,
					OriginalUrl: mapping.OriginalURL,
				})
			}
		}
	} else {
		// Получение из in-memory хранилища
		shortIDs := s.shortener.MemoryStorage.GetUserURLs(userID)
		for _, shortID := range shortIDs {
			if originalURL, exists := s.shortener.MemoryStorage.GetURL(shortID); exists {
				shortURL := s.baseURL
				if !strings.HasSuffix(shortURL, "/") {
					shortURL += "/"
				}
				shortURL += shortID

				response.Urls = append(response.Urls, &proto.URLData{
					ShortUrl:    shortURL,
					OriginalUrl: originalURL,
				})
			}
		}
	}

	return response, nil
}
