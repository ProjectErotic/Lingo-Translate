package chanomhub

import (
	"archive/zip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChanomhubZipAndPublish(t *testing.T) {
	// 1. Create a dummy lingo_translations directory
	tmpDir, err := os.MkdirTemp("", "nst_chanomhub_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	transDir := filepath.Join(tmpDir, "lingo_translations")
	_ = os.MkdirAll(transDir, 0755)
	_ = os.WriteFile(filepath.Join(transDir, "config.json"), []byte(`{"version":"1.0","language":"Thai"}`), 0644)
	_ = os.WriteFile(filepath.Join(transDir, "Map001.txt"), []byte("<<<ORIGINAL>>>\nHello\n<<<TRANSLATED>>>\nสวัสดี\n"), 0644)

	// Test ZipDirectory
	outZip := filepath.Join(tmpDir, "packed.zip")
	size, err := ZipDirectory(transDir, outZip)
	if err != nil {
		t.Fatalf("ZipDirectory failed: %v", err)
	}
	if size <= 0 {
		t.Fatalf("expected zip size > 0, got %d", size)
	}

	// Verify zip contents
	r, err := zip.OpenReader(outZip)
	if err != nil {
		t.Fatalf("failed to open created zip: %v", err)
	}
	defer r.Close()

	foundConfig := false
	foundMap := false
	for _, f := range r.File {
		if f.Name == "config.json" {
			foundConfig = true
		}
		if f.Name == "Map001.txt" {
			foundMap = true
		}
	}
	if !foundConfig || !foundMap {
		t.Errorf("zip file missing expected files. config=%v, map=%v", foundConfig, foundMap)
	}

	// 2. Setup mock server for upload & submit
	uploadHit := false
	submitHit := false

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-secret-token" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if r.URL.Path == "/upload" {
			uploadHit = true
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"key":      "uploads/nst_pack_1234.zip",
				"filename": "nst_pack_1234.zip",
				"url":      "https://oi.mock.com/uploads/nst_pack_1234.zip",
			})
			return
		}

		if strings.Contains(r.URL.Path, "/nst-submission") {
			submitHit = true
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["creditTo"] != "NST" {
				http.Error(w, "Bad creditTo", http.StatusBadRequest)
				return
			}
			if sha, ok := body["sha256"].(string); !ok || sha == "" {
				http.Error(w, "Missing sha256", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "pending",
				"id":     123,
			})
			return
		}

		http.NotFound(w, r)
	}))
	defer mockServer.Close()

	// 3. Test PublishTranslation
	client := NewClient(mockServer.URL, mockServer.URL, "test-secret-token")
	res, err := client.PublishTranslation(context.Background(), PublishRequest{
		GameDir:  tmpDir,
		Slug:     "test-game-rpgm",
		Language: "Thai",
		Engine:   "rpgm",
	})
	if err != nil {
		t.Fatalf("PublishTranslation failed: %v", err)
	}

	if !res.Success {
		t.Errorf("expected res.Success to be true")
	}
	if res.SHA256 == "" {
		t.Errorf("expected res.SHA256 to not be empty")
	}
	if res.ModID != 123 {
		t.Errorf("expected res.ModID to be 123, got %d", res.ModID)
	}
	if !uploadHit {
		t.Errorf("expected upload endpoint to be hit")
	}
	if !submitHit {
		t.Errorf("expected submit endpoint to be hit")
	}

	t.Logf("Chanomhub zip & publish test passed with download url: %s", res.DownloadURL)
}
