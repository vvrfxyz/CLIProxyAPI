package management

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
)

func TestGetLogsIncludesDownloadableResponsesEntries(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	logContent := "[2026-05-27 10:00:00] [abcd1234] [info ] 200 |          1.2s |       127.0.0.1 | POST    \"/v1/responses\"\n" +
		"[2026-05-27 10:00:01] [ef567890] [info ] 200 |          9ms |       127.0.0.1 | GET     \"/v0/management/logs\"\n"
	if err := os.WriteFile(filepath.Join(tmpDir, defaultLogFileName), []byte(logContent), 0o644); err != nil {
		t.Fatalf("write log file: %v", err)
	}

	h := NewHandler(&config.Config{LoggingToFile: true}, "", nil)
	h.SetLogDirectory(tmpDir)

	router := gin.New()
	router.GET("/logs", h.GetLogs)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/logs", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var body struct {
		Lines   []string `json:"lines"`
		Entries []struct {
			Line                   string `json:"line"`
			RequestID              string `json:"request_id"`
			Method                 string `json:"method"`
			Path                   string `json:"path"`
			RequestLogDownloadURL  string `json:"request_log_download_url"`
			RequestLogDownloadable bool   `json:"request_log_downloadable"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(body.Lines) != 2 {
		t.Fatalf("lines len = %d, want 2", len(body.Lines))
	}
	if len(body.Entries) != 2 {
		t.Fatalf("entries len = %d, want 2", len(body.Entries))
	}

	entry := body.Entries[0]
	if entry.Line != body.Lines[0] {
		t.Fatalf("entry line = %q, want %q", entry.Line, body.Lines[0])
	}
	if entry.RequestID != "abcd1234" {
		t.Fatalf("request_id = %q, want abcd1234", entry.RequestID)
	}
	if entry.Method != http.MethodPost {
		t.Fatalf("method = %q, want POST", entry.Method)
	}
	if entry.Path != "/v1/responses" {
		t.Fatalf("path = %q, want /v1/responses", entry.Path)
	}
	if !entry.RequestLogDownloadable {
		t.Fatalf("request_log_downloadable = false, want true")
	}
	if entry.RequestLogDownloadURL != "/v0/management/request-log-by-id/abcd1234" {
		t.Fatalf("download url = %q", entry.RequestLogDownloadURL)
	}

	if body.Entries[1].RequestLogDownloadable {
		t.Fatalf("management log entry should not be downloadable")
	}
}

func TestParseManagementLogEntryHandlesResponsesSubpaths(t *testing.T) {
	entry := parseManagementLogEntry(`[2026-05-27 10:00:00] [a1b2c3d4] [info ] 200 | 5ms | ::1 | POST "/v1/responses/compact?alt=sse"`)

	if !entry.RequestLogDownloadable {
		t.Fatalf("expected responses subpath to be downloadable")
	}
	if entry.RequestLogDownloadURL != "/v0/management/request-log-by-id/a1b2c3d4" {
		t.Fatalf("download url = %q", entry.RequestLogDownloadURL)
	}
}
