package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"nofx/auth"
	"nofx/config"
	"nofx/manager"
	"path/filepath"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func setupTestServer(t *testing.T) (*Server, *config.Database) {
	t.Helper()

	origSecret := auth.JWTSecret
	origAdmin := auth.AdminMode
	t.Cleanup(func() {
		auth.JWTSecret = origSecret
		auth.AdminMode = origAdmin
	})

	auth.SetAdminMode(false)
	auth.SetJWTSecret("unit-test-jwt-secret-0123456789abcdef0123456789abcdef")

	dbPath := filepath.Join(t.TempDir(), "api_test.db")
	db, err := config.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase failed: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	tm := manager.NewTraderManager()
	srv := NewServer(tm, db, 0)
	return srv, db
}

func doJSONRequest(t *testing.T, srv *Server, method, path string, body interface{}, bearerToken string) *httptest.ResponseRecorder {
	t.Helper()
	var reqBody bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&reqBody).Encode(body); err != nil {
			t.Fatalf("encode json failed: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &reqBody)
	req.Header.Set("Content-Type", "application/json")
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)
	return rec
}

func registerAndLoginUser(t *testing.T, srv *Server, email, password string) (userID, token string) {
	t.Helper()

	// Reset rate limiter so helper registrations across test cases don't collide.
	srv.authRateLimiter.Reset()

	regRec := doJSONRequest(t, srv, http.MethodPost, "/api/register", map[string]string{
		"email":    email,
		"password": password,
	}, "")
	if regRec.Code != http.StatusOK {
		t.Fatalf("register failed (%d): %s", regRec.Code, regRec.Body.String())
	}

	var regResp struct {
		UserID    string `json:"user_id"`
		OTPSecret string `json:"otp_secret"`
	}
	if err := json.Unmarshal(regRec.Body.Bytes(), &regResp); err != nil {
		t.Fatalf("unmarshal register response: %v", err)
	}

	otpCode, err := totp.GenerateCode(regResp.OTPSecret, time.Now())
	if err != nil {
		t.Fatalf("totp.GenerateCode: %v", err)
	}

	compRec := doJSONRequest(t, srv, http.MethodPost, "/api/complete-registration", map[string]string{
		"user_id":  regResp.UserID,
		"otp_code": otpCode,
	}, "")
	if compRec.Code != http.StatusOK {
		t.Fatalf("complete-registration failed (%d): %s", compRec.Code, compRec.Body.String())
	}

	var compResp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(compRec.Body.Bytes(), &compResp); err != nil {
		t.Fatalf("unmarshal complete-registration response: %v", err)
	}

	return regResp.UserID, compResp.Token
}

func TestAuthFlowAndOTPChallengeBinding(t *testing.T) {
	srv, db := setupTestServer(t)

	userID, _ := registerAndLoginUser(t, srv, "alice@example.com", "strongPass123")
	user, err := db.GetUserByID(userID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}

	otpCode, err := totp.GenerateCode(user.OTPSecret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}

	// Re-calling complete-registration on an already verified user must be rejected.
	dupComp := doJSONRequest(t, srv, http.MethodPost, "/api/complete-registration", map[string]string{
		"user_id":  userID,
		"otp_code": otpCode,
	}, "")
	if dupComp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on duplicate complete-registration, got %d: %s", dupComp.Code, dupComp.Body.String())
	}

	// Calling verify-otp without otp_token must fail.
	noChallenge := doJSONRequest(t, srv, http.MethodPost, "/api/verify-otp", map[string]string{
		"user_id":  userID,
		"otp_code": otpCode,
	}, "")
	if noChallenge.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when otp_token is missing, got %d", noChallenge.Code)
	}

	// Login issues an otp_token bound to userID.
	loginRec := doJSONRequest(t, srv, http.MethodPost, "/api/login", map[string]string{
		"email":    "alice@example.com",
		"password": "strongPass123",
	}, "")
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login failed (%d): %s", loginRec.Code, loginRec.Body.String())
	}
	var loginResp struct {
		UserID   string `json:"user_id"`
		OTPToken string `json:"otp_token"`
	}
	if err := json.Unmarshal(loginRec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("unmarshal login response: %v", err)
	}
	if loginResp.OTPToken == "" {
		t.Fatal("expected non-empty otp_token from /api/login")
	}

	// otp_token must NOT be accepted on protected routes as a bearer token.
	protectedWithChallenge := doJSONRequest(t, srv, http.MethodGet, "/api/my-traders", nil, loginResp.OTPToken)
	if protectedWithChallenge.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when using otp_token as bearer token, got %d", protectedWithChallenge.Code)
	}

	// Valid verify-otp with otp_token succeeds and issues an access JWT.
	verifyRec := doJSONRequest(t, srv, http.MethodPost, "/api/verify-otp", map[string]string{
		"user_id":   userID,
		"otp_code":  otpCode,
		"otp_token": loginResp.OTPToken,
	}, "")
	if verifyRec.Code != http.StatusOK {
		t.Fatalf("verify-otp failed (%d): %s", verifyRec.Code, verifyRec.Body.String())
	}
	var verifyResp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(verifyRec.Body.Bytes(), &verifyResp); err != nil {
		t.Fatalf("unmarshal verify response: %v", err)
	}

	tradersRec := doJSONRequest(t, srv, http.MethodGet, "/api/my-traders", nil, verifyResp.Token)
	if tradersRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/my-traders with valid JWT, got %d", tradersRec.Code)
	}
}

