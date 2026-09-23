package chanomhub

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createDummyJWT creates an unsigned JWT with given claims for testing
func createDummyJWT(payload map[string]interface{}) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payloadBytes, _ := json.Marshal(payload)
	body := base64.RawURLEncoding.EncodeToString(payloadBytes)
	sig := base64.RawURLEncoding.EncodeToString([]byte("signature"))
	return header + "." + body + "." + sig
}

func TestChanomhubAuthFlow(t *testing.T) {
	// Setup isolated user config dir
	tmpConfigDir, err := os.MkdirTemp("", "nst_auth_test_*")
	if err != nil {
		t.Fatalf("failed to create temp config dir: %v", err)
	}
	defer os.RemoveAll(tmpConfigDir)

	origConfigDir := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpConfigDir)
	defer func() {
		if origConfigDir != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origConfigDir)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	dummyToken := createDummyJWT(map[string]interface{}{
		"sub":      42,
		"username": "tester_user",
		"email":    "tester@chanomhub.com",
	})

	// Setup mock Chanomhub server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/users/login"):
			var body struct {
				User struct {
					Email    string `json:"email"`
					Password string `json:"password"`
				} `json:"user"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)

			if body.User.Email == "invalid@example.com" || body.User.Password == "wrong" {
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"message": "Invalid email/username or password",
				})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"user": map[string]interface{}{
					"id":       42,
					"name":     "tester_user",
					"username": "tester_user",
					"email":    body.User.Email,
					"token":    dummyToken,
				},
				"refreshToken": "dummy_refresh_token_123",
			})

		case strings.HasSuffix(r.URL.Path, "/user"):
			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer "+dummyToken {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"user": map[string]interface{}{
					"id":       42,
					"name":     "tester_user",
					"username": "tester_user",
					"email":    "tester@chanomhub.com",
				},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	ctx := context.Background()

	// 1. Test Whoami when not logged in
	_ = ClearConfig()
	_, _, err = Whoami(ctx)
	if err == nil {
		t.Errorf("expected error when whoami without login, got nil")
	}

	// 2. Test Invalid Login
	_, err = Login(ctx, LoginRequest{
		APIBase:         mockServer.URL,
		UsernameOrEmail: "invalid@example.com",
		Password:        "wrong",
	})
	if err == nil {
		t.Errorf("expected login failure for bad credentials, got nil")
	}

	// 3. Test Successful Password Login
	res, err := Login(ctx, LoginRequest{
		APIBase:         mockServer.URL,
		UsernameOrEmail: "tester@chanomhub.com",
		Password:        "correctpassword",
	})
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if !res.Success {
		t.Errorf("expected login success, got false")
	}
	if res.Token != dummyToken {
		t.Errorf("expected token %s, got %s", dummyToken, res.Token)
	}
	if res.UserInfo.Username != "tester_user" {
		t.Errorf("expected username 'tester_user', got %s", res.UserInfo.Username)
	}
	if res.UserInfo.UserID != "42" {
		t.Errorf("expected user ID '42', got %s", res.UserInfo.UserID)
	}

	// 4. Verify config was saved to disk
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Token != dummyToken {
		t.Errorf("saved token mismatch: %s vs %s", cfg.Token, dummyToken)
	}
	if cfg.Username != "tester_user" {
		t.Errorf("saved username mismatch: %s vs %s", cfg.Username, "tester_user")
	}

	// 5. Test Whoami when logged in
	info, registry, err := Whoami(ctx)
	if err != nil {
		t.Fatalf("Whoami failed: %v", err)
	}
	if info.Username != "tester_user" {
		t.Errorf("expected username tester_user, got %s", info.Username)
	}
	if registry != mockServer.URL {
		t.Errorf("expected registry %s, got %s", mockServer.URL, registry)
	}

	// 6. Test Direct Token Login
	_ = ClearConfig()
	resToken, err := Login(ctx, LoginRequest{
		APIBase: mockServer.URL,
		Token:   dummyToken,
	})
	if err != nil {
		t.Fatalf("Direct token login failed: %v", err)
	}
	if resToken.UserInfo.Username != "tester_user" {
		t.Errorf("expected username tester_user, got %s", resToken.UserInfo.Username)
	}

	// 7. Test Logout / ClearConfig
	if err := ClearConfig(); err != nil {
		t.Fatalf("ClearConfig failed: %v", err)
	}
	if tok := GetEffectiveToken(); tok != "" {
		t.Errorf("expected empty token after ClearConfig, got %s", tok)
	}
}

func TestSyncTokenToDesktopSettings(t *testing.T) {
	tmpConfigDir, err := os.MkdirTemp("", "nst_settings_test_*")
	if err != nil {
		t.Fatalf("failed to create temp config dir: %v", err)
	}
	defer os.RemoveAll(tmpConfigDir)

	origConfigDir := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpConfigDir)
	defer func() {
		if origConfigDir != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origConfigDir)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	nstDir := filepath.Join(tmpConfigDir, "nst")
	_ = os.MkdirAll(nstDir, 0755)
	settingsPath := filepath.Join(nstDir, "settings.json")
	_ = os.WriteFile(settingsPath, []byte(`{"default_provider":"gemini","chanomhub_token":""}`), 0644)

	cfg := &Config{
		APIBase: DefaultAPIBase,
		Token:   "sync_test_token_123",
	}
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	data, _ := os.ReadFile(settingsPath)
	var s struct {
		ChanomhubToken string `json:"chanomhub_token"`
	}
	_ = json.Unmarshal(data, &s)
	if s.ChanomhubToken != "sync_test_token_123" {
		t.Errorf("expected settings.json chanomhub_token to be 'sync_test_token_123', got '%s'", s.ChanomhubToken)
	}

	_ = ClearConfig()
	data2, _ := os.ReadFile(settingsPath)
	_ = json.Unmarshal(data2, &s)
	if s.ChanomhubToken != "" {
		t.Errorf("expected settings.json chanomhub_token to be cleared, got '%s'", s.ChanomhubToken)
	}
}

func TestLocalCallbackServerWithState(t *testing.T) {
	state := GenerateSecureState()
	if len(state) < 16 {
		t.Fatalf("expected state length >= 16, got %d", len(state))
	}

	actualPort, tokenChan, cleanup, err := StartLocalCallbackServer(0, state)
	if err != nil {
		t.Fatalf("StartLocalCallbackServer failed: %v", err)
	}
	defer cleanup()

	if actualPort <= 0 {
		t.Fatalf("expected positive actual port, got %d", actualPort)
	}

	authURL := BuildAuthorizationURL("https://chanomhub.com", "my-app", "My App Name", "http://127.0.0.1:9999/callback", state)
	if !strings.Contains(authURL, "client_id=my-app") || !strings.Contains(authURL, "state="+state) {
		t.Errorf("BuildAuthorizationURL output malformed: %s", authURL)
	}

	// 1. Test callback with invalid state -> must fail with 403
	badURL := fmt.Sprintf("http://127.0.0.1:%d/callback?token=bad_token&state=wrong_state", actualPort)
	respBad, err := http.Get(badURL)
	if err != nil {
		t.Fatalf("failed GET callback: %v", err)
	}
	defer respBad.Body.Close()
	if respBad.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for bad state, got %d", respBad.StatusCode)
	}

	// 2. Test callback with valid state -> must succeed with 200 and receive token
	goodURL := fmt.Sprintf("http://127.0.0.1:%d/callback?token=valid_token_abc&state=%s", actualPort, state)
	respGood, err := http.Get(goodURL)
	if err != nil {
		t.Fatalf("failed GET callback: %v", err)
	}
	defer respGood.Body.Close()
	if respGood.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for valid state, got %d", respGood.StatusCode)
	}

	select {
	case receivedToken := <-tokenChan:
		if receivedToken != "valid_token_abc" {
			t.Errorf("expected 'valid_token_abc', got '%s'", receivedToken)
		}
	default:
		t.Errorf("token was not sent to channel")
	}
}

func TestRequestUchsKeyAndSso_DataWrapped(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/uchs/keys/generate":
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{
				"statusCode": 201,
				"data": {
					"id": 10,
					"key": "sk-uchs-test-key-12345",
					"keyAlias": "chanomhub-user-123",
					"maxBudget": 5
				}
			}`))
		case "/api/uchs/sso-url":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"statusCode": 200,
				"data": {
					"redirectUrl": "https://uchs-th.com/auth/sso?ticket=ticket123"
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx := context.Background()

	// 1. Test RequestUchsKey with wrapped response
	key, err := RequestUchsKey(ctx, "mock_token", server.URL)
	if err != nil {
		t.Fatalf("RequestUchsKey failed: %v", err)
	}
	if key != "sk-uchs-test-key-12345" {
		t.Errorf("expected 'sk-uchs-test-key-12345', got '%s'", key)
	}

	// 2. Test GetUchsSsoURL with wrapped response
	ssoURL, err := GetUchsSsoURL(ctx, "mock_token", server.URL)
	if err != nil {
		t.Fatalf("GetUchsSsoURL failed: %v", err)
	}
	if ssoURL != "https://uchs-th.com/auth/sso?ticket=ticket123" {
		t.Errorf("expected 'https://uchs-th.com/auth/sso?ticket=ticket123', got '%s'", ssoURL)
	}
}
