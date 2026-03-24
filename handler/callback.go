package handler

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	hu "github.com/lukejoshuapark/mcp-proxy/httputil"
)

func (s *Server) HandleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	if errCode := q.Get("error"); errCode != "" {
		sessionKey := q.Get("state")
		session, ok := s.Sessions.Pop("sessions", sessionKey)
		if !ok {
			slog.Warn("callback error with unknown session", "error", errCode)
			hu.Error(w, http.StatusBadRequest, "invalid_request", "unknown or expired session")
			return
		}
		redirectURL, err := url.Parse(session.RedirectURI)
		if err != nil {
			slog.Error("failed to parse redirect_uri", "redirect_uri", session.RedirectURI, "error", err)
			hu.Error(w, http.StatusInternalServerError, "server_error", "invalid redirect_uri in session")
			return
		}
		rq := redirectURL.Query()
		rq.Set("error", errCode)
		if desc := q.Get("error_description"); desc != "" {
			rq.Set("error_description", desc)
		}
		if session.State != "" {
			rq.Set("state", session.State)
		}
		redirectURL.RawQuery = rq.Encode()
		http.Redirect(w, r, redirectURL.String(), http.StatusFound)
		return
	}

	remoteCode := q.Get("code")
	sessionKey := q.Get("state")

	if remoteCode == "" || sessionKey == "" {
		slog.Warn("callback missing code or state")
		hu.Error(w, http.StatusBadRequest, "invalid_request", "missing code or state")
		return
	}

	session, ok := s.Sessions.Pop("sessions", sessionKey)
	if !ok {
		slog.Warn("callback with unknown session", "session_key", sessionKey)
		hu.Error(w, http.StatusBadRequest, "invalid_request", "unknown or expired session")
		return
	}

	if time.Since(session.CreatedAt) > sessionTTL {
		slog.Warn("expired session", "client_id", session.ClientID, "age", time.Since(session.CreatedAt))
		hu.Error(w, http.StatusBadRequest, "invalid_request", "session has expired")
		return
	}

	tokenResp, err := s.exchangeRemoteCode(remoteCode, session.UpstreamCodeVerifier)
	if err != nil {
		slog.Error("failed to exchange remote code", "error", err)
		hu.Error(w, http.StatusBadGateway, "server_error", "failed to exchange code with remote provider")
		return
	}

	code := hu.RandomID(32)
	s.Codes.Set("codes", code, StoredCode{
		TokenResponse: tokenResp,
		CodeChallenge: session.CodeChallenge,
		ClientID:      session.ClientID,
		RedirectURI:   session.RedirectURI,
		CreatedAt:     time.Now(),
	})

	redirectURL, err := url.Parse(session.RedirectURI)
	if err != nil {
		slog.Error("failed to parse redirect_uri", "redirect_uri", session.RedirectURI, "error", err)
		hu.Error(w, http.StatusInternalServerError, "server_error", "invalid redirect_uri in session")
		return
	}
	rq := redirectURL.Query()
	rq.Set("code", code)
	if session.State != "" {
		rq.Set("state", session.State)
	}
	redirectURL.RawQuery = rq.Encode()

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

func (s *Server) exchangeRemoteCode(code, codeVerifier string) ([]byte, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {s.Config.PublicURL + "/callback"},
		"client_id":     {s.Config.RemoteClientID},
		"client_secret": {s.Config.RemoteClientSecret},
		"code_verifier": {codeVerifier},
	}

	resp, err := s.HTTPClient.PostForm(s.Config.RemoteTokenURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote token endpoint returned %d: %s", resp.StatusCode, body)
	}

	return body, nil
}
