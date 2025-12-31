package helpers

import (
	"github.com/google/uuid"
	"strings"
)

// GenerateUUID genera un UUID en formato estándar (minúsculas)
func GenerateUUID() string {
	return uuid.New().String()
}

// GenerateUUIDUpper genera un UUID en formato ASP.NET Identity (mayúsculas con guiones)
func GenerateUUIDUpper() string {
	return strings.ToUpper(uuid.New().String())
}
