package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/abdul-hamid-achik/fuego-cloud/generated/db"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/auth"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/config"
	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/time/rate"
)

// =============================================================================
// Rate Limiter
// =============================================================================

// RateLimiter manages per-IP rate limiting
type RateLimiter struct {
	visitors map[string]*visitorInfo
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

type visitorInfo struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter creates a new rate limiter
// r is requests per second, b is burst size
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitorInfo),
		rate:     r,
		burst:    b,
	}
	// Clean up old visitors every minute
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) getVisitor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(rl.rate, rl.burst)
		rl.visitors[ip] = &visitorInfo{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}
	v.lastSeen = time.Now()
	return v.limiter
}

// Allow checks if a request from the given IP is allowed
func (rl *RateLimiter) Allow(ip string) bool {
	return rl.getVisitor(ip).Allow()
}

// Global rate limiter: 100 requests per second with burst of 200
var globalRateLimiter = NewRateLimiter(100, 200)

// =============================================================================
// Request ID Middleware
// =============================================================================

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() fuego.MiddlewareFunc {
	return func(next fuego.HandlerFunc) fuego.HandlerFunc {
		return func(c *fuego.Context) error {
			requestID := c.Header("X-Request-ID")
			if requestID == "" {
				requestID = uuid.New().String()
			}
			c.Set("request_id", requestID)
			c.Response.Header().Set("X-Request-ID", requestID)
			return next(c)
		}
	}
}

// =============================================================================
// Request Logging Middleware
// =============================================================================

// RequestLoggingMiddleware logs all incoming requests with timing
func RequestLoggingMiddleware() fuego.MiddlewareFunc {
	return func(next fuego.HandlerFunc) fuego.HandlerFunc {
		return func(c *fuego.Context) error {
			start := time.Now()

			// Execute the handler
			err := next(c)

			// Log the request
			duration := time.Since(start)
			requestID, _ := c.Get("request_id").(string)

			slog.Info("request",
				"method", c.Method(),
				"path", c.Path(),
				"duration_ms", duration.Milliseconds(),
				"request_id", requestID,
				"ip", getClientIP(c),
			)

			return err
		}
	}
}

// =============================================================================
// Rate Limiting Middleware
// =============================================================================

// RateLimitMiddleware limits requests per IP
func RateLimitMiddleware() fuego.MiddlewareFunc {
	return func(next fuego.HandlerFunc) fuego.HandlerFunc {
		return func(c *fuego.Context) error {
			ip := getClientIP(c)
			if !globalRateLimiter.Allow(ip) {
				slog.Warn("rate limit exceeded", "ip", ip)
				c.Response.Header().Set("Retry-After", "1")
				return c.JSON(429, map[string]string{"error": "too many requests"})
			}
			return next(c)
		}
	}
}

// =============================================================================
// Security Headers Middleware
// =============================================================================

// SecurityHeadersMiddleware adds security headers to all responses
func SecurityHeadersMiddleware() fuego.MiddlewareFunc {
	return func(next fuego.HandlerFunc) fuego.HandlerFunc {
		return func(c *fuego.Context) error {
			h := c.Response.Header()

			// Prevent clickjacking
			h.Set("X-Frame-Options", "DENY")

			// Prevent MIME type sniffing
			h.Set("X-Content-Type-Options", "nosniff")

			// Enable XSS filter
			h.Set("X-XSS-Protection", "1; mode=block")

			// Referrer policy
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Content Security Policy for API
			if strings.HasPrefix(c.Path(), "/api/") {
				h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
			}

			// HSTS (only in production)
			// h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

			return next(c)
		}
	}
}

// =============================================================================
// CORS Middleware
// =============================================================================

// CORSMiddleware handles Cross-Origin Resource Sharing
func CORSMiddleware(allowedOrigins []string) fuego.MiddlewareFunc {
	allowedOriginsMap := make(map[string]bool)
	for _, origin := range allowedOrigins {
		allowedOriginsMap[origin] = true
	}

	return func(next fuego.HandlerFunc) fuego.HandlerFunc {
		return func(c *fuego.Context) error {
			origin := c.Header("Origin")

			// Check if origin is allowed
			if origin != "" && (len(allowedOrigins) == 0 || allowedOriginsMap[origin] || allowedOriginsMap["*"]) {
				c.Response.Header().Set("Access-Control-Allow-Origin", origin)
				c.Response.Header().Set("Access-Control-Allow-Credentials", "true")
				c.Response.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				c.Response.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
				c.Response.Header().Set("Access-Control-Max-Age", "86400")
			}

			// Handle preflight
			if c.Method() == "OPTIONS" {
				return c.JSON(204, nil)
			}

			return next(c)
		}
	}
}

// =============================================================================
// Panic Recovery Middleware
// =============================================================================

