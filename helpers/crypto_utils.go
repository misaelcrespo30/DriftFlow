package helpers

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"sync"

	"github.com/joho/godotenv"
)

var (
	secretKey []byte
	once      sync.Once
)

// initSecretKey carga el secreto una sola vez desde .env
func initSecretKey() {
	once.Do(func() {
		_ = godotenv.Load() // Ignorar si ya está cargado por main

		key := os.Getenv("ENCRYPTION_SECRET_KEY")
		if len(key) != 16 && len(key) != 24 && len(key) != 32 {
			panic("ENCRYPTION_SECRET_KEY debe tener 16, 24 o 32 bytes")
		}
		secretKey = []byte(key)
	})
}

// Encrypt cifra usando AES-GCM y la clave del entorno
func Encrypt(plaintext string) (string, error) {
	initSecretKey()
	return EncryptWithKey(plaintext, secretKey)
}

// Decrypt descifra usando AES-GCM y la clave del entorno
func Decrypt(encrypted string) (string, error) {
	initSecretKey()
	return DecryptWithKey(encrypted, secretKey)
}

func DecryptPtr(encrypted *string) (string, error) {
	if encrypted == nil || *encrypted == "" {
		return "", errors.New("missing secret")
	}
	return Decrypt(*encrypted)
}

// EncryptWithKey permite cifrar usando una clave explícita
func EncryptWithKey(plaintext string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptWithKey permite descifrar usando una clave explícita
func DecryptWithKey(encrypted string, key []byte) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
