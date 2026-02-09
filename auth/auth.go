package auth

import (
	"net/http"
	"strings"
)

func ExtractAPIKey(r *http.Request, allowQuery bool) string {
	// Priority:
	// 1) X-API-Key header
	// 2) Authorization: Bearer <key>
	// 3) api_key query (optional)
	v := strings.TrimSpace(r.Header.Get("X-API-Key"))
	if v != "" {
		return v
	}

	a := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(a), "bearer ") {
		return strings.TrimSpace(a[7:])
	}

	if !allowQuery {
		return ""
	}

	q := strings.TrimSpace(r.URL.Query().Get("api_key"))
	return q
}
