package chanomhub

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// LoginRequest holds parameters for logging in to Chanomhub
type LoginRequest struct {
	UsernameOrEmail string `json:"username_or_email"`
	Password        string `json:"password"`
	Token           string `json:"token,omitempty"`
	APIBase         string `json:"api_base,omitempty"`
	StorageURL      string `json:"storage_url,omitempty"`
}

// LoginResult holds the output after a successful login
type LoginResult struct {
	Success      bool           `json:"success"`
	Token        string         `json:"token"`
	RefreshToken string         `json:"refresh_token,omitempty"`
	UserInfo     *TokenUserInfo `json:"user_info"`
	APIBase      string         `json:"api_base"`
	StorageURL   string         `json:"storage_url"`
	Message      string         `json:"message"`
}

// DefaultConfigPath returns the canonical path to chanomhub.json
func DefaultConfigPath() string {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		cfgDir = "."
	}
	return filepath.Join(cfgDir, "lingo", "chanomhub.json")
}

// LoadConfig reads the persisted Chanomhub configuration
func LoadConfig() (*Config, error) {
	path := DefaultConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{
				APIBase:    DefaultAPIBase,
				StorageURL: DefaultStorageURL,
			}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.APIBase == "" {
		cfg.APIBase = DefaultAPIBase
	}
	if cfg.StorageURL == "" {
		cfg.StorageURL = DefaultStorageURL
	}

	return &cfg, nil
}

// SaveConfig persists the Chanomhub configuration with 0600 file permissions
func SaveConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("cannot save nil config")
	}

	path := DefaultConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}

	// Also sync token into desktop settings.json if possible
	syncTokenToDesktopSettings(cfg.Token)

	return nil
}

// ClearConfig removes the saved credentials
func ClearConfig() error {
	path := DefaultConfigPath()
	_ = os.Remove(path)

	// Also clear token in desktop settings.json
	syncTokenToDesktopSettings("")

	return nil
}

// syncTokenToDesktopSettings updates chanomhub_token in ~/.config/lingo/settings.json
func syncTokenToDesktopSettings(token string) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return
	}
	settingsPath := filepath.Join(cfgDir, "lingo", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		// Fallback to legacy nst directory
		settingsPath = filepath.Join(cfgDir, "nst", "settings.json")
		data, err = os.ReadFile(settingsPath)
		if err != nil {
			return
		}
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return
	}

	settings["chanomhub_token"] = token
	if updated, err := json.MarshalIndent(settings, "", "  "); err == nil {
		_ = os.WriteFile(settingsPath, updated, 0644)
	}
}

// GetEffectiveToken resolves the token from environment variable or stored config
func GetEffectiveToken() string {
	if env := strings.TrimSpace(os.Getenv("CHANOMHUB_TOKEN")); env != "" {
		return strings.TrimPrefix(env, "Bearer ")
	}

	cfg, err := LoadConfig()
	if err == nil && cfg != nil && cfg.Token != "" {
		return strings.TrimPrefix(strings.TrimSpace(cfg.Token), "Bearer ")
	}

	// Fallback to checking settings.json
	cfgDir, err := os.UserConfigDir()
	if err == nil {
		for _, dir := range []string{"lingo", "nst"} {
			settingsPath := filepath.Join(cfgDir, dir, "settings.json")
			if data, err := os.ReadFile(settingsPath); err == nil {
				var s struct {
					ChanomhubToken string `json:"chanomhub_token"`
				}
				if err := json.Unmarshal(data, &s); err == nil && s.ChanomhubToken != "" {
					return strings.TrimPrefix(strings.TrimSpace(s.ChanomhubToken), "Bearer ")
				}
			}
		}
	}

	return ""
}

// GetEffectiveAPIBase resolves the API Base URL from env, config, or default
func GetEffectiveAPIBase() string {
	if env := strings.TrimSpace(os.Getenv("CHANOMHUB_API_BASE")); env != "" {
		return strings.TrimRight(env, "/")
	}
	if env := strings.TrimSpace(os.Getenv("CHANOMHUB_REGISTRY")); env != "" {
		return strings.TrimRight(env, "/")
	}
	cfg, err := LoadConfig()
	if err == nil && cfg != nil && cfg.APIBase != "" {
		return strings.TrimRight(cfg.APIBase, "/")
	}
	return DefaultAPIBase
}

