package core

import (
	"azugo.io/core/http"
)

func (a *App) initHTTPClient() {
	if a.httpClient != nil {
		return
	}

	a.httpClient = http.NewClient(
		http.Instrumenter(a.Instrumenter()),
		http.Context(a.bgctx),
	)
}

func (a *App) HTTPClient(name ...string) (http.Client, error) {
	a.initHTTPClient()

	if len(name) > 0 {
		return a.httpClient.WithConfiguration(name[0])
	}

	return a.httpClient, nil
}
