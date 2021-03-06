package main

/* Base Templates */

const BASE_TEMPLATE = `package {{.PackageName}}

/* *********************************************************** */
/* This file was automatically generated by pgtogogen.         */
/* Do not modify this file unless you know what you are doing. */
/* *********************************************************** */

import (
	pgx "{{.PgxImport}}"		
	"bytes"
	"crypto/tls"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"
 	"github.com/silviucm/uuid"
	"reflect"
)

// ICacheProvider is an interface that describes caching operations. It allows using a cache 
// provider other than the default, in-memory cache
type ICacheProvider interface {
	Get(key string) (interface{}, error)
	Set(key string, value interface{})
	Exists(key string) bool
}

// Caching flags to allow functions and methods be supplied with caching behaviour options
const (
	// PgToGoFlagCacheDisable disables any caching
	PgToGoFlagCacheDisable int = 0
	
	// PgToGoFlagCacheUse uses the cache already present, otherwise populates data from database
	PgToGoFlagCacheUse int = 1
	
	// PgToGoFlagCacheReload forces a cache refresh from the database
	PgToGoFlagCacheReload int = 2
	
	// PgToGoFlagCacheDelete forces the cached entry deletion, preventing further use
	PgToGoFlagCacheDelete int = 4
)


// PgToGoOptionPanicOnInitDbErr defines the exit behaviour when, during a GetDb lazy-init,
// the InitDatabase function attempts to connect to the datasource and fails. 
// When set to true, the system panics. Otherwise, GetDb() will simply return nil.
var PgToGoOptionPanicOnInitDbErr = true

// variables that mimick the database driver standard errors, so
// we don't need to import that package in the generated table-to-struct files
// or any other package, such as pgx - the import would only reside here, in the base file
var (
	// ErrNoRows occurs when rows are expected but none are returned.
	ErrNoRows = pgx.ErrNoRows
	
	// ErrDeadConn occurs on an attempt to use a dead connection.
	ErrDeadConn = pgx.ErrDeadConn
	
	// ErrTxClosed occurs on an attempt to use an already closed transaction.
	ErrTxClosed = pgx.ErrTxClosed
	
	// ErrNotificationTimeout occurs in case of a notification timeout.
	ErrNotificationTimeout = errors.New("notification timeout")
	
	// ErrTooManyRows occurs when a db query returns more than one row for an operation
	// expecting one row at most.
	ErrTooManyRows =  errors.New("More than one row returned")
)

// debug mode flag
var isDebugMode = false

var dbHandle *pgx.ConnPool

// GetDb returns the connection pool handle for the underlying data source.
// If the handle is nil, it attempts a lazy initialization.
func GetDb() *pgx.ConnPool {
	
	if dbHandle != nil {
		return dbHandle
	}

	dbSettings := GetDefaultDbSettings()
	newHandle, err := InitDatabase(dbSettings)
	
	if err != nil {
		if PgToGoOptionPanicOnInitDbErr {
			panic("FORCED PANIC: models.GetDb() -> InitDatabase() fatal error connecting to the database: " + err.Error())
		} else {
			return nil
		}
	}
	
	dbHandle = newHandle	
	return dbHandle	
}

// GetDefaultDbSettings returns a ConnPoolConfig structure.
func GetDefaultDbSettings() pgx.ConnPoolConfig {
	
	var config pgx.ConnPoolConfig

	config.Host = DB_HOST
	config.User = DB_USER
	config.Password = DB_PASS
	config.Database = DB_NAME
	config.Port = DB_PORT
	config.MaxConnections = DB_POOL_MAX_CONNECTIONS
	
	// SSL options
	if DB_SSL == "prefer" || DB_SSL == "require" || DB_SSL == "allow" {
		config.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	} else if DB_SSL == "verify-ca" || DB_SSL == "verify-full" {
		config.TLSConfig = &tls.Config{}
	}	
	
	return config
	
}

// InitDatabase initializes the connection to the datasource using a pgx.ConnPoolConfig
// instance. It then calls PrepareDbCollections to initialize the structures and set
// default behaviours.
//
// At minimum, the pgx.ConnPoolConfig expects these values to be set:
//
// config.Host = dbHostStringVar
// config.User = dbUserStringVar
// config.Password = dbPassStringVar
// config.Database = dbNameStringVar
// config.Port = dbPortUInt16Var
//
// You can use the GetDefaultDbSettings() and modify the variables at the beginning
// of this class accordingly.
func InitDatabase(dbConfig pgx.ConnPoolConfig) (*pgx.ConnPool, error) {

	
	connPool, err := pgx.NewConnPool(dbConfig)
	if err != nil {
		return nil, NewModelsError("models.InitDatabase() -> pgx.NewConnPool", err)
		
	} 

	// prepare the Tables, Views, Functions collections with whatever
	// initialization or default behavior necessary
	PrepareDbCollections()

	dbHandle = connPool
	return dbHandle, nil

}

// InitDatabaseMinimal is a wrapper over InitDatabase, and allows direct values to be
// specied for the database connection parameters.
func InitDatabaseMinimal(host string, port uint16, user, pass, dbName string, poolMaxConnections int) (*pgx.ConnPool, error) {

	DB_HOST = host
	DB_USER = user
	DB_PASS = pass
	DB_NAME = dbName
	DB_PORT = port
	DB_POOL_MAX_CONNECTIONS = poolMaxConnections
	
	return InitDatabase(GetDefaultDbSettings())

}

// InitDatabaseSetSSLMode allows setting the ssl mode for subsequent db initialization.
// It should be called right before InitDatabaseMinimal.
// The sslmode value can be "allow", "prefer", "require", "verify-ca" or "verify-full".
// An empty string or "disable" signify no SSL. Passing an unknown value is the same 
// as passing an empty string.
func InitDatabaseSetSSLMode(sslmode string) {
	DB_SSL = sslmode
}

/* BEGIN Error and Logging utility functions */

// NewModelsError wraps an already existing error with a localized prefix.
// If the error is of type pgx.PgError then its Code field value is automatically
// transferred to the wrapper error.
func NewModelsError(errorPrefix string, originalError error) error {

	if pgErr, ok := originalError.(pgx.PgError); ok {
		return &pgToGoGenError{
			Err:           errorPrefix + ": " + originalError.Error(),
			OriginalError: originalError,
			Code:          pgErr.Code,
		}
	}
	return &pgToGoGenError{
		Err:           errorPrefix + ": " + originalError.Error(),
		OriginalError: originalError,
	}
}

// NewModelsErrorWithCode wraps an already existing error with a localized prefix.
// A code can be specified, which could be the code of the original error.
func NewModelsErrorWithCode(errorPrefix string, originalError error, code string) error {
	return &pgToGoGenError{
		Err:           errorPrefix + ": " + originalError.Error(),
		OriginalError: originalError,
		Code:          code,
	}
}

// NewModelsErrorLocal wraps locally occuring errors in a standardized error format,
// without the needing of an already existing error.
func NewModelsErrorLocal(errorPrefix string, localError string) error {
	return &pgToGoGenError{
		Err:           errorPrefix + ": " + localError,
		OriginalError: nil,
	}
}

// NewModelsErrorLocalWithCode wraps locally occuring errors in a standardized error format,
// along with an established code, without the needing of an already existing error.
func NewModelsErrorLocalWithCode(errorPrefix string, localError string, code string) error {
	return &pgToGoGenError{
		Err:           errorPrefix + ": " + localError,
		OriginalError: nil,
		Code:          code,
	}
}

// GetOriginalError attempts to retrieve the original, embedded error if there is one
// in the wrapper error. It returns an error or nil if no original error found.
func GetOriginalError(err error) error {
	if pgtgErr, ok := err.(*pgToGoGenError); ok {
		if pgtgErr.OriginalError != nil {
			return pgtgErr.OriginalError
		}
	}
	return nil
}

// GetPostgresErrorCode attempts to retrieve the Postgres error code as defined at:
// https://www.postgresql.org/docs/current/static/errcodes-appendix.html
// To obtain the code it attempts to detect if the supplied error is either
// a locally defined *pgToGoGenError or a pgx-defined pgx.PgError.
// The latter has priority.
// It returns the code or empty string if it cannot find it.
func GetPostgresErrorCode(err error) string {
	// Assume an error wrapper first
	if pgtgErr, ok := err.(*pgToGoGenError); ok {
		if pgtgErr.OriginalError != nil {
			if pgErr, ok := err.(pgx.PgError); ok {
				return pgErr.Code
			}
		}
		return pgtgErr.Code
	}
	// Attempt a type assertion to pgx.PgError directly
	if pgErr, ok := err.(pgx.PgError); ok {
		return pgErr.Code
	}
	return ""
}

// Debug logs the info using the runtime log package if debug mode is on.
func Debug(v ...interface{}) {
	if isDebugMode {
		log.Println(v...)
	}
}

// SetDebugMode sets the debug mode to true or false.
func SetDebugMode(debugMode bool) {
	isDebugMode = debugMode
}

// IsDebugMode returns true if debug mode is set to on.
func IsDebugMode() bool {
	return isDebugMode
}

type pgToGoGenError struct {
	Err           string
	Code          string
	OriginalError error
}

func (pErr *pgToGoGenError) Error() string {
	return pErr.Err
}

/* END Error and Logging utility functions */

func GetGoTypeForColumn(columnType string) (typeReturn string, goTypeToImport string) {

	typeReturn = ""
	goTypeToImport = ""

	switch columnType {
	case "character varying":
		typeReturn = "string"
	case "integer", "serial":
		typeReturn = "int32"
	case "boolean":
		typeReturn = "bool"
	case "uuid":
		typeReturn = "string"
	case "bigint":
		typeReturn = "int64"
	case "timestamp with time zone":
		typeReturn = "time.Time"
		goTypeToImport = "time"
	}

	return typeReturn, goTypeToImport
}

// Returns the string composed of the condition parameter and the stringified
// param variadic list interface{} members
func GetHashFromConditionAndParams(condition string, params ...interface{}) (string, error) {
	
	var errorPrefix = "GetHashFromConditionAndParams() ERROR: "
	
	// define the delete query
	hashBuffer := bytes.Buffer{}
	_, writeErr := hashBuffer.WriteString(condition)
	if writeErr != nil {
		return "", NewModelsError(errorPrefix + "hashBuffer.WriteString error (condition parameter):",writeErr)
	}

	for _,currentParam := range params {
		
		switch currentParam.(type) {
		case int:
			_, writeErr = hashBuffer.WriteString(Itoa(currentParam.(int)))
			if writeErr != nil {
				return "", NewModelsError(errorPrefix + "queryBuffer.WriteString error:",writeErr)
			}
		case float64:
			_, writeErr = hashBuffer.WriteString(strconv.FormatFloat(currentParam.(float64), 'f', 6, 64))
			if writeErr != nil {
				return "", NewModelsError(errorPrefix + "queryBuffer.WriteString error:",writeErr)
			}
		case string:
			_, writeErr = hashBuffer.WriteString(currentParam.(string))
			if writeErr != nil {
				return "", NewModelsError(errorPrefix + "queryBuffer.WriteString error:",writeErr)
			}
		case time.Time:
			_, writeErr = hashBuffer.WriteString((currentParam.(time.Time)).Format(time.RFC3339))
			if writeErr != nil {
				return "", NewModelsError(errorPrefix + "queryBuffer.WriteString error:",writeErr)
			}				    
		default:
			return "", NewModelsErrorLocal(errorPrefix, "undetermined interface type: " + reflect.TypeOf(currentParam).String())			
		}						
	}

	return hashBuffer.String(), nil
	
}

// Now is a wrapper over the time package Now method.
func Now() time.Time {
	return time.Now()
}

// Converts the provided time.Time to an UTC location.
// Necessary for pgtype.Timestamp (without timezone)
func utcTime(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), value.Hour(), value.Minute(), value.Second(), value.Nanosecond(), time.UTC)
}

// Returns a new Guid
func NewGuid() string {
	return uuid.NewV4().String()
}

// Itoa is a wrapper over strconv package Itoa method.
func Itoa(intValue int) string {
	return strconv.Itoa(intValue)
}

// Contains is a wrapper over the strings package Contains method.
func Contains(source string, subStr string) bool {
	return strings.Contains(source, subStr)
}

// JoinStringParts is a wrapper over strings.Join
func JoinStringParts(sourceSlice []string, separator string) string {
	return strings.Join(sourceSlice, separator)
}

// Sort comparator for string type
func LessComparatorFor_string(first, second string) bool { return first < second }

// Sort comparator for int type
func LessComparatorFor_int(first, second int) bool { return first < second }

// Sort comparator for int32 type
func LessComparatorFor_int32(first, second int32) bool { return first < second }

// Sort comparator for int64 type
func LessComparatorFor_int64(first, second int64) bool { return first < second }

// Sort comparator for float64 type
func LessComparatorFor_float64(first, second float64) bool { return first < second }

// Sort comparator for bool type
func LessComparatorFor_bool(first, second bool) bool { return first == false }

// Because LessComparatorFor_time.Time would break the compiler if a function would be
// defined as such (due to the dot) we need to create a fake struct
type tLessComparatorFor_time struct {}
var LessComparatorFor_time *tLessComparatorFor_time
func (t *tLessComparatorFor_time) Time(first, second time.Time) bool {  return first.Before(second) }

/* BEGIN conversion methods */

func BoolToNilInterface(boolVal bool) interface{} {
	return nil
}

const (
	// See http://golang.org/pkg/time/#Parse
	comparisonTimeFormat = "2006-01-02 15:04:05 MST"
)

// To be able to be properly parsed, the string must be in the following format
// "YYYY-MM-DD HH:MM:SS" (e.g. 2014-12-22 18:24:43)
func To_Time_FromString(timeDateStr string) (time.Time, error) {
	
	var errorPrefix = "To_Time_FromString() ERROR: "
	
	if timeDateStr == "" {
		return time.Now(), NewModelsErrorLocal(errorPrefix, "The input parameter is an empty string.")
	}		
	
	return time.Parse(comparisonTimeFormat, timeDateStr)
}

func To_bool_FromString(boolStr string) (bool, error) {

	var errorPrefix = "To_bool_FromString() ERROR: "
	
	if boolStr == "" {
		return false, NewModelsErrorLocal(errorPrefix, "The input parameter is an empty string.")
	}	
	
	if boolStr == "0" || boolStr == "n" || boolStr == "N" || boolStr == "No" || boolStr == "no" || boolStr == "NO" || boolStr == "false" || boolStr == "FALSE" || boolStr == "False" || boolStr == "f" || boolStr == "F" { return false, nil }
	if boolStr == "1" || boolStr == "y" || boolStr == "Y" || boolStr == "Yes" || boolStr == "yes" || boolStr == "YES" || boolStr == "true" || boolStr == "TRUE" || boolStr == "True" || boolStr == "t" || boolStr == "T" { return true, nil }
	
	return false, NewModelsErrorLocal(errorPrefix, "The input string parameter cannot be converted to bool type.")
}

func To_int32_FromString(int32Str string) (int32, error) {
	
	var errorPrefix = "To_int32_FromString() ERROR: "
	
	if int32Str == "" {
		return -1, NewModelsErrorLocal(errorPrefix, "The input parameter is an empty string.")
	}	
	
	i, err := strconv.ParseInt(int32Str, 10, 32)
	if err != nil {return -1, err }
	
	return int32(i), nil
}


func To_int64_FromString(int64Str string) (int64, error) {
	
	var errorPrefix = "To_int64_FromString() ERROR: "
	
	if int64Str == "" {
		return -1, NewModelsErrorLocal(errorPrefix, "The input parameter is an empty string.")
	}		
		
	return strconv.ParseInt(int64Str, 10, 64)

}

func To_float64_FromString(float64Str string) (float64, error) {
	
	var errorPrefix = "To_float64_FromString() ERROR: "
	
	if float64Str == "" {
		return -1, NewModelsErrorLocal(errorPrefix, "The input parameter is an empty string.")
	}		
		
	return strconv.ParseFloat(float64Str, 64)

}
`

