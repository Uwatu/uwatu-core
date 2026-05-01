package auth

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Protected creates a middleware that validates JWTs in the Authorization header.
func Protected(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		//Get the Authorization header from the request.
		authHeader := c.Get("Authorization")

		// Check if the header is empty or doesn't start with "Bearer ".
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing or malformed jwt",
			})
		}

		// Extract the actual token string.
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid authorization header format",
			})
		}
		tokenString := parts[1]

		claims, err := ValidateToken(tokenString, secret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid token: " + err.Error(),
			})
		}

		c.Locals("user_id", claims.UserID)
		c.Locals("role", claims.Role)

		// Continue to the next middleware or route handler.
		return c.Next()
	}
}

// RequireRole creates a middleware that restricts access to specific roles.
// It must be used AFTER the Protected middleware.
func RequireRole(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole, ok := c.Locals("role").(string)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "user role not found in context",
			})
		}

		// Check if the userRole exists in the allowedRoles slice.
		for _, role := range allowedRoles {
			if role == userRole {
				// Match found! Pass control to the next handler.
				return c.Next()
			}
		}

		// If the loop finishes without finding a match, return a 403 Forbidden.
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "insufficient permissions",
		})
	}
}
