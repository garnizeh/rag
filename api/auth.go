package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/garnizeh/rag/pkg/models"
	"github.com/garnizeh/rag/pkg/repository"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	engineerRepo  repository.EngineerRepo
	profileRepo   repository.ProfileRepo
	jwtSecret     string
	tokenDuration time.Duration
}

// NewAuthHandler creates a new AuthHandler with required dependencies.
func NewAuthHandler(er repository.EngineerRepo, pr repository.ProfileRepo, jwtSecret string, tokenDuration time.Duration) *AuthHandler {
	return &AuthHandler{engineerRepo: er, profileRepo: pr, jwtSecret: jwtSecret, tokenDuration: tokenDuration}
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

	// Insert engineer with password_hash and return the new id (SQLite RETURNING)
	engineer := models.Engineer{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hash),
	}

	engineerID, err := h.engineerRepo.CreateEngineer(ctx, &engineer)
	if err != nil {
		http.Error(w, "Error creating user", http.StatusInternalServerError)
		return
	}

	// Create an empty profile row linked to the new engineer id
	profile := models.Profile{
		EngineerID: engineerID,
		Bio:        "{}",
	}
	if _, err := h.profileRepo.CreateProfile(ctx, &profile); err != nil {
		http.Error(w, "Error creating user profile", http.StatusInternalServerError)
		return
	}

	// Issue JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": req.Email,
		"exp":   time.Now().Add(h.tokenDuration).Unix(),
	})
	tokenStr, err := token.SignedString([]byte(h.jwtSecret))
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

	ctx := r.Context()

	// Get password hash from engineers table
	engineer, err := h.engineerRepo.GetByEmail(ctx, req.Email)
	if err != nil || engineer == nil {
		http.Error(w, "Credentials not found", http.StatusUnauthorized)
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(engineer.PasswordHash), []byte(req.Password)) != nil {
		http.Error(w, "Credentials not found", http.StatusUnauthorized)
		return
	}

	// Issue JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": req.Email,
		"exp":   time.Now().Add(h.tokenDuration).Unix(),
	})
	tokenStr, err := token.SignedString([]byte(h.jwtSecret))
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
