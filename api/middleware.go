package api

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"log/slog"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

type ctxKey string

const CtxEngineerID ctxKey = "engineer_id"

// package-level logger used by middleware and helpers; can be set via SetLogger from caller
var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

// SetLogger installs a logger for the api package. Passing nil is a no-op.
func SetLogger(l *slog.Logger) {
	if l != nil {
		logger = l
	}
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info("request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("remote", r.RemoteAddr),
		)
		next.ServeHTTP(w, r)
	})
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("panic", slog.Any("err", err))
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func JWTAuthMiddlewareWithSecret(secret string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
				return
			}

			var tokenString string
			if _, err := fmt.Sscanf(authHeader, "Bearer %s", &tokenString); err != nil {
				// If scanning fails, log and treat as invalid header
				logger.Error("failed to parse Authorization header", slog.Any("err", err), slog.String("header", authHeader))
			}

			if tokenString == "" {
				http.Error(w, "Invalid Authorization header", http.StatusUnauthorized)
				return
			}

			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}

				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Token is valid, extract engineer_id claim if present and put into context
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				if v, found := claims["engineer_id"]; found {
					// try to handle number types (float64) and ints
					switch id := v.(type) {
					case float64:
						ctx := context.WithValue(r.Context(), CtxEngineerID, int64(id))
						r = r.WithContext(ctx)
					case int64:
						ctx := context.WithValue(r.Context(), CtxEngineerID, id)
						r = r.WithContext(ctx)
					case int:
						ctx := context.WithValue(r.Context(), CtxEngineerID, int64(id))
						r = r.WithContext(ctx)
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
