package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
)

type clientMetadata struct {
	ClientID     string   `json:"client_id"`
	ClientName   string   `json:"client_name"`
	RedirectURIs []string `json:"redirect_uris"`
}

func fetchClientMetadata(client *http.Client, clientID string) (clientMetadata, error) {
	u, err := url.Parse(clientID)
	if err != nil {
		return clientMetadata{}, fmt.Errorf("invalid client_id URL: %w", err)
	}

	if u.Scheme != "https" {
		return clientMetadata{}, fmt.Errorf("client_id must use https scheme")
	}

	if u.Path == "" || u.Path == "/" {
		return clientMetadata{}, fmt.Errorf("client_id URL must have a path component")
	}

	resp, err := client.Get(clientID)
	if err != nil {
		return clientMetadata{}, fmt.Errorf("fetching client metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return clientMetadata{}, fmt.Errorf("client metadata returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return clientMetadata{}, fmt.Errorf("reading client metadata: %w", err)
	}

	var meta clientMetadata
	if err := json.Unmarshal(body, &meta); err != nil {
		return clientMetadata{}, fmt.Errorf("decoding client metadata: %w", err)
	}

	if meta.ClientID != clientID {
		return clientMetadata{}, fmt.Errorf("client_id in metadata %q does not match URL %q", meta.ClientID, clientID)
	}

	if len(meta.RedirectURIs) == 0 {
		return clientMetadata{}, fmt.Errorf("client metadata has no redirect_uris")
	}

	return meta, nil
}

func (m clientMetadata) validRedirectURI(uri string) bool {
	return slices.Contains(m.RedirectURIs, uri)
}
