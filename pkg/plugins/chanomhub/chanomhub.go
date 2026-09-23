package chanomhub

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultAPIBase    = "https://api.chanomhub.com"
	DefaultStorageURL = "https://oi.chanomhub.com"
)

// Client handles interaction with the Chanomhub platform
type Client struct {
	APIBase    string
	StorageURL string
	Token      string
	HTTPClient *http.Client
}

// Config stores persistent settings
type Config struct {
	APIBase         string `json:"api_base"`
	StorageURL      string `json:"storage_url"`
	Token           string `json:"token"`
	RefreshToken    string `json:"refresh_token,omitempty"`
	Username        string `json:"username,omitempty"`
	Email           string `json:"email,omitempty"`
	UserID          string `json:"user_id,omitempty"`
	DefaultLanguage string `json:"default_language,omitempty"`
	LastSlug        string `json:"last_slug,omitempty"`
}

// TokenUserInfo holds user identity parsed from the Chanomhub JWT token
type TokenUserInfo struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// ParseTokenUserInfo decodes the payload of a Chanomhub JWT token without verifying secret signature
func ParseTokenUserInfo(token string) (*TokenUserInfo, error) {
	clean := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(token), "Bearer "))
	parts := strings.Split(clean, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid jwt token format")
	}

	segment := parts[1]
	if rem := len(segment) % 4; rem != 0 {
		segment += strings.Repeat("=", 4-rem)
	}

	decoded, err := base64.URLEncoding.DecodeString(segment)
	if err != nil {
		decoded, err = base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, fmt.Errorf("failed to decode jwt payload: %w", err)
		}
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse jwt json: %w", err)
	}

	info := &TokenUserInfo{}
	for _, k := range []string{"username", "user_name", "name", "displayName", "login", "user"} {
		if v, ok := claims[k].(string); ok && v != "" {
			info.Username = v
			break
		}
	}
	for _, k := range []string{"sub", "id", "user_id", "userId"} {
		if v, ok := claims[k].(string); ok && v != "" {
			info.UserID = v
			break
		} else if v, ok := claims[k].(float64); ok {
			info.UserID = fmt.Sprintf("%.0f", v)
			break
		}
	}
	if v, ok := claims["email"].(string); ok {
		info.Email = v
	}

	if info.Username == "" && info.Email != "" {
		info.Username = strings.Split(info.Email, "@")[0]
	}
	if info.Username == "" && info.UserID != "" {
		info.Username = "User_" + info.UserID
	}

	return info, nil
}

// NewClient creates a new Chanomhub client
func NewClient(apiBase, storageURL, token string) *Client {
	if apiBase == "" {
		apiBase = DefaultAPIBase
	}
	if storageURL == "" {
		storageURL = DefaultStorageURL
	}
	return &Client{
		APIBase:    strings.TrimRight(apiBase, "/"),
		StorageURL: strings.TrimRight(storageURL, "/"),
		Token:      strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(token), "Bearer ")),
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// PublishRequest holds parameters for publishing a translation mod
type PublishRequest struct {
	Workspace       string                 `json:"workspace,omitempty"`
	PatchFile       string                 `json:"patch_file,omitempty"`
	GameDir         string                 `json:"game_dir,omitempty"`
	Slug            string                 `json:"slug"`
	Language        string                 `json:"language"`
	Engine          string                 `json:"engine"`
	CreditTo        string                 `json:"credit_to"`
	GameVersion     string                 `json:"game_version,omitempty"`
	TranslatorModel string                 `json:"translator_model,omitempty"`
	SourceLanguage  string                 `json:"source_language,omitempty"`
	TargetLanguage  string                 `json:"target_language,omitempty"`
	SHA256          string                 `json:"sha256,omitempty"`
	Stats           map[string]interface{} `json:"stats,omitempty"`
	Config          map[string]interface{} `json:"config,omitempty"`
}

// PublishResult contains response details from Chanomhub
type PublishResult struct {
	Success       bool   `json:"success"`
	ModID         int    `json:"mod_id,omitempty"`
	Status        string `json:"status,omitempty"`
	DownloadURL   string `json:"download_url"`
	SHA256        string `json:"sha256,omitempty"`
	FileSizeBytes int64  `json:"file_size_bytes"`
	Message       string `json:"message"`
}

