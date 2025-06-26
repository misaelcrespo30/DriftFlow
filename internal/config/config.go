package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config es la configuración centralizada de la aplicación
type Config struct {
	// GRPC
	GRPCAddr string

	// Base de datos
	DBType     string
	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string
	DBAdminUser     string
	DBAdminPassword string	
	
	DBSSLMode  string

	// Autenticación
	JWTSecret           string
	JWTIssuer           string
	SecretKey           string
	EncryptionSecretKey string
	AccessTokenExpiry   time.Duration
	RefreshTokenExpiry  time.Duration

	// Redis
	RedisAddr     string
	RedisPassword string
}

// AppConfig contiene la configuración cargada globalmente
var AppConfig *Config

// LoadConfig carga las variables de entorno desde el sistema o archivo .env
func LoadConfig() *Config {
	_ = godotenv.Load() // Ignora si no hay archivo .env

	// GRPC
	grpcAddr := os.Getenv("GRPC_ADDR")
	if grpcAddr == "" {
		grpcAddr = ":50051"
		log.Printf("GRPC_ADDR no definido, usando valor por defecto: %s", grpcAddr)
	}

	// Tokens
	accessTokenExpiry := parseDurationWithDefault("ACCESS_TOKEN_EXPIRY", 15*time.Minute)
	refreshTokenExpiry := parseDurationWithDefault("REFRESH_TOKEN_EXPIRY", 7*24*time.Hour)

	AppConfig = &Config{
		GRPCAddr: grpcAddr,

		// Base de datos
		DBType:     getEnvOrDefault("DB_TYPE", "postgres"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBSSLMode:  os.Getenv("DB_SSLMODE"),

		// Autenticación
		JWTSecret:           os.Getenv("JWT_SECRET"),
		JWTIssuer:           os.Getenv("JWT_ISSUER"),
		SecretKey:           os.Getenv("SECRET_KEY"),
		EncryptionSecretKey: os.Getenv("ENCRYPTION_SECRET_KEY"),
		AccessTokenExpiry:   accessTokenExpiry,
		RefreshTokenExpiry:  refreshTokenExpiry,

		// Redis
		RedisAddr:     os.Getenv("REDIS_ADDR"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
	}

	validateConfig(AppConfig)

	return AppConfig
}

// parseDurationWithDefault intenta leer una duración o devuelve un valor por defecto
func parseDurationWithDefault(envVar string, def time.Duration) time.Duration {
	val := os.Getenv(envVar)
	if val == "" {
		return def
	}
	dur, err := time.ParseDuration(val)
	if err != nil {
		log.Printf("Advertencia: %s no válido ('%s'), usando valor por defecto", envVar, val)
		return def
	}
	return dur
}

// getEnvOrDefault retorna el valor de entorno o el valor por defecto si está vacío
func getEnvOrDefault(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

// validateConfig verifica que las variables críticas estén presentes
func validateConfig(cfg *Config) {
	required := map[string]string{
		"DB_USER":     cfg.DBUser,
		"DB_PASSWORD": cfg.DBPassword,
		"DB_NAME":     cfg.DBName,
		"DB_HOST":     cfg.DBHost,
		"DB_PORT":     cfg.DBPort,
		"JWT_SECRET":  cfg.JWTSecret,
		"JWT_ISSUER":  cfg.JWTIssuer,
		"SECRET_KEY":  cfg.SecretKey,
	}

	for key, value := range required {
		if value == "" {
			log.Fatalf("Error: Falta la variable de entorno obligatoria: %s", key)
		}
	}
}
