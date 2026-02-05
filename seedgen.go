package driftflow

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/misaelcrespo30/DriftFlow/config"
)

const seedTemplateCount = 10

type seedModelInfo struct {
	modelType     reflect.Type
	primaryJSON   string
	primaryType   reflect.Type
	primaryIsText bool
}

func dummyValue(t reflect.Type, idx int, base time.Time) interface{} {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.PkgPath() == "time" && t.Name() == "Time" {
		return base.Add(time.Duration(idx) * time.Hour)
	}
	switch t.Kind() {
	case reflect.Bool:
		return idx%2 == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return idx + 1
	case reflect.Float32, reflect.Float64:
		return float64(idx + 1)
	case reflect.String:
		return fmt.Sprintf("value %d", idx+1)
	default:
		return reflect.Zero(t).Interface()
	}
}

func dummyValueForField(name string, t reflect.Type, idx int, base time.Time) interface{} {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	name = strings.ToLower(name)

	if value, ok := ruleValueForField(name, t, idx); ok {
		return value
	}

	if isDBTypeField(name) {
		return dbTypeForIndex(idx)
	}

	if name == "connection_string" {
		return connectionStringForType(dbTypeForIndex(idx), idx)
	}

	if isServicePlanField(name) {
		return servicePlanForIndex(idx)
	}

	if isVersionField(name) {
		return versionForIndex(idx)
	}

	if name == "email" {
		return fmt.Sprintf("user%d@example.com", idx+1)
	}

	if name == "username" || name == "user_name" {
		return fmt.Sprintf("user%d", idx+1)
	}

	if isFirstNameField(name) {
		return firstNameForIndex(idx)
	}

	if isLastNameField(name) {
		return lastNameForIndex(idx)
	}

	if isFullNameField(name) {
		return fmt.Sprintf("%s %s", firstNameForIndex(idx), lastNameForIndex(idx))
	}

	if isGenericNameField(name) {
		label := nameLabel(name)
		if label != "" {
			return fmt.Sprintf("%s %d", label, idx+1)
		}
	}

	if isAddressField(name) {
		return fmt.Sprintf("%d Main St", 100+idx)
	}

	if isCityField(name) {
		return fmt.Sprintf("Ciudad %d", idx+1)
	}

	if isStateField(name) {
		return fmt.Sprintf("Estado %d", idx+1)
	}

	if isPostalCodeField(name) {
		return fmt.Sprintf("%05d", 10000+idx)
	}

	if isCountryField(name) {
		return "País"
	}

	if name == "phone" || name == "phone_number" || name == "phonenumber" {
		return fmt.Sprintf("+15551234%04d", idx+1)
	}

	if name == "security_stamp" || name == "securitystamp" {
		return uuid.NewString()
	}

	if strings.HasSuffix(name, "_id") || name == "id" {
		if t.Kind() == reflect.String {
			return uuid.NewString()
		}
		switch t.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return int64(idx + 1)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return uint64(idx + 1)
		}
	}

	return dummyValue(t, idx, base)
}

func uniqueValueForField(name string, t reflect.Type, idx int, base time.Time) interface{} {
	value := dummyValueForField(name, t, idx, base)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() == reflect.String {
		if isKeyField(name) {
			return value
		}
		strValue, ok := value.(string)
		if !ok {
			return value
		}
		suffix := strconv.Itoa(idx + 1)
		if strings.Contains(strValue, suffix) {
			return strValue
		}
		return fmt.Sprintf("%s %s", strValue, suffix)
	}
	return value
}

func hasPasswordHashField(t reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		if f.Name == "PasswordHash" {
			return true
		}
	}
	return false
}

func isFirstNameField(name string) bool {
	return name == "first_name" || name == "firstname" || name == "given_name"
}

func isLastNameField(name string) bool {
	return name == "last_name" || name == "lastname" || name == "surname" || name == "family_name"
}

func isFullNameField(name string) bool {
	return name == "full_name" || name == "fullname" || name == "name"
}

func isGenericNameField(name string) bool {
	if name == "username" || name == "user_name" {
		return false
	}
	return strings.HasSuffix(name, "_name")
}

