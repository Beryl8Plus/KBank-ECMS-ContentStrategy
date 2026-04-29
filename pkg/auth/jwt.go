package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims represents the JWT claims structure for user authentication
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	jwt.RegisteredClaims
}

// ClientClaims represents the JWT claims structure for client credentials flow
type ClientClaims struct {
	ClientID string   `json:"client_id"`
	Scopes   []string `json:"scopes"`
	jwt.RegisteredClaims
}

// ClientConfig holds client configuration for OAuth2 client credentials flow
type ClientConfig struct {
	ClientID     string
	ClientSecret string
	Scopes       []string
}

// ClientRegistry manages client credentials
type ClientRegistry struct {
	clients map[string]*ClientConfig
}

// NewClientRegistry creates a new client registry
func NewClientRegistry(clients map[string]*ClientConfig) *ClientRegistry {
	return &ClientRegistry{
		clients: clients,
	}
}

// ValidateClient validates client credentials
func (r *ClientRegistry) ValidateClient(clientID, clientSecret string) (*ClientConfig, bool) {
	client, exists := r.clients[clientID]
	if !exists {
		return nil, false
	}
	if client.ClientSecret != clientSecret {
		return nil, false
	}
	return client, true
}

// GetClient retrieves client configuration by ID
func (r *ClientRegistry) GetClient(clientID string) (*ClientConfig, bool) {
	client, exists := r.clients[clientID]
	return client, exists
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	SecretKey     string
	TokenDuration time.Duration
}

// JWTService handles JWT token operations
type JWTService struct {
	config JWTConfig
}

// NewJWTService creates a new JWT service instance
func NewJWTService(config JWTConfig) *JWTService {
	return &JWTService{
		config: config,
	}
}

// GenerateToken generates a new JWT token for a user
func (j *JWTService) GenerateToken(userID uuid.UUID, email string) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.config.TokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(j.config.SecretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// VerifyToken verifies a JWT token and returns the claims
func (j *JWTService) VerifyToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.config.SecretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

// RefreshToken generates a new token with extended expiration
func (j *JWTService) RefreshToken(tokenString string) (string, error) {
	claims, err := j.VerifyToken(tokenString)
	if err != nil {
		return "", err
	}

	// Generate new token with same user info but new expiration
	return j.GenerateToken(claims.UserID, claims.Email)
}

// GenerateClientToken generates a JWT token for client credentials flow
func (j *JWTService) GenerateClientToken(clientID string, scopes []string) (string, error) {
	claims := ClientClaims{
		ClientID: clientID,
		Scopes:   scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.config.TokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(j.config.SecretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// VerifyClientToken verifies a client JWT token and returns the client claims
func (j *JWTService) VerifyClientToken(tokenString string) (*ClientClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &ClientClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.config.SecretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*ClientClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

// HasScope checks if the client token has the required scope
func (j *JWTService) HasScope(claims *ClientClaims, requiredScope string) bool {
	for _, scope := range claims.Scopes {
		if scope == requiredScope {
			return true
		}
	}
	return false
}
