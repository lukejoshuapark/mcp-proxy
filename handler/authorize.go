package handler

import (
	"log/slog"
	"net/http"
	"net/url"
	"time"

	hu "github.com/lukejoshuapark/mcp-proxy/httputil"
)

func (s *Server) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	responseType := q.Get("response_type")
	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")
	state := q.Get("state")
	codeChallenge := q.Get("code_challenge")
	codeChallengeMethod := q.Get("code_challenge_method")
	scope := q.Get("scope")

	if responseType != "code" {
		hu.Error(w, http.StatusBadRequest, "unsupported_response_type", "only code response_type is supported")
		return
	}

	if clientID == "" || redirectURI == "" || codeChallenge == "" {
		hu.Error(w, http.StatusBadRequest, "invalid_request", "missing required parameters")
		return
	}

	if codeChallengeMethod != "S256" {
		hu.Error(w, http.StatusBadRequest, "invalid_request", "only S256 code_challenge_method is supported")
		return
	}

	meta, err := s.getClientMetadata(clientID)
	if err != nil {
		slog.Warn("failed to fetch client metadata", "client_id", clientID, "error", err)
		hu.Error(w, http.StatusBadRequest, "invalid_client", "failed to validate client_id metadata document")
		return
	}

	if !meta.validRedirectURI(redirectURI) {
		hu.Error(w, http.StatusBadRequest, "invalid_request", "redirect_uri not in client metadata")
		return
	}

	sessionKey := hu.RandomID(16)
	s.Sessions.Set("sessions", sessionKey, AuthSession{
		ClientID:      clientID,
		RedirectURI:   redirectURI,
		State:         state,
		CodeChallenge: codeChallenge,
		CreatedAt:     time.Now(),
	})

	remoteURL, err := url.Parse(s.Config.RemoteAuthURL)
	if err != nil {
		slog.Error("failed to parse remote auth URL", "url", s.Config.RemoteAuthURL, "error", err)
		hu.Error(w, http.StatusInternalServerError, "server_error", "misconfigured remote auth URL")
		return
	}
	rq := remoteURL.Query()
	rq.Set("response_type", "code")
	rq.Set("client_id", s.Config.RemoteClientID)
	rq.Set("redirect_uri", s.Config.PublicURL+"/callback")
	rq.Set("state", sessionKey)
	if s.Config.Scopes != "" {
		rq.Set("scope", s.Config.Scopes)
	} else if scope != "" {
		rq.Set("scope", scope)
	}
	remoteURL.RawQuery = rq.Encode()

	http.Redirect(w, r, remoteURL.String(), http.StatusFound)
}