func nameLabel(name string) string {
	base := strings.TrimSuffix(name, "_name")
	base = strings.ReplaceAll(base, "_", " ")
	if base == "" {
		return ""
	}
	return titleWords(base)
}

func isAddressField(name string) bool {
	switch name {
	case "address", "address_line", "address_line1", "address_line2", "street", "street_address", "direccion":
		return true
	default:
		return strings.HasSuffix(name, "_address")
	}
}

func isCityField(name string) bool {
	return name == "city" || name == "town"
}

func isStateField(name string) bool {
	return name == "state" || name == "province" || name == "region"
}

func isPostalCodeField(name string) bool {
	return name == "zip" || name == "zipcode" || name == "postal_code" || name == "postcode"
}

func isCountryField(name string) bool {
	return name == "country" || name == "country_code"
}

func titleWords(value string) string {
	parts := strings.Fields(value)
	for i, part := range parts {
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func firstNameForIndex(idx int) string {
	firstNames := []string{"Ana", "Luis", "Marta", "Carlos", "Sofía", "Diego", "Valeria", "Jorge", "Lucía", "Pedro"}
	return firstNames[idx%len(firstNames)]
}

func lastNameForIndex(idx int) string {
	lastNames := []string{"García", "Pérez", "Rodríguez", "López", "Martínez", "Gómez", "Hernández", "Díaz", "Moreno", "Vargas"}
	return lastNames[idx%len(lastNames)]
}

func ruleValueForField(name string, t reflect.Type, idx int) (interface{}, bool) {
	normalized := normalizedFieldName(name)

	if normalized == "defaultclientid" {
		return "web", true
	}

	if normalized == "app" {
		return appValueForIndex(idx), true
	}

	if isJSONType(t) {
		if normalized == "allowedredirecturis" {
			return jsonValue(allowedRedirectURIs()), true
		}
		if normalized == "allowedscopes" {
			return jsonValue(allowedScopes()), true
		}
		if normalized == "allowedrootdomains" {
			return jsonValue(allowedRootDomains()), true
		}
		if normalized == "allowedredirectsubs" || normalized == "allowedredirectsubdomains" {
			return jsonValue(allowedRedirectSubs()), true
		}
		if strings.Contains(normalized, "domain") {
			return jsonValue(allowedRootDomains()), true
		}
	}

	if strings.Contains(normalized, "domain") && t.Kind() == reflect.String {
		return domainForIndex(idx), true
	}

	if isKeyField(name) {
		return keyValueForField(name, idx), true
	}

	return nil, false
}

func normalizedFieldName(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), "_", "")
}

func jsonValue(value interface{}) json.RawMessage {
	b, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage("null")
	}
	return json.RawMessage(b)
}

func domainForIndex(idx int) string {
	domains := allowedRootDomains()
	return domains[idx%len(domains)]
}

func allowedRootDomains() []string {
	return []string{"elevitae.com", "mydailychoice.com"}
}

func allowedRedirectURIs() []string {
	return []string{
		"https://web.elevitae.com/auth/callback",
		"https://web.mydailychoice.com/auth/callback",
	}
}

func allowedScopes() []string {
	return []string{"openid", "email", "profile"}
}

func allowedRedirectSubs() []string {
	return []string{"web", "backoffices", "login"}
}

func appValueForIndex(idx int) string {
	apps := allowedRedirectSubs()
	return apps[idx%len(apps)]
}

func isKeyField(name string) bool {
	name = strings.ToLower(name)
	return strings.Contains(name, "key")
}

func keyValueForField(name string, idx int) string {
	name = strings.ToLower(name)
	if name == "service_key" {
		return keyValueFromCatalog([]string{"branding", "network", "provisioning", "subscription"}, idx)
	}
	if name == "plan_key" {
		return keyValueFromCatalog([]string{"basic", "pro", "enterprise"}, idx)
	}
	base := strings.TrimSuffix(name, "_key")
	if base == name {
		base = "key"
	}
	base = slugify(base)
	return fmt.Sprintf("%s-%d", base, idx+1)
}

