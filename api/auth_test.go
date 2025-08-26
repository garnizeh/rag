package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/garnizeh/rag/api"
	"github.com/garnizeh/rag/internal/models"
	"github.com/garnizeh/rag/pkg/repository/mock"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthHandlers(t *testing.T) {
	secret := "testsecret"
	tokenDur := 1 * time.Hour

	tests := []struct {
		name       string
		method     string
		path       string
		body       any
		prepare    func(m *mock.Mocks)
		wantStatus int
		checkBody  func(t *testing.T, body []byte)
	}{
		{
			name:       "Signup_InvalidRequest",
			method:     http.MethodPost,
			path:       "/signup",
			body:       "not a json",
			prepare:    func(m *mock.Mocks) {},
			wantStatus: http.StatusBadRequest,
			checkBody:  func(t *testing.T, b []byte) {},
		},
		{
			name:       "Signup_MissingFields_Name",
			method:     http.MethodPost,
			path:       "/signup",
			body:       map[string]string{"email": "alice@example.com", "password": "s3cret"},
			prepare:    func(m *mock.Mocks) {},
			wantStatus: http.StatusBadRequest,
			checkBody:  func(t *testing.T, b []byte) {},
		},
		{
			name:       "Signup_MissingFields_Email",
			method:     http.MethodPost,
			path:       "/signup",
			body:       map[string]string{"name": "Alice", "password": "s3cret"},
			prepare:    func(m *mock.Mocks) {},
			wantStatus: http.StatusBadRequest,
			checkBody:  func(t *testing.T, b []byte) {},
		},
		{
			name:       "Signup_MissingFields_Paswword",
			method:     http.MethodPost,
			path:       "/signup",
			body:       map[string]string{"name": "Alice", "email": "alice@example.com"},
			prepare:    func(m *mock.Mocks) {},
			wantStatus: http.StatusBadRequest,
			checkBody:  func(t *testing.T, b []byte) {},
		},
		{
			name:       "Signup_Success",
			method:     http.MethodPost,
			path:       "/signup",
			body:       map[string]string{"name": "Alice", "email": "alice@example.com", "password": "s3cret"},
			prepare:    func(m *mock.Mocks) {},
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, b []byte) {
				var ar struct {
					Token string `json:"token"`
				}
				if err := json.Unmarshal(b, &ar); err != nil {
					t.Fatalf("unmarshal token: %v", err)
				}
				if ar.Token == "" {
					t.Fatalf("empty token")
				}
				if _, err := jwt.Parse(ar.Token, func(token *jwt.Token) (any, error) { return []byte(secret), nil }); err != nil {
					t.Fatalf("invalid token: %v", err)
				}
			},
		},
		{
			name:   "Signup_DuplicateEmail",
			method: http.MethodPost,
			path:   "/signup",
			body:   map[string]string{"name": "Dup", "email": "dup@example.com", "password": "pw"},
			prepare: func(m *mock.Mocks) {
				m.EngRepo.CreateErr = fmt.Errorf("unique constraint")
			},
			wantStatus: http.StatusInternalServerError,
			checkBody:  func(t *testing.T, b []byte) {},
		},
		{
			name:       "Signin_InvalidRequest",
			method:     http.MethodPost,
			path:       "/signin",
			body:       "not a json",
			prepare:    func(m *mock.Mocks) {},
			wantStatus: http.StatusBadRequest,
			checkBody:  func(t *testing.T, b []byte) {},
		},
		{
			name:       "Signin_MissingFields_Email",
			method:     http.MethodPost,
			path:       "/signin",
			body:       map[string]string{"password": "nop"},
			prepare:    func(m *mock.Mocks) {},
			wantStatus: http.StatusBadRequest,
			checkBody:  func(t *testing.T, b []byte) {},
		},
		{
			name:       "Signin_MissingFields_Paswword",
			method:     http.MethodPost,
			path:       "/signin",
			body:       map[string]string{"email": "missing@example.com"},
			prepare:    func(m *mock.Mocks) {},
			wantStatus: http.StatusBadRequest,
			checkBody:  func(t *testing.T, b []byte) {},
		},
		{
			name:   "Signin_MissingUser",
			method: http.MethodPost,
			path:   "/signin",
			body:   map[string]string{"email": "missing@example.com", "password": "nop"},
			prepare: func(m *mock.Mocks) {
				m.EngRepo.Stored = nil
			},
			wantStatus: http.StatusUnauthorized,
			checkBody:  func(t *testing.T, b []byte) {},
		},
		{
			name:   "Signin_Success",
			method: http.MethodPost,
			path:   "/signin",
			body:   map[string]string{"email": "bob@example.com", "password": "hunter2"},
			prepare: func(m *mock.Mocks) {
				pw := "hunter2"
				hash, _ := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
				m.EngRepo.Stored = &models.Engineer{ID: 2, Email: "bob@example.com", PasswordHash: string(hash)}
			},
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, b []byte) {
				var ar struct {
					Token string `json:"token"`
				}
				if err := json.Unmarshal(b, &ar); err != nil {
					t.Fatalf("unmarshal token: %v", err)
				}
				if ar.Token == "" {
					t.Fatalf("empty token")
				}
				if _, err := jwt.Parse(ar.Token, func(token *jwt.Token) (any, error) { return []byte(secret), nil }); err != nil {
					t.Fatalf("invalid token: %v", err)
				}
			},
		},
		{
			name:   "Signin_WrongPassword",
			method: http.MethodPost,
			path:   "/signin",
			body:   map[string]string{"email": "c@example.com", "password": "wrongpw"},
			prepare: func(m *mock.Mocks) {
				pw := "rightpw"
				hash, _ := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
				m.EngRepo.Stored = &models.Engineer{ID: 3, Email: "c@example.com", PasswordHash: string(hash)}
			},
			wantStatus: http.StatusUnauthorized,
			checkBody:  func(t *testing.T, b []byte) {},
		},
		{
			name:       "Signout_OK",
			method:     http.MethodPost,
			path:       "/signout",
			body:       nil,
			prepare:    func(m *mock.Mocks) {},
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, b []byte) {
				if !bytes.Contains(b, []byte("signed out")) {
					t.Fatalf("unexpected body: %s", string(b))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocks := mock.NewMocks()
			if tt.prepare != nil {
				tt.prepare(mocks)
			}
			handler := api.NewAuthHandler(mocks.EngRepo, mocks.ProfRepo, secret, tokenDur)

			var bodyReader io.Reader
			if tt.body != nil {
				b, _ := json.Marshal(tt.body)
				bodyReader = bytes.NewReader(b)
			}
			req := httptest.NewRequest(tt.method, tt.path, bodyReader)
			w := httptest.NewRecorder()

			switch tt.path {
			case "/signup":
				handler.Signup(w, req)
			case "/signin":
				handler.Signin(w, req)
			case "/signout":
				handler.Signout(w, req)
			default:
				t.Fatalf("unknown path %s", tt.path)
			}

			res := w.Result()
			defer res.Body.Close()
			if res.StatusCode != tt.wantStatus {
				data, _ := io.ReadAll(res.Body)
				t.Fatalf("%s: expected status %d got %d body=%s", tt.name, tt.wantStatus, res.StatusCode, string(data))
			}
			data, _ := io.ReadAll(res.Body)
			if tt.checkBody != nil {
				tt.checkBody(t, data)
			}
			// If token present, validate claims for email and exp when applicable
			if tt.wantStatus == http.StatusOK && (tt.path == "/signup" || tt.path == "/signin") {
				var ar struct {
					Token string `json:"token"`
				}
				if err := json.Unmarshal(data, &ar); err == nil && ar.Token != "" {
					tok, err := jwt.Parse(ar.Token, func(token *jwt.Token) (any, error) { return []byte(secret), nil })
					if err != nil {
						t.Fatalf("parse token: %v", err)
					}
					if claims, ok := tok.Claims.(jwt.MapClaims); ok {
						if _, ok := claims["email"]; !ok {
							t.Fatalf("missing email claim")
						}
						if expF, ok := claims["exp"].(float64); !ok || int64(expF) < time.Now().Unix() {
							t.Fatalf("invalid exp claim")
						}
					}
				}
			}
		})
	}
}
