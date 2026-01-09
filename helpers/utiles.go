package helpers

import (
	"bytes"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/crypto/pbkdf2"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dromara/carbon/v2"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// funcion para validar relaciones
func ValidateRelations(db *gorm.DB, relations map[string]struct {
	Model interface{}
	ID    uint
}) error {
	for name, relation := range relations {
		if err := db.First(relation.Model, relation.ID).Error; err != nil {
			return fmt.Errorf("%s with ID %d does not exist", name, relation.ID)
		}
	}
	return nil
}

//Validar fechas  inicio y final

func ValidateAndParseDates(start, end string) (time.Time, time.Time, error) {
	startDate, err := time.Parse("2006-01-02", start)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid start date format: %v", err)
	}

	endDate, err := time.Parse("2006-01-02", end)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid end date format: %v", err)
	}

	if endDate.Before(startDate) {
		return time.Time{}, time.Time{}, errors.New("end date cannot be earlier than start date")
	}

	return startDate.UTC(), endDate.UTC(), nil
}

/* Funciones random ---------------------------------------------------------------------------------------------*/

// Crear un generador de números aleatorios personalizado
var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

// RandomBool devuelve un valor aleatorio true o false
func RandomBool() bool {
	return rnd.Intn(2) == 0
}

// GetRandomRecord obtiene un registro aleatorio de una tabla específica
func GetRandomRecord[T any](db *gorm.DB, model *T) *T {
	var count int64
	db.Model(model).Count(&count)

	if count == 0 {
		return nil
	}

	// Usar el generador personalizado para obtener un índice aleatorio
	randomIndex := rnd.Intn(int(count))
	var record T
	db.Model(model).Offset(randomIndex).First(&record)
	return &record
}

// RandomLabel devuelve una etiqueta aleatoria de una lista de etiquetas
func GetRandomLabel(labels []string) string {
	return labels[rnd.Intn(len(labels))]
}

func RandomInt(min, max int) int {
	return rnd.Intn(max-min+1) + min
}

/*-----------------------------------------------------------------------------------*/
/* Utiles de fechas      */

// CustomDate usa un campo interno time.Time para la fecha
type CustomDate struct {
	Time time.Time
}

// UnmarshalJSON decodifica el JSON usando Carbon para manejar la fecha en formato "yyyy-MM-dd"
func (c *CustomDate) UnmarshalJSON(data []byte) error {
	var dateStr string
	if err := json.Unmarshal(data, &dateStr); err != nil {
		return err
	}

	parsedDate := carbon.Parse(dateStr, carbon.UTC)
	if parsedDate.Error != nil {
		return parsedDate.Error
	}

	c.Time = parsedDate.StdTime()
	return nil
}

// MarshalJSON convierte el valor time.Time a formato JSON "yyyy-MM-dd"
func (c CustomDate) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Time.Format("2006-01-02"))
}

// ToTime devuelve el valor time.Time almacenado
func (c CustomDate) ToTime() time.Time {
	return c.Time
}

// Value convierte CustomDate a driver.Value para ser almacenado en la base de datos
func (c CustomDate) Value() (driver.Value, error) {
	return c.Time.Format("2006-01-02"), nil
}

// Scan convierte un valor de la base de datos a CustomDate
func (c *CustomDate) Scan(value interface{}) error {
	if date, ok := value.(time.Time); ok {
		c.Time = date
		return nil
	}
	return fmt.Errorf("cannot scan type %T into CustomDate", value)
}

/*----------------------------------------------------------------*/

// ReadJSON lee un archivo JSON y lo deserializa en una estructura de tipo genérico
func ReadJSON[T any](filePath string, result *T) error {
	// Abrir el archivo JSON
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo JSON: %w", err)
	}
	defer file.Close()

	// Decodificar el JSON
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(result); err != nil {
		return fmt.Errorf("error al decodificar el JSON: %w", err)
	}

	return nil
}

/*------------------------------------------------------------------------------*/

// RespondWithError envía una respuesta de error estandarizada
func RespondWithError(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, gin.H{
		"success": false,
		"error":   message,
	})
}

// RespondWithValidationError envía un error con detalles de validación
func RespondWithValidationError(c *gin.Context, errors map[string]string) {
	c.JSON(http.StatusBadRequest, gin.H{
		"success": false,
		"error":   "Validación fallida",
		"details": errors,
	})
}

// RespondWithUnauthorizedError envía un error de autenticación
func RespondWithUnauthorizedError(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, gin.H{
		"success": false,
		"error":   message,
	})
}

/*-------------------------------------------------------------------------------------------------------------*/
// CreateAndPreload crea un registro y precarga las relaciones indicadas.
func CreateAndPreload(tx *gorm.DB, model interface{}, preloadFields []string) error {
	// Crear el registro
	if err := tx.Create(model).Error; err != nil {
		return fmt.Errorf("failed to create record: %w", err)
	}

	// Precargar relaciones
	query := tx.Model(model)
	for _, field := range preloadFields {
		query = query.Preload(field)
	}

	// Cargar el modelo con las relaciones
	if err := query.First(model).Error; err != nil {
		return fmt.Errorf("failed to preload relations: %w", err)
	}

	return nil
}

/*-------------------------------------------------------------------------------------------------------------*/

// Tratamiendo con ficheros

// GenerateUniqueFileName genera un nombre de archivo único basado en la hora actual y un sufijo aleatorio.
func GenerateUniqueFileName(originalName string) string {
	ext := filepath.Ext(originalName)
	name := filepath.Base(originalName[:len(originalName)-len(ext)])
	return fmt.Sprintf("%s_%d%s", name, time.Now().UnixNano()+rand.Int63(), ext)
}