func keyValueFromCatalog(values []string, idx int) string {
	if len(values) == 0 {
		return fmt.Sprintf("key-%d", idx+1)
	}
	value := values[idx%len(values)]
	if idx >= len(values) {
		return fmt.Sprintf("%s-%d", value, idx+1)
	}
	return value
}

func slugify(value string) string {
	value = strings.ToLower(value)
	var buf strings.Builder
	lastDash := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			buf.WriteRune(r)
			lastDash = false
			continue
		}
		if lastDash {
			continue
		}
		buf.WriteByte('-')
		lastDash = true
	}
	result := strings.Trim(buf.String(), "-")
	if result == "" {
		return "key"
	}
	return result
}

func isJSONType(t reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 && t.Name() == "JSON"
}

// GenerateSeedTemplates writes JSON seed files with dummy data for the provided
// models into dir. Each file contains an array of 10 objects and will be
// overwritten if it already exists.
func GenerateSeedTemplates(models []interface{}, dir string) error {
	return GenerateSeedTemplatesWithData(models, dir, nil)
}

// GenerateSeedAssets writes JSON seed templates and Go seeder scaffolding for
// the provided models into internal/database/{data,seed} (or a custom dir).
func GenerateSeedAssets(models []interface{}, dir string) error {
	dataDir, seedDir, err := resolveSeedGenDirs(dir)
	if err != nil {
		return err
	}
	if err := generateSeedTemplates(models, dataDir, nil); err != nil {
		return err
	}
	if err := generateSeedScaffold(models, seedDir); err != nil {
		return err
	}
	return generateSeedRegistryHooks()
}

// GenerateSeedTemplatesWithData is like GenerateSeedTemplates but allows providing
// custom generator functions for field values. The map key should match the JSON
// field name. If no generator is found for a field, a zero value is used.
func GenerateSeedTemplatesWithData(models []interface{}, dir string, gens map[string]func() interface{}) error {
	dataDir, _, err := resolveSeedGenDirs(dir)
	if err != nil {
		return err
	}
	return generateSeedTemplates(models, dataDir, gens)
}

func resolveSeedGenDirs(dir string) (string, string, error) {
	if strings.TrimSpace(dir) == "" {
		cfg := config.Load()
		dir = cfg.SeedGenDir
		if strings.TrimSpace(dir) == "" {
			dir = "internal/database/data"
			fmt.Println("No se definió 'SEED_GEN_DIR', se usará ruta por defecto:", dir)
		}
	}
	baseDir := dir
	if filepath.Base(dir) == "data" {
		baseDir = filepath.Dir(dir)
	}
	dataDir := filepath.Join(baseDir, "data")
	seedDir := filepath.Join(baseDir, "seed")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(seedDir, 0o755); err != nil {
		return "", "", err
	}
	return dataDir, seedDir, nil
}

