package api_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/garnizeh/rag/api"
	"github.com/golang-jwt/jwt/v5"
)

func TestLoggingMiddleware(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	handler := api.LoggingMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/log", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}
	b, _ := io.ReadAll(res.Body)
	if string(b) != "ok" {
		t.Fatalf("unexpected body: %q", string(b))
	}
}

func TestCORSMiddleware(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := api.CORSMiddleware(next)

	// OPTIONS should return 204 and not call next
	reqOpt := httptest.NewRequest(http.MethodOptions, "/cors", nil)
	wOpt := httptest.NewRecorder()
	handler.ServeHTTP(wOpt, reqOpt)
	resOpt := wOpt.Result()
	defer resOpt.Body.Close()
	if resOpt.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for OPTIONS, got %d", resOpt.StatusCode)
	}
	if got := resOpt.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected CORS header set, got %q", got)
	}

	// GET should pass through and set headers
	reqGet := httptest.NewRequest(http.MethodGet, "/cors", nil)
	wGet := httptest.NewRecorder()
	handler.ServeHTTP(wGet, reqGet)
	resGet := wGet.Result()
	defer resGet.Body.Close()
	if resGet.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for GET, got %d", resGet.StatusCode)
	}
	if got := resGet.Header.Get("Access-Control-Allow-Methods"); !strings.Contains(got, "GET") {
		t.Fatalf("expected Allow-Methods to include GET, got %q", got)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	// handler that panics
	pan := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})
	handler := api.RecoveryMiddleware(pan)
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500 from panic recovery, got %d", res.StatusCode)
	}
	b, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(b), "Internal Server Error") {
		t.Fatalf("unexpected body for recovery: %s", string(b))
	}

	// normal handler should pass through
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	handler2 := api.RecoveryMiddleware(ok)
	w2 := httptest.NewRecorder()
	handler2.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/ok", nil))
	if w2.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for normal path, got %d", w2.Result().StatusCode)
	}
}

func TestJWTAuthMiddlewareWithSecret(t *testing.T) {
	secret := "s3cr3t"
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mw := api.JWTAuthMiddlewareWithSecret(secret)
	handler := mw(next)

	cases := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{name: "MissingHeader", authHeader: "", wantStatus: http.StatusUnauthorized},
		{name: "EmptyBearer", authHeader: "Bearer ", wantStatus: http.StatusUnauthorized},
		{name: "BadToken", authHeader: "Bearer bad.token.here", wantStatus: http.StatusUnauthorized},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/jwt", nil)
			if c.authHeader != "" {
				req.Header.Set("Authorization", c.authHeader)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Result().StatusCode != c.wantStatus {
				t.Fatalf("%s: want %d got %d", c.name, c.wantStatus, w.Result().StatusCode)
			}
		})
	}

	// now test valid token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"email": "a@b", "exp": time.Now().Add(time.Hour).Unix()})
	tokStr, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/jwt", nil)
	req.Header.Set("Authorization", "Bearer "+tokStr)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("valid token: expected 200 got %d", w.Result().StatusCode)
	}
}