// ZipDirectory compresses a directory into a zip archive
func ZipDirectory(srcDir, outZipPath string) (int64, error) {
	if fi, err := os.Stat(srcDir); err != nil || !fi.IsDir() {
		return 0, fmt.Errorf("source directory does not exist: %s", srcDir)
	}

	if err := os.MkdirAll(filepath.Dir(outZipPath), 0755); err != nil {
		return 0, err
	}

	zipFile, err := os.Create(outZipPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		// Use forward slashes for zip compatibility
		relPath = filepath.ToSlash(relPath)

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = relPath
		header.Method = zip.Deflate

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})

	if err != nil {
		return 0, fmt.Errorf("error during archiving: %w", err)
	}

	_ = archive.Close()
	_ = zipFile.Close()

	stat, err := os.Stat(outZipPath)
	if err != nil {
		return 0, err
	}
	return stat.Size(), nil
}

// PublishTranslation zips the nst_translations directory, uploads to storage, and submits mod metadata
func (c *Client) PublishTranslation(ctx context.Context, req PublishRequest) (*PublishResult, error) {
	if c.Token == "" {
		return nil, fmt.Errorf("chanomhub API token is required")
	}
	if req.Slug == "" {
		return nil, fmt.Errorf("article slug is required")
	}
	if req.Language == "" {
		req.Language = "Thai"
	}
	if req.Engine == "" {
		req.Engine = "rpgm"
	}
	if req.CreditTo == "" {
		req.CreditTo = "NST"
	}

	var uploadFilePath string
	var uploadFileName string
	var fileSize int64
	var isTempFile bool

	if req.PatchFile != "" {
		fi, err := os.Stat(req.PatchFile)
		if err != nil {
			return nil, fmt.Errorf("patch file does not exist: %s", req.PatchFile)
		}
		uploadFilePath = req.PatchFile
		uploadFileName = filepath.Base(req.PatchFile)
		fileSize = fi.Size()
	} else {
		// 1. Locate translations directory (lingo_translations, or fallback to legacy nst_translations)
		transDir := filepath.Join(req.GameDir, "lingo_translations")
		if fi, err := os.Stat(transDir); err != nil || !fi.IsDir() {
			transDir = filepath.Join(req.GameDir, "nst_translations")
		}
		if fi, err := os.Stat(transDir); err != nil || !fi.IsDir() {
			// Fallback: check if gameDir itself has config.json or translations
			if _, err := os.Stat(filepath.Join(req.GameDir, "config.json")); err == nil {
				transDir = req.GameDir
			} else {
				return nil, fmt.Errorf("no patch file specified and no 'lingo_translations' or 'nst_translations' directory found in %s", req.GameDir)
			}
		}

		// 2. Compress into temporary zip
		tempZip := filepath.Join(os.TempDir(), fmt.Sprintf("nst_pack_%d.zip", time.Now().UnixMilli()))
		isTempFile = true
		uploadFilePath = tempZip
		uploadFileName = filepath.Base(tempZip)

		sz, err := ZipDirectory(transDir, tempZip)
		if err != nil {
			return nil, fmt.Errorf("failed to pack translations: %w", err)
		}
		fileSize = sz
	}

	if isTempFile {
		defer os.Remove(uploadFilePath)
	}

	// 3. Upload archive to storage service (GOR2-compatible POST /upload?bucket=storage&game=<slug>)
	fileBytes, err := os.ReadFile(uploadFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read upload file: %w", err)
	}

	h := sha256.New()
	h.Write(fileBytes)
	fileSHA256 := hex.EncodeToString(h.Sum(nil))

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", uploadFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(fileBytes); err != nil {
		return nil, fmt.Errorf("failed to write multipart data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	uploadURL := fmt.Sprintf("%s/upload?bucket=storage&game=%s", c.StorageURL, req.Slug)
	uploadReq, err := http.NewRequestWithContext(ctx, "POST", uploadURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create upload request: %w", err)
	}
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadReq.Header.Set("Authorization", "Bearer "+c.Token)

	uploadResp, err := c.HTTPClient.Do(uploadReq)
	if err != nil {
		return nil, fmt.Errorf("upload HTTP request failed: %w", err)
	}
	defer uploadResp.Body.Close()

	uploadRespBody, _ := io.ReadAll(uploadResp.Body)
	if uploadResp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("upload failed (HTTP %d): %s", uploadResp.StatusCode, string(uploadRespBody))
	}

	var uploadJSON map[string]interface{}
	if err := json.Unmarshal(uploadRespBody, &uploadJSON); err != nil {
		return nil, fmt.Errorf("invalid upload response JSON: %w", err)
	}

	var downloadURL string
	if fu, ok := uploadJSON["full_url"].(string); ok && strings.HasPrefix(fu, "http") {
		downloadURL = fu
	} else if u, ok := uploadJSON["url"].(string); ok && strings.HasPrefix(u, "http") {
		downloadURL = u
	} else {
		fileKey := ""
		if key, ok := uploadJSON["key"].(string); ok && key != "" {
			fileKey = key
		} else if u, ok := uploadJSON["url"].(string); ok && u != "" {
			fileKey = u
		} else if fn, ok := uploadJSON["filename"].(string); ok {
			fileKey = fn
		}
		downloadURL = fmt.Sprintf("%s/%s", c.StorageURL, strings.TrimPrefix(fileKey, "/"))
	}

	// 4. Register as pending TRANSLATION mod
	submitPayload := map[string]interface{}{
		"downloadLink":     downloadURL,
		"sha256":           fileSHA256,
		"language":         req.Language,
		"engine":           req.Engine,
		"creditTo":         req.CreditTo,
		"fileSizeBytes":    fileSize,
		"gameVersion":      req.GameVersion,
		"translatorModel":  req.TranslatorModel,
		"sourceLanguage":   req.SourceLanguage,
		"targetLanguage":   req.TargetLanguage,
		"stats":            req.Stats,
		"config":           req.Config,
	}
	payloadBytes, _ := json.Marshal(submitPayload)

	apiBase := c.APIBase
	if !strings.HasSuffix(apiBase, "/api") {
		apiBase = apiBase + "/api"
	}
	submitURL := fmt.Sprintf("%s/mods/article/%s/nst-submission", apiBase, req.Slug)
	submitReq, err := http.NewRequestWithContext(ctx, "POST", submitURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create submit request: %w", err)
	}
	submitReq.Header.Set("Content-Type", "application/json")
	submitReq.Header.Set("Authorization", "Bearer "+c.Token)

	submitResp, err := c.HTTPClient.Do(submitReq)
	if err != nil {
		return nil, fmt.Errorf("submission HTTP request failed: %w", err)
	}
	defer submitResp.Body.Close()

	submitRespBody, _ := io.ReadAll(submitResp.Body)
	if submitResp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("submission failed (HTTP %d): %s", submitResp.StatusCode, string(submitRespBody))
	}

	var submitJSON map[string]interface{}
	modID := 0
	modStatus := "PENDING"
	if err := json.Unmarshal(submitRespBody, &submitJSON); err == nil {
		if dataObj, ok := submitJSON["data"].(map[string]interface{}); ok {
			submitJSON = dataObj
		}
		if modObj, ok := submitJSON["mod"].(map[string]interface{}); ok {
			if id, ok := modObj["id"].(float64); ok {
				modID = int(id)
			}
			if st, ok := modObj["status"].(string); ok {
				modStatus = st
			}
		} else if id, ok := submitJSON["id"].(float64); ok {
			modID = int(id)
		}
	}

	msg := "Submitted successfully! Translation is pending moderation on Chanomhub."
	if modID > 0 {
		msg = fmt.Sprintf("Submitted successfully (Mod ID: %d, Status: %s)! Translation is pending moderation on Chanomhub.", modID, modStatus)
	}

	return &PublishResult{
		Success:       true,
		ModID:         modID,
		Status:        modStatus,
		DownloadURL:   downloadURL,
		SHA256:        fileSHA256,
		FileSizeBytes: fileSize,
		Message:       msg,
	}, nil
}
