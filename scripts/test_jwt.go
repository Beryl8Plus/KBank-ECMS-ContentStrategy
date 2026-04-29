package main

import (
	"fmt"
	"time"

	"kbank-ecms/pkg/auth"

	"github.com/google/uuid"
)

func main() {
	// Create JWT service with the same configuration as the app
	config := auth.JWTConfig{
		SecretKey:     "your-secret-key-change-in-production",
		TokenDuration: 24 * time.Hour,
	}
	jwtService := auth.NewJWTService(config)

	fmt.Println("=== Testing User JWT Token ===")
	// Generate a test user token
	userID := uuid.New()
	email := "test@example.com"
	userToken, err := jwtService.GenerateToken(userID, email)
	if err != nil {
		fmt.Printf("Error generating user token: %v\n", err)
		return
	}

	fmt.Printf("Generated User JWT Token:\n%s\n", userToken)
	fmt.Printf("User ID: %s\n", userID)
	fmt.Printf("Email: %s\n", email)
	fmt.Printf("\nTest user token with curl:\n")
	fmt.Printf("curl -X GET \"http://localhost:8081/schedules\" -H \"Authorization: Bearer %s\"\n\n", userToken)

	fmt.Println("=== Testing OAuth2 Client Credentials Token ===")
	// Generate a test client token
	clientID := "service-cmc"
	scopes := []string{"read:rules", "write:rules"}
	clientToken, err := jwtService.GenerateClientToken(clientID, scopes)
	if err != nil {
		fmt.Printf("Error generating client token: %v\n", err)
		return
	}

	fmt.Printf("Generated Client JWT Token:\n%s\n", clientToken)
	fmt.Printf("Client ID: %s\n", clientID)
	fmt.Printf("Scopes: %v\n", scopes)
	fmt.Printf("\nTest client token with curl:\n")
	fmt.Printf("curl -X GET \"http://localhost:8081/schedules\" -H \"Authorization: Bearer %s\"\n", clientToken)

	fmt.Println("\n=== Testing OAuth2 Token Endpoint ===")
	fmt.Printf("Test token endpoint with curl:\n")
	fmt.Printf("curl -X POST \"http://localhost:8081/token\" \\\n")
	fmt.Printf("  -d \"grant_type=client_credentials\" \\\n")
	fmt.Printf("  -d \"client_id=service-cmc\" \\\n")
	fmt.Printf("  -d \"client_secret=super-secret-key-cmc\"\n")
}
