package web

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/stretchr/testify/require"
)

// publicHtmlDir resolves the shipped _datafiles/html/public from this file's
// own path; the test binary's CWD is not reliable (auth_test.go chdirs).
func publicHtmlDir(t *testing.T) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(here), "..", "..", "_datafiles", "html", "public")
}

// TestViewConfigShowsNoSecret renders the real viewconfig.html through
// serveTemplate, the unauthenticated "/" handler.
func TestViewConfigShowsNoSecret(t *testing.T) {
	const sentinel = `sk-viewconfig-sentinel-93b1`
	c := configs.GetConfig()
	c.Integrations.Discord.WebhookUrl = configs.ConfigSecret(sentinel)
	c.Modules = configs.Modules{`aicompanion`: map[string]any{`APIKey`: sentinel, `RelayOrigin`: `https://keys.example.org`}}
	configs.SetConfigForTest(t, c)

	prev := httpRoot
	httpRoot = publicHtmlDir(t)
	t.Cleanup(func() { httpRoot = prev })

	req := httptest.NewRequest(http.MethodGet, `/viewconfig`, nil)
	rec := httptest.NewRecorder()
	serveTemplate(rec, req)

	body := rec.Body.String()
	require.Equal(t, http.StatusOK, rec.Code, body)
	require.Contains(t, body, `Server.MudName`, `the page must still render config rows`)
	require.NotContains(t, body, sentinel)
	require.NotContains(t, body, `Modules.aicompanion`, `/viewconfig leaves module settings out`)
}
