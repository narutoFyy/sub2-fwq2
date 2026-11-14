//go:build unit

package routes

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newDownloadsRouter(t *testing.T, dir string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterDownloadRoutes(r, dir)
	return r
}

func getDownload(r *gin.Engine, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

func TestDownloadRoutes(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "cc-switch"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "cc-switch", "manifest.json"), []byte(`{"version":"v1"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "cc-switch", "CC-Switch-Windows.msi"), []byte("binary"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "cc-switch", ".secret"), []byte("hidden"), 0o644))
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "leak.txt"), []byte("leak"), 0o644))
	require.NoError(t, os.Symlink(outside, filepath.Join(root, "link")))

	r := newDownloadsRouter(t, root)

	t.Run("serves the manifest uncached", func(t *testing.T) {
		w := getDownload(r, "/downloads/cc-switch/manifest.json")
		require.Equal(t, http.StatusOK, w.Code)
		require.JSONEq(t, `{"version":"v1"}`, w.Body.String())
		require.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	})

	t.Run("answers HEAD for download managers", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodHead, "/downloads/cc-switch/CC-Switch-Windows.msi", nil))
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "6", w.Header().Get("Content-Length"))
		require.Empty(t, w.Body.String())
	})

	t.Run("serves installers as attachments", func(t *testing.T) {
		w := getDownload(r, "/downloads/cc-switch/CC-Switch-Windows.msi")
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "binary", w.Body.String())
		require.Contains(t, w.Header().Get("Content-Disposition"), `filename="CC-Switch-Windows.msi"`)
	})

	for name, path := range map[string]string{
		"missing file":      "/downloads/cc-switch/nope.msi",
		"directory listing": "/downloads/cc-switch/",
		"downloads root":    "/downloads/",
		"hidden file":       "/downloads/cc-switch/.secret",
		"dot-dot traversal": "/downloads/../../etc/passwd",
		"encoded traversal": "/downloads/%2e%2e/%2e%2e/etc/passwd",
		"symlink escape":    "/downloads/link/leak.txt",
	} {
		t.Run("404 for "+name, func(t *testing.T) {
			require.Equal(t, http.StatusNotFound, getDownload(r, path).Code)
		})
	}

	t.Run("disabled when no directory is configured", func(t *testing.T) {
		off := newDownloadsRouter(t, "  ")
		require.Equal(t, http.StatusNotFound, getDownload(off, "/downloads/cc-switch/manifest.json").Code)
	})
}