func generateSeedTemplates(models []interface{}, dir string, gens map[string]func() interface{}) error {
	modelInfos := buildSeedModelInfo(models)
	infoByType := make(map[reflect.Type]seedModelInfo, len(modelInfos))
	primaryIDs := make(map[reflect.Type][]string)
	primaryByJSON := make(map[string]seedModelInfo)
	uniqueFields := buildSeedUniqueFields(models)
	for _, info := range modelInfos {
		infoByType[info.modelType] = info
		if info.primaryJSON != "" && info.primaryIsText {
			ids := make([]string, seedTemplateCount)
			for i := 0; i < seedTemplateCount; i++ {
				ids[i] = uuid.NewString()
			}
			primaryIDs[info.modelType] = ids
			primaryByJSON[info.primaryJSON] = info
		}
	}

	base := time.Now()
	for _, m := range models {
		t := reflect.TypeOf(m)
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			continue
		}
		file := strings.ToLower(t.Name()) + ".seed.json"
		path := filepath.Join(dir, file)

		objs := make([]*orderedMap, seedTemplateCount)
		hasPassword := hasPasswordHashField(t)
		hasDBType := modelHasFieldName(t, "dbtype", "db_type", "database_type", "engine", "db_engine")
		naturalKeyFields := naturalKeyFieldsForModel(t)
		for i := 0; i < seedTemplateCount; i++ {
			obj := newOrderedMap()
			dbType := ""
			if hasDBType {
				dbType = dbTypeForIndex(i)
			}
			for j := 0; j < t.NumField(); j++ {
				f := t.Field(j)
				if !f.IsExported() {
					continue
				}
				if f.Anonymous && f.Type.PkgPath() == "gorm.io/gorm" && f.Type.Name() == "Model" {
					continue
				}
				gtag := f.Tag.Get("gorm")
				if gtag == "-" || strings.Contains(gtag, "->") {
					continue
				}
				tag := f.Tag.Get("json")
				if strings.Split(tag, ",")[0] == "-" {
					continue
				}
				name := strings.Split(tag, ",")[0]
				if name == "" {
					name = strings.ToLower(f.Name)
				}
				if gens != nil {
					if fn, ok := gens[name]; ok {
						obj.set(name, fn())
						continue
					}
				}
				if id, ok := primaryIDValue(primaryIDs, infoByType, t, name, i); ok {
					obj.set(name, id)
					continue
				}
				if id, ok := foreignIDValue(primaryIDs, primaryByJSON, t, name, i); ok {
					obj.set(name, id)
					continue
				}
				if isDBTypeField(name) && dbType != "" {
					obj.set(name, dbType)
					continue
				}
				if name == "connection_string" && dbType != "" {
					obj.set(name, connectionStringForType(dbType, i))
					continue
				}
				if uniqueFields != nil && uniqueFields[t][name] {
					obj.set(name, uniqueValueForField(name, f.Type, i, base))
					continue
				}
				obj.set(name, dummyValueForField(name, f.Type, i, base))
			}
			if hasPassword {
				obj.set("password_hash", fmt.Sprintf("password%d", i+1))
			}
			objs[i] = obj
		}
		objs = dedupeSeedObjects(objs, naturalKeyFields)

		b, err := json.MarshalIndent(objs, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func generateSeedScaffold(models []interface{}, seedDir string) error {
	seeders := make([]string, 0, len(models))
	written := map[string]struct{}{}

	for _, m := range models {
		t := reflect.TypeOf(m)
		if t == nil {
			continue
		}
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			continue
		}
		pkgPath := t.PkgPath()
		if pkgPath == "" {
			continue
		}

		seederName := t.Name() + "Seeder"
		seeders = append(seeders, seederName)

		fileName := strings.ToLower(t.Name()) + "_seeder.go"
		if _, ok := written[fileName]; ok {
			continue
		}
		written[fileName] = struct{}{}

		connField, connPointer := connectionStringField(t)
		source := buildSeederSource(pkgPath, t.Name(), seederName, hasPasswordHashField(t), connField, connPointer)
		formatted, err := format.Source(source)
		if err != nil {
			return err
		}

		if err := os.WriteFile(filepath.Join(seedDir, fileName), formatted, 0o644); err != nil {
			return err
		}
	}

	registerSource := buildRegisterSource(seeders)
	formatted, err := format.Source(registerSource)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(seedDir, "register.go"), formatted, 0o644)
}

