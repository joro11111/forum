package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// CheckPassword compares a password with a hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateUUID generates a UUID-like string for sessions
func GenerateUUID() (string, error) {
	// Generate 16 random bytes
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// Set version (4) and variant bits
	bytes[6] = (bytes[6] & 0x0f) | 0x40 // Version 4
	bytes[8] = (bytes[8] & 0x3f) | 0x80 // Variant 10

	// Format as UUID string
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16]), nil
}

// GenerateSessionToken generates a secure session token
func GenerateSessionToken() (string, error) {
	// Generate random bytes
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// Hash the bytes with current timestamp for additional entropy
	hash := sha256.Sum256(append(bytes, []byte(time.Now().String())...))
	return hex.EncodeToString(hash[:]), nil
}

// ValidateEmail performs basic email validation
func ValidateEmail(email string) bool {
	if len(email) < 5 || len(email) > 254 {
		return false
	}

	// Check for @ symbol
	atCount := 0
	atIndex := -1
	for i, char := range email {
		if char == '@' {
			atCount++
			atIndex = i
		}
	}

	if atCount != 1 || atIndex == 0 || atIndex == len(email)-1 {
		return false
	}

	// Basic format check
	localPart := email[:atIndex]
	domainPart := email[atIndex+1:]

	if len(localPart) == 0 || len(domainPart) == 0 {
		return false
	}

	// Check domain has at least one dot
	hasDot := false
	for _, char := range domainPart {
		if char == '.' {
			hasDot = true
			break
		}
	}

	return hasDot
}

// ValidatePassword checks password strength
func ValidatePassword(password string) error {
	if len(password) < 6 {
		return fmt.Errorf("password must be at least 6 characters long")
	}
	if len(password) > 128 {
		return fmt.Errorf("password is too long")
	}
	return nil
}

// ValidateUsername checks username validity
func ValidateUsername(username string) error {
	if len(username) < 3 {
		return fmt.Errorf("username must be at least 3 characters long")
	}
	if len(username) > 50 {
		return fmt.Errorf("username is too long")
	}

	// Check for valid characters (alphanumeric, underscores, hyphens)
	for _, char := range username {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '-') {
			return fmt.Errorf("username can only contain letters, numbers, underscores, and hyphens")
		}
	}

	return nil
}
