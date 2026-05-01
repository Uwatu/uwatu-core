package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the data we want to store inside our JWT token.
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	// Standard claims include expiration time, issued at time, etc.
	jwt.RegisteredClaims
}

// GenerateToken creates a new JWT string signed with our secret key.
func GenerateToken(userID string, role string, secret string, duration time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString([]byte(secret))

	return signedToken, err
}

// ValidateToken parses a JWT string and extracts the claims if the signature is valid.
func ValidateToken(tokenString string, secret string) (*Claims, error) {
	// Parse the token string into a token object, providing a key function to verify the signature.
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {

		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// If it is valid, return the secret as a byte slice so the parser can check the signature.
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