// GetEffectiveStorageURL resolves the Storage URL from env, config, or default
func GetEffectiveStorageURL() string {
	if env := strings.TrimSpace(os.Getenv("CHANOMHUB_STORAGE_URL")); env != "" {
		return strings.TrimRight(env, "/")
	}
	cfg, err := LoadConfig()
	if err == nil && cfg != nil && cfg.StorageURL != "" {
		return strings.TrimRight(cfg.StorageURL, "/")
	}
	return DefaultStorageURL
}

// Login authenticates with Chanomhub and persists the token
func Login(ctx context.Context, req LoginRequest) (*LoginResult, error) {
	apiBase := strings.TrimRight(req.APIBase, "/")
	if apiBase == "" {
		apiBase = GetEffectiveAPIBase()
	}

	storageURL := strings.TrimRight(req.StorageURL, "/")
	if storageURL == "" {
		storageURL = GetEffectiveStorageURL()
	}

	client := &http.Client{Timeout: 30 * time.Second}

	// Direct token login
	if req.Token != "" {
		token := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(req.Token), "Bearer "))
		userInfo, err := ParseTokenUserInfo(token)
		if err != nil {
			return nil, fmt.Errorf("invalid token: %w", err)
		}

		// Verify with server if reachable
		if verifiedUser, err := fetchCurrentUser(ctx, client, apiBase, token); err == nil && verifiedUser != nil {
			if verifiedUser.Username != "" {
				userInfo.Username = verifiedUser.Username
			}
			if verifiedUser.Email != "" {
				userInfo.Email = verifiedUser.Email
			}
			if verifiedUser.UserID != "" {
				userInfo.UserID = verifiedUser.UserID
			}
		}

		cfg := &Config{
			APIBase:    apiBase,
			StorageURL: storageURL,
			Token:      token,
			Username:   userInfo.Username,
			Email:      userInfo.Email,
			UserID:     userInfo.UserID,
		}
		if err := SaveConfig(cfg); err != nil {
			return nil, fmt.Errorf("failed to save config: %w", err)
		}

		return &LoginResult{
			Success:    true,
			Token:      token,
			UserInfo:   userInfo,
			APIBase:    apiBase,
			StorageURL: storageURL,
			Message:    fmt.Sprintf("Logged in as %s", userInfo.Username),
		}, nil
	}

	// Username / Email + Password login
	usernameOrEmail := strings.TrimSpace(req.UsernameOrEmail)
	if usernameOrEmail == "" {
		return nil, fmt.Errorf("username or email is required")
	}
	if req.Password == "" {
		return nil, fmt.Errorf("password is required")
	}

	loginPayload := map[string]interface{}{
		"user": map[string]string{
			"email":    usernameOrEmail,
			"password": req.Password,
		},
	}
	payloadBytes, _ := json.Marshal(loginPayload)

	apiEndpoint := apiBase
	if !strings.HasSuffix(apiEndpoint, "/api") {
		apiEndpoint += "/api"
	}
	loginURL := fmt.Sprintf("%s/users/login", apiEndpoint)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", loginURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create login request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("connection to %s failed: %w", apiBase, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var errData map[string]interface{}
		if err := json.Unmarshal(respBody, &errData); err == nil {
			if msg, ok := errData["message"].(string); ok && msg != "" {
				return nil, fmt.Errorf("login failed: %s", msg)
			}
		}
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("invalid username/email or password")
		}
		return nil, fmt.Errorf("login failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var resData struct {
		User struct {
			ID       interface{} `json:"id"`
			Name     string      `json:"name"`
			Username string      `json:"username"`
			Email    string      `json:"email"`
			Token    string      `json:"token"`
		} `json:"user"`
		RefreshToken string `json:"refreshToken"`
	}

	if err := json.Unmarshal(respBody, &resData); err != nil {
		return nil, fmt.Errorf("failed to parse login response: %w", err)
	}

	token := strings.TrimSpace(resData.User.Token)
	if token == "" {
		return nil, fmt.Errorf("server returned empty auth token")
	}

	userInfo, err := ParseTokenUserInfo(token)
	if err != nil {
		userInfo = &TokenUserInfo{}
	}

	if resData.User.Name != "" {
		userInfo.Username = resData.User.Name
	} else if resData.User.Username != "" {
		userInfo.Username = resData.User.Username
	}

	if resData.User.Email != "" {
		userInfo.Email = resData.User.Email
	}

	if resData.User.ID != nil {
		userInfo.UserID = fmt.Sprintf("%v", resData.User.ID)
	}

	if userInfo.Username == "" && userInfo.Email != "" {
		userInfo.Username = strings.Split(userInfo.Email, "@")[0]
	}

	cfg := &Config{
		APIBase:      apiBase,
		StorageURL:   storageURL,
		Token:        token,
		RefreshToken: resData.RefreshToken,
		Username:     userInfo.Username,
		Email:        userInfo.Email,
		UserID:       userInfo.UserID,
	}

	if err := SaveConfig(cfg); err != nil {
		return nil, fmt.Errorf("failed to persist credentials: %w", err)
	}

	return &LoginResult{
		Success:      true,
		Token:        token,
		RefreshToken: resData.RefreshToken,
		UserInfo:     userInfo,
		APIBase:      apiBase,
		StorageURL:   storageURL,
		Message:      fmt.Sprintf("Logged in as %s", userInfo.Username),
	}, nil
}