func TestAuthRateLimiting(t *testing.T) {
	srv, _ := setupTestServer(t)
	srv.authRateLimiter = auth.NewRateLimiter(3, time.Minute)

	for i := 0; i < 3; i++ {
		rec := doJSONRequest(t, srv, http.MethodPost, "/api/login", map[string]string{
			"email":    "nobody@example.com",
			"password": "wrongpassword",
		}, "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected 401, got %d", i+1, rec.Code)
		}
	}

	rec := doJSONRequest(t, srv, http.MethodPost, "/api/login", map[string]string{
		"email":    "nobody@example.com",
		"password": "wrongpassword",
	}, "")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 Too Many Requests, got %d", rec.Code)
	}
}

func TestIDORProtectionOnTraderEndpoints(t *testing.T) {
	srv, db := setupTestServer(t)

	userA, _ := registerAndLoginUser(t, srv, "usera@example.com", "passwordA123")
	_, tokenB := registerAndLoginUser(t, srv, "userb@example.com", "passwordB123")

	traderA := &config.TraderRecord{
		ID:                  "binance_deepseek_usera1",
		UserID:              userA,
		Name:                "User A Secret Trader",
		AIModelID:           userA + "_deepseek",
		ExchangeID:          "binance",
		InitialBalance:      5000,
		ScanIntervalMinutes: 3,
		BTCETHLeverage:      5,
		AltcoinLeverage:     5,
		CustomPrompt:        "secret strategy",
		IsCrossMargin:       true,
	}
	if err := db.CreateTrader(traderA); err != nil {
		t.Fatalf("CreateTrader failed: %v", err)
	}

	// Unauthenticated access to /api/equity-history must return 401.
	unauthEquity := doJSONRequest(t, srv, http.MethodGet, "/api/equity-history?trader_id="+traderA.ID, nil, "")
	if unauthEquity.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated /api/equity-history, got %d", unauthEquity.Code)
	}

	// User B attempting to query User A's trader must get 403 Forbidden.
	for _, endpoint := range []string{
		"/api/status?trader_id=" + traderA.ID,
		"/api/account?trader_id=" + traderA.ID,
		"/api/positions?trader_id=" + traderA.ID,
		"/api/decisions?trader_id=" + traderA.ID,
		"/api/equity-history?trader_id=" + traderA.ID,
		"/api/performance?trader_id=" + traderA.ID,
	} {
		rec := doJSONRequest(t, srv, http.MethodGet, endpoint, nil, tokenB)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("GET %s as User B: expected 403, got %d: %s", endpoint, rec.Code, rec.Body.String())
		}
	}

	// User B attempting to update User A's prompt must get 404 and leave prompt unchanged.
	promptRec := doJSONRequest(t, srv, http.MethodPut, "/api/traders/"+traderA.ID+"/prompt", map[string]interface{}{
		"custom_prompt":        "hijacked prompt",
		"override_base_prompt": true,
	}, tokenB)
	if promptRec.Code != http.StatusNotFound {
		t.Fatalf("PUT prompt as User B: expected 404, got %d", promptRec.Code)
	}

	// User B attempting to delete User A's trader must get 404 and leave trader intact.
	delRec := doJSONRequest(t, srv, http.MethodDelete, "/api/traders/"+traderA.ID, nil, tokenB)
	if delRec.Code != http.StatusNotFound {
		t.Fatalf("DELETE trader as User B: expected 404, got %d", delRec.Code)
	}

	tradersA, err := db.GetTraders(userA)
	if err != nil || len(tradersA) != 1 || tradersA[0].CustomPrompt != "secret strategy" {
		t.Fatalf("User A's trader was unexpectedly modified or deleted: %+v, err=%v", tradersA, err)
	}
}

