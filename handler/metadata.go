package handler

import (
	"net/http"

	hu "github.com/lukejoshuapark/mcp-proxy/httputil"
)

func (s *Server) HandleMetadata(w http.ResponseWriter, r *http.Request) {
	base := s.Config.PublicURL
	hu.JSON(w, http.StatusOK, map[string]any{
		"issuer":                                base,
		"authorization_endpoint":                base + "/authorize",
		"token_endpoint":                        base + "/token",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"client_id_metadata_document_supported": true,
	})
}