func buildSeederSource(pkgPath, modelName, seederName string, hashPasswords bool, connectionField string, connectionPointer bool) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "package seed\n\n")
	fmt.Fprintf(&buf, "import (\n")
	fmt.Fprintf(&buf, "\t\"encoding/json\"\n")
	if hashPasswords {
		fmt.Fprintf(&buf, "\t\"fmt\"\n")
	}
	if connectionField != "" {
		fmt.Fprintf(&buf, "\t\"github.com/misaelcrespo30/DriftFlow/helpers\"\n")
	}
	fmt.Fprintf(&buf, "\t\"os\"\n\n")
	fmt.Fprintf(&buf, "\tmodels \"%s\"\n", pkgPath)
	if hashPasswords {
		fmt.Fprintf(&buf, "\t\"golang.org/x/crypto/bcrypt\"\n")
	}
	fmt.Fprintf(&buf, "\t\"gorm.io/gorm\"\n")
	fmt.Fprintf(&buf, ")\n\n")
	fmt.Fprintf(&buf, "type %s struct{}\n\n", seederName)
	if hashPasswords {
		fmt.Fprintf(&buf, "type %sSeed struct {\n", strings.ToLower(modelName))
		fmt.Fprintf(&buf, "\tmodels.%s\n", modelName)
		fmt.Fprintf(&buf, "\tPassword string `json:\"password_hash\"`\n")
		fmt.Fprintf(&buf, "}\n\n")
	}
	fmt.Fprintf(&buf, "func (s %s) Seed(db *gorm.DB, filePath string) error {\n", seederName)
	fmt.Fprintf(&buf, "\tdata, err := os.ReadFile(filePath)\n")
	fmt.Fprintf(&buf, "\tif err != nil {\n\t\treturn err\n\t}\n\n")
	if hashPasswords {
		fmt.Fprintf(&buf, "\tvar rows []%sSeed\n", strings.ToLower(modelName))
		fmt.Fprintf(&buf, "\tif err := json.Unmarshal(data, &rows); err != nil {\n\t\treturn err\n\t}\n")
		fmt.Fprintf(&buf, "\tif len(rows) == 0 {\n\t\treturn nil\n\t}\n\n")
		fmt.Fprintf(&buf, "\titems := make([]models.%s, 0, len(rows))\n", modelName)
		fmt.Fprintf(&buf, "\tfor _, row := range rows {\n")
		fmt.Fprintf(&buf, "\t\thashedPassword, err := bcrypt.GenerateFromPassword([]byte(row.Password), bcrypt.DefaultCost)\n")
		fmt.Fprintf(&buf, "\t\tif err != nil {\n")
		fmt.Fprintf(&buf, "\t\t\treturn fmt.Errorf(\"hashing password for %s %%s: %%w\", row.Email, err)\n", strings.ToLower(modelName))
		fmt.Fprintf(&buf, "\t\t}\n")
		fmt.Fprintf(&buf, "\t\trow.%s.PasswordHash = string(hashedPassword)\n", modelName)
		if connectionField != "" {
			if connectionPointer {
				fmt.Fprintf(&buf, "\t\tif row.%s != nil {\n", connectionField)
				fmt.Fprintf(&buf, "\t\t\tencrypted, err := helpers.Encrypt(*row.%s)\n", connectionField)
				fmt.Fprintf(&buf, "\t\t\tif err != nil {\n")
				fmt.Fprintf(&buf, "\t\t\t\treturn err\n")
				fmt.Fprintf(&buf, "\t\t\t}\n")
				fmt.Fprintf(&buf, "\t\t\trow.%s = &encrypted\n", connectionField)
				fmt.Fprintf(&buf, "\t\t}\n")
			} else {
				fmt.Fprintf(&buf, "\t\tif row.%s != \"\" {\n", connectionField)
				fmt.Fprintf(&buf, "\t\t\tencrypted, err := helpers.Encrypt(row.%s)\n", connectionField)
				fmt.Fprintf(&buf, "\t\t\tif err != nil {\n")
				fmt.Fprintf(&buf, "\t\t\t\treturn err\n")
				fmt.Fprintf(&buf, "\t\t\t}\n")
				fmt.Fprintf(&buf, "\t\t\trow.%s = encrypted\n", connectionField)
				fmt.Fprintf(&buf, "\t\t}\n")
			}
		}
		fmt.Fprintf(&buf, "\t\titems = append(items, row.%s)\n", modelName)
		fmt.Fprintf(&buf, "\t}\n\n")
		fmt.Fprintf(&buf, "\treturn db.Create(&items).Error\n")
	} else {
		fmt.Fprintf(&buf, "\tvar rows []models.%s\n", modelName)
		fmt.Fprintf(&buf, "\tif err := json.Unmarshal(data, &rows); err != nil {\n\t\treturn err\n\t}\n")
		fmt.Fprintf(&buf, "\tif len(rows) == 0 {\n\t\treturn nil\n\t}\n\n")
		if connectionField != "" {
			fmt.Fprintf(&buf, "\tfor i := range rows {\n")
			fmt.Fprintf(&buf, "\t\trow := &rows[i]\n")
			if connectionPointer {
				fmt.Fprintf(&buf, "\t\tif row.%s != nil {\n", connectionField)
				fmt.Fprintf(&buf, "\t\t\tencrypted, err := helpers.Encrypt(*row.%s)\n", connectionField)
				fmt.Fprintf(&buf, "\t\t\tif err != nil {\n")
				fmt.Fprintf(&buf, "\t\t\t\treturn err\n")
				fmt.Fprintf(&buf, "\t\t\t}\n")
				fmt.Fprintf(&buf, "\t\t\trow.%s = &encrypted\n", connectionField)
				fmt.Fprintf(&buf, "\t\t}\n")
			} else {
				fmt.Fprintf(&buf, "\t\tif row.%s != \"\" {\n", connectionField)
				fmt.Fprintf(&buf, "\t\t\tencrypted, err := helpers.Encrypt(row.%s)\n", connectionField)
				fmt.Fprintf(&buf, "\t\t\tif err != nil {\n")
				fmt.Fprintf(&buf, "\t\t\t\treturn err\n")
				fmt.Fprintf(&buf, "\t\t\t}\n")
				fmt.Fprintf(&buf, "\t\t\trow.%s = encrypted\n", connectionField)
				fmt.Fprintf(&buf, "\t\t}\n")
			}
			fmt.Fprintf(&buf, "\t}\n\n")
		}
		fmt.Fprintf(&buf, "\treturn db.Create(&rows).Error\n")
	}
	fmt.Fprintf(&buf, "}\n")
	return buf.Bytes()
}

