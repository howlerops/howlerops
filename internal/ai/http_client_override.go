package ai

import "net/http"

type httpClientSetter interface {
	setHTTPClient(*http.Client)
}

// SetHTTPClient overrides the underlying HTTP client for a provider. Primarily used in tests.
func SetHTTPClient(provider AIProvider, client *http.Client) {
	if provider == nil || client == nil {
		return
	}

	if setter, ok := provider.(httpClientSetter); ok {
		setter.setHTTPClient(client)
	}
}
