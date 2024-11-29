package tools

import (
	"github.com/google/uuid"
)

// Generates a random name using UUID
func GenerateRandomName() (string, error) {
	// Generate a UUID
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	// Return the UUID as a string
	return id.String(), nil
}