func buildRegisterSource(seeders []string) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "package seed\n\n")
	fmt.Fprintf(&buf, "import driftflow \"github.com/misaelcrespo30/DriftFlow\"\n\n")
	fmt.Fprintf(&buf, "func RegisterSeeders() []driftflow.Seeder {\n")
	fmt.Fprintf(&buf, "\treturn []driftflow.Seeder{\n")
	for _, seeder := range seeders {
		fmt.Fprintf(&buf, "\t\t%s{},\n", seeder)
	}
	fmt.Fprintf(&buf, "\t}\n")
	fmt.Fprintf(&buf, "}\n")
	fmt.Fprintf(&buf, "\n")
	fmt.Fprintf(&buf, "func init() {\n")
	fmt.Fprintf(&buf, "\tdriftflow.SetSeederRegistry(RegisterSeeders)\n")
	fmt.Fprintf(&buf, "}\n")
	return buf.Bytes()
}

func buildSeedModelInfo(models []interface{}) []seedModelInfo {
	infos := make([]seedModelInfo, 0, len(models))
	for _, m := range models {
		t := reflect.TypeOf(m)
		if t == nil {
			continue
		}
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			continue
		}
		info := seedModelInfo{modelType: t}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			if f.Anonymous && f.Type.PkgPath() == "gorm.io/gorm" && f.Type.Name() == "Model" {
				continue
			}
			gtag := f.Tag.Get("gorm")
			if gtag == "-" || strings.Contains(gtag, "->") {
				continue
			}
			if !strings.Contains(gtag, "primaryKey") {
				continue
			}
			tag := f.Tag.Get("json")
			if strings.Split(tag, ",")[0] == "-" {
				continue
			}
			name := strings.Split(tag, ",")[0]
			if name == "" {
				name = strings.ToLower(f.Name)
			}
			info.primaryJSON = name
			info.primaryType = f.Type
			info.primaryIsText = isTextType(f.Type)
			break
		}
		infos = append(infos, info)
	}
	return infos
}

func naturalKeyFieldsForModel(t reflect.Type) []string {
	var keyField string
	var emailField string
	var tenantField string
	var appField string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		if f.Anonymous && f.Type.PkgPath() == "gorm.io/gorm" && f.Type.Name() == "Model" {
			continue
		}
		gtag := f.Tag.Get("gorm")
		if gtag == "-" || strings.Contains(gtag, "->") {
			continue
		}
		tag := f.Tag.Get("json")
		if strings.Split(tag, ",")[0] == "-" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name == "" {
			name = strings.ToLower(f.Name)
		}
		lowerName := strings.ToLower(name)
		if keyField == "" && strings.HasSuffix(lowerName, "_key") {
			keyField = name
		}
		if emailField == "" && lowerName == "email" {
			emailField = name
		}
		if tenantField == "" && normalizedFieldName(lowerName) == "tenantid" {
			tenantField = name
		}
		if appField == "" && normalizedFieldName(lowerName) == "app" {
			appField = name
		}
	}
	if keyField != "" {
		return []string{keyField}
	}
	if emailField != "" {
		return []string{emailField}
	}
	if tenantField != "" && appField != "" {
		return []string{tenantField, appField}
	}
	return nil
}