// Whoami checks and returns the current authenticated user
func Whoami(ctx context.Context) (*TokenUserInfo, string, error) {
	token := GetEffectiveToken()
	if token == "" {
		return nil, "", fmt.Errorf("not logged in to Chanomhub. Run 'nst login' to authenticate")
	}

	apiBase := GetEffectiveAPIBase()
	client := &http.Client{Timeout: 15 * time.Second}

	// First try to fetch fresh user info from server
	userInfo, err := fetchCurrentUser(ctx, client, apiBase, token)
	if err == nil && userInfo != nil {
		return userInfo, apiBase, nil
	}

	// Fallback to parsing local JWT claims if server is unreachable
	if jwtInfo, parseErr := ParseTokenUserInfo(token); parseErr == nil && jwtInfo != nil {
		return jwtInfo, apiBase, nil
	}

	return nil, apiBase, fmt.Errorf("failed to verify Chanomhub session: %w", err)
}

// fetchCurrentUser calls GET /api/user with the bearer token
func fetchCurrentUser(ctx context.Context, client *http.Client, apiBase, token string) (*TokenUserInfo, error) {
	apiEndpoint := apiBase
	if !strings.HasSuffix(apiEndpoint, "/api") {
		apiEndpoint += "/api"
	}
	userURL := fmt.Sprintf("%s/user", apiEndpoint)

	req, err := http.NewRequestWithContext(ctx, "GET", userURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var res struct {
		User struct {
			ID       interface{} `json:"id"`
			Name     string      `json:"name"`
			Username string      `json:"username"`
			Email    string      `json:"email"`
		} `json:"user"`
	}

	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}

	info := &TokenUserInfo{
		Username: res.User.Username,
		Email:    res.User.Email,
	}
	if info.Username == "" {
		info.Username = res.User.Name
	}
	if res.User.ID != nil {
		info.UserID = fmt.Sprintf("%v", res.User.ID)
	}
	if info.Username == "" && info.Email != "" {
		info.Username = strings.Split(info.Email, "@")[0]
	}

	return info, nil
}

const DefaultWebURL = "https://chanomhub.com"

// GetEffectiveWebURL resolves the frontend web URL from env or default
func GetEffectiveWebURL() string {
	if env := strings.TrimSpace(os.Getenv("CHANOMHUB_WEB_URL")); env != "" {
		return strings.TrimRight(env, "/")
	}
	return DefaultWebURL
}

// GenerateSecureState generates a cryptographically random 16-byte hex nonce
func GenerateSecureState() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// BuildAuthorizationURL creates a standard, universal OAuth/CLI authorization link
func BuildAuthorizationURL(webBase, clientID, clientName, redirectURI, state string) string {
	if webBase == "" {
		webBase = GetEffectiveWebURL()
	}
	endpoint := strings.TrimRight(webBase, "/") + "/auth/cli"

	u, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}

	q := u.Query()
	if clientID != "" {
		q.Set("client_id", clientID)
	}
	if clientName != "" {
		q.Set("client_name", clientName)
	}
	if redirectURI != "" {
		q.Set("redirect_uri", redirectURI)
	}
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()

	return u.String()
}

