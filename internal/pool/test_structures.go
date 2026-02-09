package pool

// URLRequest представляет HTTP запрос.
type URLRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    []byte
	Timeout int
}

// NewURLRequest создает новый URLRequest.
func NewURLRequest() URLRequest {
	return URLRequest{
		Headers: make(map[string]string),
		Body:    make([]byte, 0, 1024),
	}
}

// Reset сбрасывает состояние URLRequest.
// Важно: метод имеет pointer receiver!
func (r *URLRequest) Reset() {
	if r == nil {
		return
	}
	r.Method = ""
	r.URL = ""
	clear(r.Headers)
	r.Body = r.Body[:0]
	r.Timeout = 0
}

// ResetFunc для URLRequest - принимает указатель!
func URLRequestReset(req *URLRequest) {
	req.Reset()
}
