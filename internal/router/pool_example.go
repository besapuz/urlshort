package router

import (
	"github.com/besapuz/urlshort/internal/pool"
)

// RequestContext представляет контекст запроса.
type RequestContext struct {
	UserID      string
	IP          string
	UserAgent   string
	RequestID   string
	StartTime   int64
	Params      map[string]string
	Headers     map[string][]string
	SessionData map[string]interface{}
}

// NewRequestContext создает новый RequestContext.
func NewRequestContext() RequestContext {
	return RequestContext{
		Params:      make(map[string]string),
		Headers:     make(map[string][]string),
		SessionData: make(map[string]interface{}),
	}
}

// Reset сбрасывает состояние RequestContext.
func (rc *RequestContext) Reset() {
	rc.UserID = ""
	rc.IP = ""
	rc.UserAgent = ""
	rc.RequestID = ""
	rc.StartTime = 0
	clear(rc.Params)
	clear(rc.Headers)
	clear(rc.SessionData)
}

// ShortenRequest представляет запрос на сокращение URL.
type ShortenRequest struct {
	OriginalURL string
	CustomAlias string
	UserID      string
	ExpiresAt   int64
	Tags        []string
	Metadata    map[string]string
}

// NewShortenRequest создает новый ShortenRequest.
func NewShortenRequest() ShortenRequest {
	return ShortenRequest{
		Tags:     make([]string, 0, 5),
		Metadata: make(map[string]string),
	}
}

// Reset сбрасывает состояние ShortenRequest.
func (sr *ShortenRequest) Reset() {
	sr.OriginalURL = ""
	sr.CustomAlias = ""
	sr.UserID = ""
	sr.ExpiresAt = 0
	sr.Tags = sr.Tags[:0]
	clear(sr.Metadata)
}

// RequestPool предоставляет пул для RequestContext
var (
	requestPool = pool.New(NewRequestContext, 100)

	shortenRequestPool = pool.New(NewShortenRequest, 50)
)

// GetRequestContext возвращает RequestContext из пула.
func GetRequestContext() RequestContext {
	return requestPool.Get()
}

// PutRequestContext возвращает RequestContext в пул.
func PutRequestContext(ctx RequestContext) {
	requestPool.Put(ctx)
}

// GetShortenRequest возвращает ShortenRequest из пула.
func GetShortenRequest() ShortenRequest {
	return shortenRequestPool.Get()
}

// PutShortenRequest возвращает ShortenRequest в пул.
func PutShortenRequest(req ShortenRequest) {
	shortenRequestPool.Put(req)
}

// PoolMetrics возвращает метрики пулов.
type PoolMetrics struct {
	RequestContextSize int
	ShortenRequestSize int
}

// GetPoolMetrics возвращает метрики всех пулов.
func GetPoolMetrics() PoolMetrics {
	return PoolMetrics{
		RequestContextSize: requestPool.Size(),
		ShortenRequestSize: shortenRequestPool.Size(),
	}
}
