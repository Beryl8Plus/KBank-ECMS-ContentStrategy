package handler

import (
	"net/http"
	"strings"

	domainrepo "kbank-ecms/internal/domain/repository"
	"kbank-ecms/pkg/auth"

	"github.com/gin-gonic/gin"
)

// TokenHandler handles OAuth2 token endpoint for client credentials flow.
// It validates client credentials against the database and generates JWT
// tokens with scopes derived from the client's profile permissions.
type TokenHandler struct {
	jwtService       *auth.JWTService
	oauth2ClientRepo domainrepo.OAuth2ClientRepository
}

// NewTokenHandler creates a new token handler.
func NewTokenHandler(jwtService *auth.JWTService, oauth2ClientRepo domainrepo.OAuth2ClientRepository) *TokenHandler {
	return &TokenHandler{
		jwtService:       jwtService,
		oauth2ClientRepo: oauth2ClientRepo,
	}
}

// TokenRequest represents the token request body
type TokenRequest struct {
	GrantType    string `form:"grant_type" binding:"required"`
	ClientID     string `form:"client_id" binding:"required"`
	ClientSecret string `form:"client_secret" binding:"required"`
	Scope        string `form:"scope"`
}

// TokenResponse represents the OAuth2 token response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	Scope       string `json:"scope,omitempty"`
}

// HandleToken handles POST /token endpoint for client credentials flow.
// It looks up the client in the database, validates the secret, and issues
// a JWT token whose scopes are derived from the client's profile permissions.
func (h *TokenHandler) HandleToken(c *gin.Context) {
	var req TokenRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	// Validate grant type
	if req.GrantType != "client_credentials" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_grant_type"})
		return
	}

	// Look up client from database
	client, err := h.oauth2ClientRepo.GetByClientID(c.Request.Context(), req.ClientID)
	if err != nil || client == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}

	// Validate client secret (plain comparison; replace with bcrypt in production)
	if client.ClientSecret != req.ClientSecret {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}

	// Resolve scopes from database based on profile permissions
	scopes, err := h.oauth2ClientRepo.GetClientScopes(c.Request.Context(), client.ClientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	// Generate JWT token for client with resolved scopes
	token, err := h.jwtService.GenerateClientToken(client.ClientID, scopes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	// Return OAuth2 compliant response
	response := TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   86400, // 24 hours in seconds
		Scope:       strings.Join(scopes, " "),
	}

	c.JSON(http.StatusOK, response)
}