func TestSecretMaskingAndPartialUpdatePreservation(t *testing.T) {
	srv, db := setupTestServer(t)
	userID, token := registerAndLoginUser(t, srv, "keys@example.com", "passwordKeys123")

	// Save initial model and exchange secrets.
	putModel := doJSONRequest(t, srv, http.MethodPut, "/api/models", map[string]interface{}{
		"models": map[string]interface{}{
			"deepseek": map[string]interface{}{
				"enabled":           true,
				"api_key":           "sk-live-deepseek-secret-value",
				"custom_api_url":    "https://api.deepseek.com/v1",
				"custom_model_name": "deepseek-chat",
			},
		},
	}, token)
	if putModel.Code != http.StatusOK {
		t.Fatalf("PUT /api/models failed (%d): %s", putModel.Code, putModel.Body.String())
	}

	putEx := doJSONRequest(t, srv, http.MethodPut, "/api/exchanges", map[string]interface{}{
		"exchanges": map[string]interface{}{
			"aster": map[string]interface{}{
				"enabled":           true,
				"testnet":           false,
				"aster_user":        "0xuser",
				"aster_signer":      "0xsigner",
				"aster_private_key": "deadbeefprivkey",
			},
		},
	}, token)
	if putEx.Code != http.StatusOK {
		t.Fatalf("PUT /api/exchanges failed (%d): %s", putEx.Code, putEx.Body.String())
	}

	// GET /api/models must return masked sentinel instead of raw key.
	getModels := doJSONRequest(t, srv, http.MethodGet, "/api/models", nil, token)
	if getModels.Code != http.StatusOK {
		t.Fatalf("GET /api/models failed: %d", getModels.Code)
	}
	if bytes.Contains(getModels.Body.Bytes(), []byte("sk-live-deepseek-secret-value")) {
		t.Fatal("GET /api/models leaked raw API key")
	}
	if !bytes.Contains(getModels.Body.Bytes(), []byte(MaskedSecretSentinel)) {
		t.Fatalf("expected masked sentinel in GET /api/models, got: %s", getModels.Body.String())
	}

	// GET /api/exchanges must return masked sentinel instead of raw Aster private key.
	getEx := doJSONRequest(t, srv, http.MethodGet, "/api/exchanges", nil, token)
	if getEx.Code != http.StatusOK {
		t.Fatalf("GET /api/exchanges failed: %d", getEx.Code)
	}
	if bytes.Contains(getEx.Body.Bytes(), []byte("deadbeefprivkey")) {
		t.Fatal("GET /api/exchanges leaked raw aster private key")
	}

	// Partial edit sending masked model key must preserve the raw key in DB.
	putModelEdit := doJSONRequest(t, srv, http.MethodPut, "/api/models", map[string]interface{}{
		"models": map[string]interface{}{
			"deepseek": map[string]interface{}{
				"enabled":           true,
				"api_key":           MaskedSecretSentinel,
				"custom_api_url":    "https://api.deepseek.com/v1",
				"custom_model_name": "deepseek-reasoner",
			},
		},
	}, token)
	if putModelEdit.Code != http.StatusOK {
		t.Fatalf("partial PUT /api/models failed (%d): %s", putModelEdit.Code, putModelEdit.Body.String())
	}

	modelsInDB, err := db.GetAIModels(userID)
	if err != nil || len(modelsInDB) != 1 || modelsInDB[0].APIKey != "sk-live-deepseek-secret-value" {
		t.Fatalf("expected raw model key preserved in DB, got: %+v err=%v", modelsInDB, err)
	}

	// Partial edit toggling Aster testnet with empty aster_private_key must preserve stored private key.
	putExEdit := doJSONRequest(t, srv, http.MethodPut, "/api/exchanges", map[string]interface{}{
		"exchanges": map[string]interface{}{
			"aster": map[string]interface{}{
				"enabled":           true,
				"testnet":           true,
				"aster_user":        "0xuser",
				"aster_signer":      "0xsigner",
				"aster_private_key": "",
			},
		},
	}, token)
	if putExEdit.Code != http.StatusOK {
		t.Fatalf("partial PUT /api/exchanges failed (%d): %s", putExEdit.Code, putExEdit.Body.String())
	}

	exInDB, err := db.GetExchanges(userID)
	if err != nil || len(exInDB) != 1 || exInDB[0].AsterPrivateKey != "deadbeefprivkey" || !exInDB[0].Testnet {
		t.Fatalf("expected aster private key preserved and testnet=true, got: %+v err=%v", exInDB, err)
	}

	// Disabling/deleting exchange with enabled=false and empty secret must clear it.
	putExDelete := doJSONRequest(t, srv, http.MethodPut, "/api/exchanges", map[string]interface{}{
		"exchanges": map[string]interface{}{
			"aster": map[string]interface{}{
				"enabled":           false,
				"aster_private_key": "",
			},
		},
	}, token)
	if putExDelete.Code != http.StatusOK {
		t.Fatalf("disable PUT /api/exchanges failed (%d): %s", putExDelete.Code, putExDelete.Body.String())
	}

	exAfterDelete, err := db.GetExchanges(userID)
	if err != nil || len(exAfterDelete) != 1 || exAfterDelete[0].AsterPrivateKey != "" {
		t.Fatalf("expected aster private key cleared on disable, got: %+v err=%v", exAfterDelete, err)
	}
}

