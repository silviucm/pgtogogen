package main

/* Base Templates */

const BASE_TEMPLATE = `package {{.PackageName}}

/* *********************************************************** */
/* This file was automatically generated by pgtogogen.         */
/* Do not modify this file unless you know what you are doing. */
/* *********************************************************** */

import (
	"github.com/silviucm/pgx"	
	"bytes"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"
 	"github.com/twinj/uuid"
	"reflect"
)

// Wrapper structure over the pgx transaction package, so we don't need to import
// that package in the generated table-to-struct files.
type Transaction struct {
	Tx *pgx.Tx
}

// Interface to allow cache provider other than the default, in-memory cache
type ICacheProvider interface {
	Get(key string) (interface{}, error)
	Set(key string, value interface{})
	Exists(key string) bool
}

// Caching flags to allow functions and methods be supplied with caching behaviour options
const (
	// do not use caching
	FLAG_CACHE_DISABLE int = 0
	
	// if the cache is already there, use it, otherwise populate it from database
	FLAG_CACHE_USE int = 1
	
	// forces the reload of the cache from the database
	FLAG_CACHE_RELOAD int = 2
	
	// delete the cache entry and do not use it
	FLAG_CACHE_DELETE int = 4
)


// If this flag is set to true, the system will panic if the database
// connection cannot be made. Otherwise, GetDb() will simply return nil.
const FLAG_PANIC_ON_INIT_DB_FAIL bool = true

// variables that mimick the database driver standard errors, so
// we don't need to import that package in the generated table-to-struct files
// or any other package, such as pgx - the import would only reside here, in the base file

var ErrNoRows = pgx.ErrNoRows
var ErrDeadConn = pgx.ErrDeadConn
var ErrTxClosed = pgx.ErrTxClosed
var ErrNotificationTimeout = pgx.ErrNotificationTimeout

// Transaction isolation levels for the pgx package

var IsoLevelSerializable = pgx.Serializable
var IsoLevelRepeatableRead = pgx.RepeatableRead
var IsoLevelReadCommitted = pgx.ReadCommitted
var IsoLevelReadUncommitted = pgx.ReadUncommitted

// debug mode flag
var isDebugMode bool = false

var dbHandle *pgx.ConnPool

func GetDb() *pgx.ConnPool {
	
	if dbHandle != nil {
		return dbHandle
	}

	dbSettings := GetDefaultDbSettings()
	newHandle, err := InitDatabase(dbSettings)
	
	if err != nil {
		if FLAG_PANIC_ON_INIT_DB_FAIL {
			panic("FORCED PANIC: models.GetDb() -> InitDatabase() fatal error connecting to the database: " + err.Error())
		} else {
			return nil
		}
	}

	
	dbHandle = newHandle	
	return dbHandle
	
}

// Returns a ConnPoolConfig structure.
func GetDefaultDbSettings() pgx.ConnPoolConfig {
	
	var config pgx.ConnPoolConfig

	config.Host = DB_HOST
	config.User = DB_USER
	config.Password = DB_PASS
	config.Database = DB_NAME
	config.Port = DB_PORT
	config.MaxConnections = DB_POOL_MAX_CONNECTIONS
	
	return config
	
}

// Minimally, the pgx.ConnPoolConfig expects these values to be set:
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

func InitDatabaseMinimal(host string, port uint16, user, pass, dbName string, poolMaxConnections int) (*pgx.ConnPool, error) {

	DB_HOST = host
	DB_USER = user
	DB_PASS = pass
	DB_NAME = dbName
	DB_PORT = port
	DB_POOL_MAX_CONNECTIONS = poolMaxConnections
	
	return InitDatabase(GetDefaultDbSettings())

}

/* BEGIN Transactions utility functions */

// Begins and returns a transaction using the default isolation level.
// Unlike TxWrap, it is the responsibility of the caller to commit and
// rollback the transaction if necessary.
func TxBegin() (*Transaction, error) {

	txWrapper := &Transaction{}
	tx, err := GetDb().Begin()

	if err != nil {
		return nil, err
	} else {
		txWrapper.Tx = tx
		return txWrapper, nil
	}

}

// Begins and returns a transaction using the specified isolation level.
// The following global variables can be passed:
// models.IsoLevelSerializable
// models.IsoLevelRepeatableRead
// models.IsoLevelReadCommitted
// models.IsoLevelReadUncommitted
func TxBeginIso(isolationLevel string) (*Transaction, error) {

	txWrapper := &Transaction{}
	tx, err := GetDb().BeginIso(isolationLevel)

	if err != nil {
		return nil, err
	} else {
		txWrapper.Tx = tx
		return txWrapper, nil
	}

}

func TxWrap(wrapperFunc func(tx *Transaction) error) error {

	var errorPrefix = "TxWrap() ERROR: "

	realTx, err := GetDb().Begin()
	if err != nil {
		return NewModelsError(errorPrefix+"GetDb().Begin() error: ", err)
	}

	// pgx package note: Rollback is safe to call even if the tx is already closed,
	// so if the tx commits successfully, this is a no-op
	defer realTx.Rollback()

	// wrap the real tx into our wrapper
	tx := &Transaction{Tx: realTx}

	err = wrapperFunc(tx)
	if err != nil {
		return NewModelsError(errorPrefix+"inner wrapperFunc() error - will return and rollback: ", err)
	}

	err = realTx.Commit()
	if err != nil {
		return NewModelsError(errorPrefix+"tx.Commit() error: ", err)
	}

	return nil

}


/* END Transactions utility functions */

/* BEGIN Error and Logging utility functions */

// Wraps an already existing error with a localized prefix
func NewModelsError(errorPrefix string, originalError error) error {
	return errors.New(errorPrefix + ": " + originalError.Error())
}

// Wraps local issues in an error format, without needing an already existing error
func NewModelsErrorLocal(errorPrefix string, localError string) error {
	return errors.New(errorPrefix + ": " + localError)
}

// Logs the message to the console 
func Debug(v ...interface{}) {
	if isDebugMode {
		log.Println(v)
	}
}

// Set the debug mode to true or false
func SetDebugMode(debugMode bool) {
	isDebugMode = debugMode
}

func IsDebugMode() bool  {
	return isDebugMode
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

// Wrapper over time package Now method
func Now() time.Time {
	return time.Now()
}

// Returns a new Guid
func NewGuid() string {
	return uuid.NewV4().String()
}

// Wrapper over strconv package Itoa method
func Itoa(intValue int) string {
	return strconv.Itoa(intValue)
}

func Contains(source string, subStr string) bool {
	return strings.Contains(source, subStr)
}

// Wrapper over strings.Join
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

// Sort comparator for bool type
func LessComparatorFor_bool(first, second bool) bool { return first == false }

// Because LessComparatorFor_time.Time would break the compiler if a function would be
// defined as such (due to the dot) we need to create a fake struct
type tLessComparatorFor_time struct {}
var LessComparatorFor_time *tLessComparatorFor_time
func (t *tLessComparatorFor_time) Time(first, second time.Time) bool {  return first.Before(second) }

/* BEGIN conversion methods */
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
