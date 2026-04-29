package handler

import (
	"net/http"

	"kbank-ecms/pkg/auth"

	"github.com/gin-gonic/gin"
)

// TokenHandler handles OAuth2 token endpoint for client credentials flow
type TokenHandler struct {
	jwtService     *auth.JWTService
	clientRegistry *auth.ClientRegistry
}

// NewTokenHandler creates a new token handler
func NewTokenHandler(jwtService *auth.JWTService, clientRegistry *auth.ClientRegistry) *TokenHandler {
	return &TokenHandler{
		jwtService:     jwtService,
		clientRegistry: clientRegistry,
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

// HandleToken handles POST /token endpoint for client credentials flow
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

	// Validate client credentials
	clientConfig, valid := h.clientRegistry.ValidateClient(req.ClientID, req.ClientSecret)
	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}

	// Generate JWT token for client
	token, err := h.jwtService.GenerateClientToken(clientConfig.ClientID, clientConfig.Scopes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	// Return OAuth2 compliant response
	response := TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   86400, // 24 hours in seconds
	}

	c.JSON(http.StatusOK, response)
}