const BASE_TEMPLATE_SETTINGS = `package {{.PackageName}}

/* ************************************************************* */
/* This file was automatically generated by pgtogogen.           */
/* This is a one-time only generation. Customize when necessary. */
/* ************************************************************* */

// Database settings variables, with initial, dummy values

var DB_HOST string = "localhost"
var DB_PORT uint16 = 5432
var DB_USER string = "testuser"
var DB_PASS string = "testuser"
var DB_NAME string = "testdb"
var DB_POOL_MAX_CONNECTIONS int = 100
var DB_SSL string = ""
`

const BASE_TEMPLATE_COLLECTIONS = `package {{.PackageName}}

/* ************************************************************* */
/* This file was automatically generated by pgtogogen.           */
/* This is a one-time only generation. Customize when necessary. */
/* ************************************************************* */

{{if and .Tables (lt 0 (len .Tables))}}
// Container struct for table collections
type stTables struct {
	{{range .Tables}}{{.GoFriendlyName}} t{{.GoFriendlyName}}Utils
	{{end}}
	PgToGo_IgnorePKValuesWhenInsertingAndUseSequence bool // set this to true if you want Inserts to ignore the PK fields

	// Set this to true if you want New or Create operations to automatically
	// set all time.Time (datetime) fields to time.Now()
	PgToGo_SetDateTimeFieldsToNowForNewRecords bool 

	// Set this to true if you want New or Create operations to automatically
	// set all Guid fields to a new guid
	PgToGo_SetGuidFieldsToNewGuidsNewRecords bool
}

var Tables stTables

// Iterates through all tables prefixed with "lookup" (case-insensitive)
// and enables cache, then loads all rows inside the respective caches
func (t *stTables) CacheLookupTables() {
	
	{{range .Tables}}{{if startsWith .GoFriendlyName "lookup"}} t.{{.GoFriendlyName}}.Cache.EnableAndLoadAllRows()
	{{end}}{{end}}	
}
{{end}}

{{if and .Views (lt 0 (len .Views))}}
// Container struct for view collections
type stViews struct {
	{{range .Views}}{{.GoFriendlyName}} t{{.GoFriendlyName}}Utils
	{{end}}	
}

var Views stViews
{{end}}

// This gets called in case of a successful InitDatabase() call.
// Customize what happens inside as necessary.
func PrepareDbCollections() {

	{{if and .Tables (lt 0 (len .Tables))}}
	// Tables-specific default settings
	
	// by setting this to true, the inserts will assume PKs are inserted by the database
	// so whatever PK id is set in the structure will be ignored for insert operations
	Tables.PgToGo_IgnorePKValuesWhenInsertingAndUseSequence = true

	// by setting this to true, whenever a New() or CreateFrom...() method is called
	// to generate a new table instance struct, the time.Time fields will be automatically
	// populate to time.Now()
	Tables.PgToGo_SetDateTimeFieldsToNowForNewRecords = true

	// by setting this to true, whenever a New() or CreateFrom...() method is called
	// to generate a new table instance struct, the Guid fields will be automatically
	// populated with a newly generated Guid
	Tables.PgToGo_SetGuidFieldsToNewGuidsNewRecords = true
		
	{{end}}
}

`

const BASE_TEMPLATE_FORMS = `package {{.PackageName}}

/* ************************************************************* */
/* This file was automatically generated by pgtogogen.           */
/* This is a one-time only generation. Customize when necessary. */
/* ************************************************************* */

// The Validator interface enables structs that implement it
// to return the validation state for that particular instance
type Validator interface {
	Validate() (bool, []error)
}

`
