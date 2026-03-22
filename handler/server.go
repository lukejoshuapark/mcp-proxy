package handler

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/lukejoshuapark/mcp-proxy/config"
	hu "github.com/lukejoshuapark/mcp-proxy/httputil"
	"github.com/lukejoshuapark/mcp-proxy/store"
)

const (
	sessionTTL = 10 * time.Minute
	codeTTL    = 10 * time.Minute
)

type Server struct {
	Config         config.Config
	Sessions       store.Store[AuthSession]
	Codes          store.Store[StoredCode]
	Proxy          *httputil.ReverseProxy
	HTTPClient     *http.Client
	MetadataClient *http.Client
}

type AuthSession struct {
	ClientID      string    `json:"client_id"`
	RedirectURI   string    `json:"redirect_uri"`
	State         string    `json:"state"`
	CodeChallenge string    `json:"code_challenge"`
	CreatedAt     time.Time `json:"created_at"`
}

type StoredCode struct {
	TokenResponse []byte    `json:"token_response"`
	CodeChallenge string    `json:"code_challenge"`
	ClientID      string    `json:"client_id"`
	RedirectURI   string    `json:"redirect_uri"`
	CreatedAt     time.Time `json:"created_at"`
}

func NewServer(
	cfg config.Config,
	sessions store.Store[AuthSession],
	codes store.Store[StoredCode],
) (*Server, error) {
	upstream, err := url.Parse(cfg.UpstreamMCPURL)
	if err != nil {
		return nil, fmt.Errorf("invalid UPSTREAM_MCP_URL: %w", err)
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.Out.Header.Del("X-Forwarded-For")
			r.Out.Header.Del("X-Forwarded-Host")
			r.Out.Header.Del("X-Forwarded-Proto")
			r.Out.Header.Del("X-Real-IP")
			r.SetURL(upstream)
			r.Out.Host = upstream.Host
		},
		FlushInterval: -1,
	}

	return &Server{
		Config:         cfg,
		Sessions:       sessions,
		Codes:          codes,
		Proxy:          proxy,
		HTTPClient:     hu.NewHTTPClient(),
		MetadataClient: hu.NewSSRFSafeClient(),
	}, nil
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/oauth-authorization-server", s.HandleMetadata)
	mux.HandleFunc("GET /authorize", s.HandleAuthorize)
	mux.HandleFunc("GET /callback", s.HandleCallback)
	mux.HandleFunc("POST /token", s.HandleToken)
	mux.HandleFunc("/", s.HandleProxy)
	return securityHeaders(mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}