func dedupeSeedObjects(objs []*orderedMap, naturalKeyFields []string) []*orderedMap {
	if len(naturalKeyFields) == 0 {
		return objs
	}
	seen := make(map[string]struct{}, len(objs))
	filtered := make([]*orderedMap, 0, len(objs))
	for _, obj := range objs {
		key, ok := naturalKeyForObject(obj, naturalKeyFields)
		if !ok {
			filtered = append(filtered, obj)
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		filtered = append(filtered, obj)
	}
	return filtered
}

func naturalKeyForObject(obj *orderedMap, fields []string) (string, bool) {
	if obj == nil {
		return "", false
	}
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		value, ok := obj.values[field]
		if !ok || value == nil {
			return "", false
		}
		parts = append(parts, fmt.Sprint(value))
	}
	return strings.Join(parts, "|"), true
}

func isDBTypeField(name string) bool {
	switch name {
	case "dbtype", "db_type", "database_type", "engine", "db_engine":
		return true
	default:
		return false
	}
}

func dbTypeForIndex(idx int) string {
	types := []string{"postgres", "mssql", "mysql", "mongodb"}
	return types[idx%len(types)]
}

func connectionStringForType(dbType string, idx int) string {
	switch strings.ToLower(dbType) {
	case "mssql", "sqlserver":
		return fmt.Sprintf("sqlserver://user%d:pass@localhost:1433?database=db%d", idx+1, idx+1)
	case "mysql":
		return fmt.Sprintf("mysql://user%d:pass@tcp(localhost:3306)/db%d", idx+1, idx+1)
	case "mongodb":
		return fmt.Sprintf("mongodb://user%d:pass@localhost:27017/db%d", idx+1, idx+1)
	default:
		return fmt.Sprintf("postgres://user%d:pass@localhost:5432/db%d", idx+1, idx+1)
	}
}

func isServicePlanField(name string) bool {
	return name == "service_plan" || name == "serviceplan"
}

func servicePlanForIndex(idx int) string {
	plans := []string{"standard", "pro", "enterprise"}
	return plans[idx%len(plans)]
}

func isVersionField(name string) bool {
	return name == "version"
}

func versionForIndex(idx int) string {
	versions := []string{"v1", "v2", "v3"}
	return versions[idx%len(versions)]
}

func modelHasFieldName(t reflect.Type, names ...string) bool {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		if f.Anonymous && f.Type.PkgPath() == "gorm.io/gorm" && f.Type.Name() == "Model" {
			continue
		}
		gtag := f.Tag.Get("gorm")
		if gtag == "-" || strings.Contains(gtag, "->") {
			continue
		}
		tag := f.Tag.Get("json")
		if strings.Split(tag, ",")[0] == "-" {
			continue
		}
		jsonName := strings.Split(tag, ",")[0]
		if jsonName == "" {
			jsonName = strings.ToLower(f.Name)
		}
		for _, name := range names {
			if jsonName == name {
				return true
			}
		}
	}
	return false
}

func connectionStringField(t reflect.Type) (string, bool) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		if f.Anonymous && f.Type.PkgPath() == "gorm.io/gorm" && f.Type.Name() == "Model" {
			continue
		}
		gtag := f.Tag.Get("gorm")
		if gtag == "-" || strings.Contains(gtag, "->") {
			continue
		}
		tag := f.Tag.Get("json")
		if strings.Split(tag, ",")[0] == "-" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name == "" {
			name = strings.ToLower(f.Name)
		}
		if name != "connection_string" {
			continue
		}
		ft := f.Type
		if ft.Kind() == reflect.Pointer {
			if ft.Elem().Kind() == reflect.String {
				return f.Name, true
			}
			continue
		}
		if ft.Kind() == reflect.String {
			return f.Name, false
		}
	}
	return "", false
}