func TestSSRFURLValidation(t *testing.T) {
	t.Setenv("ALLOW_PRIVATE_URLS", "")

	blocked := []string{
		"http://127.0.0.1:8080/secret",
		"http://localhost/admin",
		"http://169.254.169.254/latest/meta-data/",
		"http://10.0.0.1/internal",
		"http://172.16.0.10/internal",
		"http://192.168.1.1/router",
		"http://[::1]:8080/",
		"http://metadata.google.internal/computeMetadata/v1/",
		"file:///etc/passwd",
		"gopher://127.0.0.1:6379/_INFO",
	}
	for _, raw := range blocked {
		if err := ValidateExternalURL(raw); err == nil {
			t.Errorf("expected URL %q to be blocked", raw)
		}
	}

	allowed := []string{
		"",
		"https://api.deepseek.com/v1",
		"https://openrouter.ai/api/v1/chat/completions#",
		"https://signals.example.com/coin-pool",
	}
	for _, raw := range allowed {
		if err := ValidateExternalURL(raw); err != nil {
			t.Errorf("expected URL %q to be allowed, got error: %v", raw, err)
		}
	}

	// Endpoint enforcement test.
	srv, _ := setupTestServer(t)
	_, token := registerAndLoginUser(t, srv, "ssrf@example.com", "passwordSSRF123")

	rec := doJSONRequest(t, srv, http.MethodPost, "/api/user/signal-sources", map[string]string{
		"coin_pool_url": "http://169.254.169.254/latest/meta-data/",
		"oi_top_url":    "",
	}, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for metadata SSRF URL on signal-sources, got %d", rec.Code)
	}

	// Explicit opt-in allows private URLs (e.g. local Ollama).
	t.Setenv("ALLOW_PRIVATE_URLS", "true")
	if err := ValidateExternalURL("http://127.0.0.1:11434/v1"); err != nil {
		t.Fatalf("expected localhost URL allowed when ALLOW_PRIVATE_URLS=true, got %v", err)
	}
}
