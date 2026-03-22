package handler

import (
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	hu "github.com/lukejoshuapark/mcp-proxy/httputil"
)

func (s *Server) HandleToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		hu.Error(w, http.StatusBadRequest, "invalid_request", "malformed form body")
		return
	}

	switch r.FormValue("grant_type") {
	case "authorization_code":
		s.handleAuthorizationCode(w, r)
	case "refresh_token":
		s.handleRefreshToken(w, r)
	default:
		hu.Error(w, http.StatusBadRequest, "unsupported_grant_type", "only authorization_code and refresh_token are supported")
	}
}

func (s *Server) handleAuthorizationCode(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	codeVerifier := r.FormValue("code_verifier")
	clientID := r.FormValue("client_id")
	redirectURI := r.FormValue("redirect_uri")

	if code == "" || codeVerifier == "" || clientID == "" {
		hu.Error(w, http.StatusBadRequest, "invalid_request", "missing required parameters")
		return
	}

	stored, ok := s.Codes.Pop("codes", code)
	if !ok {
		slog.Warn("unknown or expired authorization code")
		hu.Error(w, http.StatusBadRequest, "invalid_grant", "unknown or expired authorization code")
		return
	}

	if time.Since(stored.CreatedAt) > codeTTL {
		slog.Warn("expired authorization code", "client_id", stored.ClientID, "age", time.Since(stored.CreatedAt))
		hu.Error(w, http.StatusBadRequest, "invalid_grant", "authorization code has expired")
		return
	}

	if stored.ClientID != clientID {
		slog.Warn("client_id mismatch on token exchange", "expected", stored.ClientID, "got", clientID)
		hu.Error(w, http.StatusBadRequest, "invalid_grant", "client_id mismatch")
		return
	}

	if stored.RedirectURI != redirectURI {
		slog.Warn("redirect_uri mismatch on token exchange", "expected", stored.RedirectURI, "got", redirectURI)
		hu.Error(w, http.StatusBadRequest, "invalid_grant", "redirect_uri mismatch")
		return
	}

	if !hu.VerifyPKCE(codeVerifier, stored.CodeChallenge) {
		slog.Warn("PKCE verification failed", "client_id", clientID)
		hu.Error(w, http.StatusBadRequest, "invalid_grant", "PKCE verification failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	w.Write(stored.TokenResponse)
}

func (s *Server) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken := r.FormValue("refresh_token")
	if refreshToken == "" {
		hu.Error(w, http.StatusBadRequest, "invalid_request", "missing refresh_token")
		return
	}

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {s.Config.RemoteClientID},
		"client_secret": {s.Config.RemoteClientSecret},
	}

	resp, err := s.HTTPClient.PostForm(s.Config.RemoteTokenURL, data)
	if err != nil {
		slog.Error("failed to refresh token with remote provider", "error", err)
		hu.Error(w, http.StatusBadGateway, "server_error", "failed to refresh token with remote provider")
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, io.LimitReader(resp.Body, 1<<20))
}