func ApplySearchAndFilters(query *gorm.DB, search map[string]interface{}, fields []string) (*gorm.DB, error) {
	var globalSearch string
	var filters []map[string]string

	// Extraer búsqueda global y filtros
	if global, ok := search["global"]; ok {
		// Verificar si es un string directamente
		if str, isString := global.(string); isString {
			globalSearch = str
		} else if globalMap, isMap := global.(map[string]interface{}); isMap {
			if value, hasValue := globalMap["value"].(string); hasValue {
				globalSearch = value
			}
		} else {
			return nil, fmt.Errorf("invalid format for global search: %v", global)
		}
	}

	if filtersArray, ok := search["filters"].([]map[string]string); ok {
		filters = filtersArray
	}

	// Construir condiciones para búsqueda global
	if globalSearch != "" {
		orConditions := make([]string, 0)
		values := make([]interface{}, 0)

		for _, field := range fields {
			orConditions = append(orConditions, fmt.Sprintf("%s ILIKE ?", field))
			values = append(values, "%"+globalSearch+"%")
		}

		// Aplicar las condiciones de búsqueda global con OR
		query = query.Where("("+strings.Join(orConditions, " OR ")+")", values...)
	}

	// Aplicar filtros específicos con AND
	for _, filter := range filters {
		field := filter["field"]
		filterType := filter["type"]
		value := filter["value"]

		// Omitir filtros con valores vacíos
		if value == "" {
			continue // No aplica ningún filtro, pasa al siguiente
		}

		switch filterType {
		case "number":
			query = query.Where(fmt.Sprintf("%s = ?", field), value)
		case "date":
			// Manejar operadores como menor, mayor, entre fechas
			if strings.HasPrefix(value, "<") {
				dateValue := strings.TrimPrefix(value, "<")
				query = query.Where(fmt.Sprintf("%s < ?", field), dateValue)
			} else if strings.HasPrefix(value, ">") {
				dateValue := strings.TrimPrefix(value, ">")
				query = query.Where(fmt.Sprintf("%s > ?", field), dateValue)
			} else if strings.Contains(value, ",") {
				dates := strings.Split(value, ",")
				if len(dates) == 2 {
					query = query.Where(fmt.Sprintf("%s BETWEEN ? AND ?", field), dates[0], dates[1])
				}
			} else {
				query = query.Where(fmt.Sprintf("%s = ?", field), value)
			}
		case "text":
			query = query.Where(fmt.Sprintf("%s ILIKE ?", field), "%"+value+"%")
		default:
			log.Printf("Unsupported filter type: %s", filterType)
			continue // Salta este filtro y sigue con el siguiente
		}
	}

	return query, nil
}

// ConvertFilters convierte un arreglo de filtros genéricos (interface{}) a un slice de mapas de strings
func ConvertFilters(rawFilters []interface{}) ([]map[string]string, error) {
	filters := make([]map[string]string, 0)

	for _, filter := range rawFilters {
		// Verificar si el filtro es un mapa
		if filterMap, ok := filter.(map[string]interface{}); ok {
			convertedFilter := map[string]string{}
			// Convertir cada campo a string
			for key, value := range filterMap {
				if strValue, ok := value.(string); ok {
					convertedFilter[key] = strValue
				} else {
					return nil, fmt.Errorf("filter value for key '%s' is not a string", key)
				}
			}
			filters = append(filters, convertedFilter)
		} else {
			return nil, fmt.Errorf("filter is not a valid map[string]interface{}")
		}
	}

	return filters, nil
}

func HashPasswordLikeIdentityV3(password string) (string, error) {
	const (
		saltSize   = 16
		hashSize   = 32
		iterations = 10000
		version    = 0x01 // Identity v3
	)

	// Salt aleatorio
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// Derivar clave con PBKDF2-HMAC-SHA256
	derived := pbkdf2.Key([]byte(password), salt, iterations, hashSize, sha256.New)

	// Ensamblar hash final: [version|salt|hash]
	full := make([]byte, 1+saltSize+hashSize)
	full[0] = version
	copy(full[1:1+saltSize], salt)
	copy(full[1+saltSize:], derived)

	// Codificar a base64
	return base64.StdEncoding.EncodeToString(full), nil
}

func VerifyIdentityV3Hash(password string, encodedHash string) (bool, error) {
	const (
		version       = 0x01
		saltSize      = 16
		hashSize      = 32
		iterations    = 10000
		expectedTotal = 1 + saltSize + hashSize
	)

	// Decodifica el hash base64
	hashBytes, err := base64.StdEncoding.DecodeString(encodedHash)
	if err != nil {
		return false, fmt.Errorf("invalid base64 hash: %w", err)
	}

	// Verifica longitud mínima y versión
	if len(hashBytes) != expectedTotal || hashBytes[0] != version {
		return false, errors.New("unsupported hash format or version")
	}

	// Extrae salt y hash
	salt := hashBytes[1 : 1+saltSize]
	storedHash := hashBytes[1+saltSize:]

	// Calcula hash de la contraseña dada usando mismo salt y parámetros
	calculated := pbkdf2.Key([]byte(password), salt, iterations, hashSize, sha256.New)

	// Comparación constante
	if !bytes.Equal(calculated, storedHash) {
		return false, nil // Contraseña incorrecta
	}

	return true, nil // Contraseña válida
}

func BuildFullName(firstName, lastName string) string {
	if firstName == "" && lastName == "" {
		return ""
	}
	if firstName == "" {
		return lastName
	}
	if lastName == "" {
		return firstName
	}
	return firstName + " " + lastName
}