// StartLocalCallbackServer starts a temporary HTTP server on localhost to listen for the OAuth/CLI token callback
func StartLocalCallbackServer(preferredPort int, expectedState string) (actualPort int, tokenChan chan string, cleanup func(), err error) {
	portsToTry := []int{}
	if preferredPort > 0 {
		portsToTry = append(portsToTry, preferredPort)
		for i := 1; i <= 5; i++ {
			portsToTry = append(portsToTry, preferredPort+i)
		}
	}
	portsToTry = append(portsToTry, 0) // 0 lets OS pick random free port

	var listener net.Listener
	for _, p := range portsToTry {
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p))
		if err == nil {
			listener = l
			break
		}
	}
	if listener == nil {
		return 0, nil, nil, fmt.Errorf("failed to bind local callback server to any port")
	}

	actualPort = listener.Addr().(*net.TCPAddr).Port
	tokenChan = make(chan string, 1)

	mux := http.NewServeMux()
	server := &http.Server{Handler: mux}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Verify state nonce if expected
		if expectedState != "" {
			reqState := strings.TrimSpace(r.URL.Query().Get("state"))
			if reqState != expectedState {
				http.Error(w, "Security Error: State nonce mismatch (possible CSRF attack)", http.StatusForbidden)
				return
			}
		}

		token := strings.TrimSpace(r.URL.Query().Get("token"))
		if token == "" {
			token = strings.TrimSpace(r.URL.Query().Get("jwt"))
		}
		if token == "" {
			token = strings.TrimSpace(r.URL.Query().Get("access_token"))
		}

		if token != "" {
			select {
			case tokenChan <- token:
			default:
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>Authorization Successful</title></head>
<body style="font-family:system-ui,-apple-system,sans-serif;text-align:center;padding:60px 20px;background:#09090b;color:#fafafa;">
  <div style="max-width:420px;margin:0 auto;background:#18181b;padding:32px;border-radius:16px;border:1px solid #27272a;box-shadow:0 10px 25px -5px rgba(0,0,0,0.5);">
    <div style="width:52px;height:52px;background:rgba(16,185,129,0.15);color:#10b981;border-radius:50%;display:flex;align-items:center;justify-content:center;margin:0 auto 16px auto;font-size:26px;">✓</div>
    <h2 style="color:#fafafa;margin-bottom:8px;font-size:20px;">Authorization Successful!</h2>
    <p style="color:#a1a1aa;font-size:14px;line-height:1.5;margin-bottom:0;">You can now close this browser tab and return to your terminal.</p>
  </div>
</body>
</html>`))
		} else {
			http.Error(w, "Missing token parameter", http.StatusBadRequest)
		}
	})

	go func() {
		_ = server.Serve(listener)
	}()

	cleanup = func() {
		_ = server.Close()
		_ = listener.Close()
	}

	return actualPort, tokenChan, cleanup, nil
}

// OpenBrowser opens the given URL in the user's default browser
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// RequestUchsKey requests a new Virtual Key from Chanomhub for the UCHS AI gateway
func RequestUchsKey(ctx context.Context, token, apiBase string) (string, error) {
	if token == "" {
		token = GetEffectiveToken()
	}
	if token == "" {
		return "", fmt.Errorf("authentication token is required (please log in to Chanomhub first)")
	}
	if apiBase == "" {
		apiBase = GetEffectiveAPIBase()
	}
	apiBase = strings.TrimRight(apiBase, "/")

	req, err := http.NewRequestWithContext(ctx, "POST", apiBase+"/api/uchs/keys/generate", bytes.NewReader([]byte("{}")))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Chanomhub UCHS key endpoint: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}

	var res struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return "", fmt.Errorf("failed to parse key response: %w", err)
	}

	if res.Key == "" {
		return "", fmt.Errorf("server returned empty key")
	}

	return res.Key, nil
}

// GetUchsSsoURL retrieves a one-click SSO redirect URL from Chanomhub to open the UCHS portal
func GetUchsSsoURL(ctx context.Context, token, apiBase string) (string, error) {
	if token == "" {
		token = GetEffectiveToken()
	}
	if token == "" {
		return "", fmt.Errorf("authentication token is required (please log in to Chanomhub first)")
	}
	if apiBase == "" {
		apiBase = GetEffectiveAPIBase()
	}
	apiBase = strings.TrimRight(apiBase, "/")

	req, err := http.NewRequestWithContext(ctx, "POST", apiBase+"/api/uchs/sso-url", bytes.NewReader([]byte("{}")))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Chanomhub SSO URL endpoint: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}

	var res struct {
		RedirectURL string `json:"redirectUrl"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return "", fmt.Errorf("failed to parse SSO URL response: %w", err)
	}

	if res.RedirectURL == "" {
		return "", fmt.Errorf("server returned empty redirect URL")
	}

	return res.RedirectURL, nil
}

// OpenUchsPortal retrieves the SSO URL and opens it in the user's default browser
func OpenUchsPortal(ctx context.Context, token, apiBase string) error {
	redirectURL, err := GetUchsSsoURL(ctx, token, apiBase)
	if err != nil {
		return err
	}
	return OpenBrowser(redirectURL)
}