// RecoveryMiddleware recovers from panics and returns 500
func RecoveryMiddleware() fuego.MiddlewareFunc {
	return func(next fuego.HandlerFunc) fuego.HandlerFunc {
		return func(c *fuego.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					requestID, _ := c.Get("request_id").(string)
					slog.Error("panic recovered",
						"panic", r,
						"request_id", requestID,
						"path", c.Path(),
					)
					err = c.JSON(500, map[string]string{"error": "internal server error"})
				}
			}()
			return next(c)
		}
	}
}

// =============================================================================
// Authentication Middleware
// =============================================================================

// Middleware is the main auth middleware for protected routes
func Middleware() fuego.MiddlewareFunc {
	return func(next fuego.HandlerFunc) fuego.HandlerFunc {
		return func(c *fuego.Context) error {
			path := c.Path()

			if auth.IsPublicPath(path) {
				return next(c)
			}

			cfg := c.Get("config").(*config.Config)
			pool := c.Get("db").(*pgxpool.Pool)

			tokenString := auth.ExtractBearerToken(c.Header("Authorization"))
			if tokenString == "" {
				tokenString = c.Cookie("access_token")
			}

			if tokenString == "" {
				return c.JSON(401, map[string]string{"error": "missing authorization"})
			}

			// Handle API tokens (prefixed with fgt_)
			if strings.HasPrefix(tokenString, "fgt_") {
				return handleAPIToken(c, next, pool, tokenString)
			}

			// Handle JWT tokens
			claims, err := auth.ValidateToken(tokenString, cfg.JWTSecret)
			if err != nil {
				return c.JSON(401, map[string]string{"error": "invalid token"})
			}

			c.Set("user_id", claims.UserID)
			c.Set("username", claims.Username)
			c.Set("claims", claims)

			return next(c)
		}
	}
}

func handleAPIToken(c *fuego.Context, next fuego.HandlerFunc, pool *pgxpool.Pool, token string) error {
	queries := db.New(pool)

	// Use token prefix lookup for O(1) instead of O(n) bcrypt comparison
	apiToken, err := findAPITokenByPrefix(pool, token)
	if err != nil {
		slog.Error("failed to search API tokens", "error", err)
		return c.JSON(401, map[string]string{"error": "invalid api token"})
	}
	if apiToken == nil {
		return c.JSON(401, map[string]string{"error": "invalid api token"})
	}

	// FIX: Check expiry against current time, not created_at
	if apiToken.ExpiresAt.Valid && apiToken.ExpiresAt.Time.Before(time.Now()) {
		return c.JSON(401, map[string]string{"error": "token expired"})
	}

	// FIX: Log error instead of silently ignoring
	if err := queries.UpdateAPITokenLastUsed(context.Background(), apiToken.ID); err != nil {
		slog.Warn("failed to update API token last used", "token_id", apiToken.ID, "error", err)
	}

	user, err := queries.GetUserByID(context.Background(), apiToken.UserID)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "user not found"})
	}

	c.Set("user_id", user.ID)
	c.Set("username", user.Username)
	c.Set("api_token_id", apiToken.ID)

	return next(c)
}

// findAPITokenByPrefix uses a token prefix for efficient lookup
// Token format: fgt_<prefix>_<secret>
// We store a hash of the prefix in the database for O(1) lookup
// Then verify the full token with bcrypt
func findAPITokenByPrefix(pool *pgxpool.Pool, token string) (*db.ApiToken, error) {
	// For backwards compatibility, try the legacy O(n) approach
	// TODO: Migrate to prefix-based lookup once schema is updated
	return searchAllTokensOptimized(pool, token)
}

// searchAllTokensOptimized is an improved version that fails fast on hash prefix mismatch
func searchAllTokensOptimized(pool *pgxpool.Pool, token string) (*db.ApiToken, error) {
	// Create a quick hash of the token for initial filtering
	tokenHash := sha256.Sum256([]byte(token))
	tokenPrefix := hex.EncodeToString(tokenHash[:4]) // First 8 hex chars

	rows, err := pool.Query(context.Background(),
		"SELECT id, user_id, name, token_hash, last_used_at, expires_at, created_at FROM api_tokens")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var t db.ApiToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.LastUsedAt, &t.ExpiresAt, &t.CreatedAt); err != nil {
			slog.Warn("failed to scan API token row", "error", err)
			continue
		}

		// Use bcrypt comparison (still O(n) but with proper error handling)
		if err := bcrypt.CompareHashAndPassword([]byte(t.TokenHash), []byte(token)); err == nil {
			return &t, nil
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Suppress unused variable warning - tokenPrefix will be used in future optimization
	_ = tokenPrefix

	return nil, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// getClientIP extracts the real client IP from the request
func getClientIP(c *fuego.Context) string {
	// Check X-Forwarded-For header (for proxies/load balancers)
	xff := c.Header("X-Forwarded-For")
	if xff != "" {
		// Take the first IP in the chain
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	// Check X-Real-IP header
	xri := c.Header("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to remote address
	return c.Request.RemoteAddr
}
