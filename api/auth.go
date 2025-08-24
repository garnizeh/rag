package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/garnizeh/rag/internal/db"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	DB            *db.DB
	JWTSecret     string
	TokenDuration time.Duration
}

type signupRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type signinRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string `json:"token"`
}

func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Email == "" || req.Password == "" {
		http.Error(w, "Missing fields", http.StatusBadRequest)
		return
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	now := time.Now().UTC().UnixMilli()

	// Insert engineer with password_hash and return the new id (SQLite RETURNING)
	var engineerID int64
	if err = h.DB.QueryRow(
		ctx,
		"INSERT INTO engineers (name, email, updated, password_hash) VALUES (?, ?, ?, ?) RETURNING id",
		req.Name,
		req.Email,
		now,
		string(hash),
	).Scan(&engineerID); err != nil {
		http.Error(w, "Error creating user", http.StatusInternalServerError)
		return
	}

	// Create an empty profile row linked to the new engineer id
	if _, err := h.DB.Exec(
		ctx,
		"INSERT INTO engineer_profiles (engineer_id, bio, updated) VALUES (?, ?, ?)",
		engineerID,
		"{}",
		now,
	); err != nil {
		http.Error(w, "Error creating user profile", http.StatusInternalServerError)
		return
	}

	// Issue JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": req.Email,
		"exp":   time.Now().Add(h.TokenDuration).Unix(),
	})
	tokenStr, err := token.SignedString([]byte(h.JWTSecret))
	if err != nil {
		http.Error(w, "Error signing token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(authResponse{Token: tokenStr})
}

func (h *AuthHandler) Signin(w http.ResponseWriter, r *http.Request) {
	var req signinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" {
		http.Error(w, "Missing fields", http.StatusBadRequest)
		return
	}

	// Get password hash from engineers table
	var hash string
	err := h.DB.QueryRow(r.Context(),
		"SELECT password_hash FROM engineers WHERE email = ?", req.Email).Scan(&hash)
	if err != nil {
		http.Error(w, "Credentials not found", http.StatusUnauthorized)
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		http.Error(w, "Credentials not found", http.StatusUnauthorized)
		return
	}

	// Issue JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": req.Email,
		"exp":   time.Now().Add(h.TokenDuration).Unix(),
	})
	tokenStr, err := token.SignedString([]byte(h.JWTSecret))
	if err != nil {
		http.Error(w, "Error signing token", http.StatusInternalServerError)
		return
	}
	
	json.NewEncoder(w).Encode(authResponse{Token: tokenStr})
}

func (h *AuthHandler) Signout(w http.ResponseWriter, r *http.Request) {
	// For stateless JWT, signout is client-side (just delete token)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, `{"message":"signed out"}`)
}