func buildSeedUniqueFields(models []interface{}) map[reflect.Type]map[string]bool {
	uniqueFields := make(map[reflect.Type]map[string]bool)

	for _, m := range models {
		t := reflect.TypeOf(m)
		if t == nil {
			continue
		}
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			continue
		}

		// 1) Contar cuántos campos participan en cada índice único con nombre
		uniqueIndexFieldCount := map[string]int{}

		// Guardamos info temporal por campo para el segundo pase
		type fieldInfo struct {
			jsonName string
			tags     []indexTag
		}
		fields := make([]fieldInfo, 0, t.NumField())

		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			if f.Anonymous && f.Type.PkgPath() == "gorm.io/gorm" && f.Type.Name() == "Model" {
				continue
			}
			gtag := f.Tag.Get("gorm")
			if gtag == "-" || strings.Contains(gtag, "->") {
				continue
			}

			tag := f.Tag.Get("json")
			if strings.Split(tag, ",")[0] == "-" {
				continue
			}
			jsonName := strings.Split(tag, ",")[0]
			if jsonName == "" {
				jsonName = strings.ToLower(f.Name)
			}

			tags := parseIndexTags(gtag)
			fields = append(fields, fieldInfo{jsonName: jsonName, tags: tags})

			for _, it := range tags {
				if it.Unique && it.Name != "" {
					uniqueIndexFieldCount[it.Name]++
				}
			}
		}

		// 2) Marcar como unique field solo los realmente “single-column unique”
		for _, fi := range fields {
			mark := false

			for _, it := range fi.tags {
				if !it.Unique {
					continue
				}

				// Caso A: constraint "unique" (sin nombre) = columna única real
				if it.Kind == indexKindUniqueConstraint {
					mark = true
					break
				}

				// Caso B: índice único con nombre pero solo si es de 1 campo
				if it.Name != "" && uniqueIndexFieldCount[it.Name] == 1 {
					mark = true
					break
				}
			}

			if !mark {
				continue
			}
			if uniqueFields[t] == nil {
				uniqueFields[t] = make(map[string]bool)
			}
			uniqueFields[t][fi.jsonName] = true
		}
	}

	return uniqueFields
}

func generateSeedRegistryHooks() error {
	modulePath, err := readModulePath("go.mod")
	if err != nil {
		return err
	}
	if modulePath == "" {
		return nil
	}
	seedImport := modulePath + "/internal/database/seed"

	hookTargets := []string{
		filepath.Join("cmd", "driftflow-demo"),
		filepath.Join("cmd", "driftflow"),
	}

	for _, dir := range hookTargets {
		mainPath := filepath.Join(dir, "main.go")
		if _, err := os.Stat(mainPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		content := fmt.Sprintf("package main\n\nimport _ %q\n", seedImport)
		if err := os.WriteFile(filepath.Join(dir, "seed_registry.go"), []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func readModulePath(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", nil
}

func primaryIDValue(primaryIDs map[reflect.Type][]string, infoByType map[reflect.Type]seedModelInfo, modelType reflect.Type, jsonName string, idx int) (string, bool) {
	ids, ok := primaryIDs[modelType]
	if !ok {
		return "", false
	}
	if idx < 0 || idx >= len(ids) {
		return "", false
	}
	info, ok := infoByType[modelType]
	if !ok {
		return "", false
	}
	if jsonName == "" {
		return "", false
	}
	if info.primaryJSON != "" && jsonName != info.primaryJSON {
		return "", false
	}
	return ids[idx], true
}

func foreignIDValue(primaryIDs map[reflect.Type][]string, primaryByJSON map[string]seedModelInfo, modelType reflect.Type, jsonName string, idx int) (string, bool) {
	rel, ok := primaryByJSON[jsonName]
	if !ok {
		return "", false
	}
	if rel.modelType == modelType {
		return "", false
	}
	ids, ok := primaryIDs[rel.modelType]
	if !ok || len(ids) == 0 {
		return "", false
	}
	return ids[idx%len(ids)], true
}

func isTextType(t reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Kind() == reflect.String
}
