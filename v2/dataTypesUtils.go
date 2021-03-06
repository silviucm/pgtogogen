package main

import (
	"log"
	"strings"

	pgtype "github.com/silviucm/pgtogogen/v2/internal/pgx/pgtype"
)

const (
	NULLABLE_TYPE_BOOL         = "pgtype.Bool"
	NULLABLE_TYPE_FLOAT32      = "pgtype.Float4"
	NULLABLE_TYPE_FLOAT64      = "pgtype.Float8"
	NULLABLE_TYPE_INT16        = "pgtype.Int2"
	NULLABLE_TYPE_INT32        = "pgtype.Int4"
	NULLABLE_TYPE_INT64        = "pgtype.Int8"
	NULLABLE_TYPE_INTERVAL     = "pgtype.Interval"
	NULLABLE_TYPE_JSON         = "JSON"
	NULLABLE_TYPE_JSONB        = "JSONB"
	NULLABLE_TYPE_NUMERIC      = "Numeric"
	NULLABLE_TYPE_STRING       = "pgtype.Text"
	NULLABLE_TYPE_TEXT         = "pgtype.Text"
	NULLABLE_TYPE_VARCHAR      = "pgtype.Varchar"
	NULLABLE_TYPE_UUID         = "pgtype.UUID"
	NULLABLE_TYPE_TIMESTAMP_TZ = "pgtype.Timestamptz"
	NULLABLE_TYPE_TIMESTAMP    = "pgtype.Timestamp"
	NULLABLE_TYPE_DATE         = "pgtype.Date"
)

/* Utility methods for dealing with SQL data types in general and PostgreSQL data types in particular */

func GetGoFriendlyNameForColumn(columnName string) string {

	// find if the table name has underscore
	if strings.Contains(columnName, "_") == false {
		return strings.Title(columnName)
	}

	subNames := strings.Split(columnName, "_")

	if subNames == nil {
		log.Fatal("GetGoFriendlyNameForColumn() fatal error for column name: ", columnName, ". Please ensure a valid column name is provided.")
	}

	for i := range subNames {
		subNames[i] = strings.Title(subNames[i])
	}

	return strings.Join(subNames, "")
}

// GetGoInsertNameForColumn returns the Go-friendly column names specialized for insert
// operations. The reason for this is that when we use custom types such as Numeric,
// instead of the underlying pgx (pgtypes) Numeric, the insert operation will fail, as
// pgx does not know how to convert the custom type. For such exceptions, provide additional
// methods and expose them when rendering the Go-friendly column name in the insert
// method generation.
// Any other types that do not require these customization will simply return the Go name.
func GetGoInsertNameForColumn(currentGoName, resolvedGoType string) string {
	switch resolvedGoType {
	case NULLABLE_TYPE_NUMERIC:
		return currentGoName + ".EmbeddedVal()"
	}
	return currentGoName
}

// GetGoTypeForColumn identifies the proper Go type for the database type
// specified in columnType. An additional udtName may be needed for array types, in
// which case udtName is not empty. (e.g. character[] will come as columnType=ARRAY
// and udtName=_bpchar).
func GetGoTypeForColumn(columnType string, nullable bool, udtName string) (typeReturn,
	nullableTypeReturn, goTypeToImport string) {

	typeReturn = ""
	goTypeToImport = ""
	nullableTypeReturn = ""

	switch columnType {

	case "ARRAY":
		if udtName == "_bpchar" {
			typeReturn = "string"
			if nullable {
				nullableTypeReturn = NULLABLE_TYPE_STRING
			}
		}

	case "boolean":
		typeReturn = "bool"
		if nullable {
			nullableTypeReturn = NULLABLE_TYPE_BOOL
		}

	case "character varying", "text", "character", "character[]":
		typeReturn = "string"
		if nullable {
			nullableTypeReturn = NULLABLE_TYPE_STRING
		}

	case "double precision":
		typeReturn = "float64"
		if nullable {
			nullableTypeReturn = NULLABLE_TYPE_FLOAT64
		}

	case "int2", "smallint":
		typeReturn = "int16"
		if nullable {
			nullableTypeReturn = NULLABLE_TYPE_INT16
		}

	case "int4", "int32", "integer", "serial":
		typeReturn = "int32"
		if nullable {
			nullableTypeReturn = NULLABLE_TYPE_INT32
		}

	case "interval":
		typeReturn = NULLABLE_TYPE_INTERVAL
		if nullable {
			nullableTypeReturn = NULLABLE_TYPE_INTERVAL
		}

	// We need to make sure have a "JSON" type embedding pgtypes.JSON
	// inside the generated models package
	case "json":
		typeReturn = "string"
		if nullable {
			nullableTypeReturn = NULLABLE_TYPE_JSON
		}

	// We need to make sure have a "JSONB" type embedding pgtypes.JSONB
	// inside the generated models package
	case "jsonb":
		typeReturn = "string"
		if nullable {
			nullableTypeReturn = NULLABLE_TYPE_JSONB
		}

	case "numeric":
		typeReturn = NULLABLE_TYPE_NUMERIC
		if nullable {
			nullableTypeReturn = NULLABLE_TYPE_NUMERIC
		}

	case "uuid":
		typeReturn = "string"
		if nullable {
			nullableTypeReturn = NULLABLE_TYPE_UUID
		}

	case "int8", "bigint", "bigserial", "int64":
		typeReturn = "int64"
		if nullable {
			nullableTypeReturn = NULLABLE_TYPE_INT64
		}

	case "timestamp with time zone":

		typeReturn = "time.Time"
		goTypeToImport = "time"

		if nullable {
			nullableTypeReturn = NULLABLE_TYPE_TIMESTAMP_TZ
		}

	case "timestamp without time zone":

		typeReturn = "time.Time"
		goTypeToImport = "time"

		if nullable {
			nullableTypeReturn = NULLABLE_TYPE_TIMESTAMP
		}

	case "time with time zone":

		typeReturn = "time.Time"
		goTypeToImport = "time"

		if nullable {
			nullableTypeReturn = NULLABLE_TYPE_TIMESTAMP_TZ
		}

	case "time without time zone":

		typeReturn = "time.Time"
		goTypeToImport = "time"

		if nullable {
			nullableTypeReturn = NULLABLE_TYPE_TIMESTAMP
		}
	case "date":

		typeReturn = "time.Time"
		goTypeToImport = "time"

		if nullable {
			nullableTypeReturn = NULLABLE_TYPE_DATE
		}
	}

	return typeReturn, nullableTypeReturn, goTypeToImport
}

func GetGoTypeNullableType(goType string) string {

	switch goType {

	case "bool":
		return NULLABLE_TYPE_BOOL
	case "float32":
		return NULLABLE_TYPE_FLOAT32
	case "float64":
		return NULLABLE_TYPE_FLOAT64
	case "int2", "smallint":
		return NULLABLE_TYPE_INT16
	case "int4", "int32", "integer", "serial":
		return NULLABLE_TYPE_INT32
	case "int8", "int64", "bigserial", "bigint":
		return NULLABLE_TYPE_INT64
	case "Numeric":
		return NULLABLE_TYPE_NUMERIC
	case "JSONString":
		return NULLABLE_TYPE_JSON
	case "JSONBString":
		return NULLABLE_TYPE_JSONB
	case "string":
		return NULLABLE_TYPE_STRING
	case "time.Time":
		return NULLABLE_TYPE_TIMESTAMP_TZ
	}

	return ""
}

// GenerateNullableTypeStructTemplate is a convenience method to be used when
// generating the models.
// It produces a Go source code sequence that instantiates a pgx nullable
// type struct. For example, a call such as:
//
//  GenerateNullableTypeStructTemplate("pgtype.Varchar", "sourceCmsArticle.Overview", "sourceCmsArticle.Overview_IsNotNull")
//
// would generate the following string:
//
//  "&pgtype.Varchar{String:sourceCmsArticle.Overview, Status: sourceCmsArticle.Overview_Is_Present}"
//
// The forInsert flag is used to generate particular code for insert statements. For example, the
// custom Numeric type is not recognized by pgx own Numeric type, so, at insert, we need to use that one.
func GenerateNullableTypeStructTemplate(goNullableType, valueField, statusField string, forInsert bool) string {

	switch goNullableType {

	case NULLABLE_TYPE_BOOL:
		return "&pgtype.Bool{Bool: " + valueField + ", Status: statusFromBool(" + statusField + ")}"
	case NULLABLE_TYPE_FLOAT32:
		return "&pgtype.Float4{Float: " + valueField + ", Status: statusFromBool(" + statusField + ")}"
	case NULLABLE_TYPE_FLOAT64:
		return "&pgtype.Float8{Float: " + valueField + ", Status: statusFromBool(" + statusField + ")}"
	case NULLABLE_TYPE_NUMERIC:
		if forInsert {
			return "toPgxNumeric(" + valueField + ", " + statusField + ")"
		}
		return "toNumeric(" + valueField + ", " + statusField + ")"
	case NULLABLE_TYPE_INT16:
		return "&pgtype.Int2{Int: " + valueField + ", Status: statusFromBool(" + statusField + ")}"
	case NULLABLE_TYPE_INT32:
		return "&pgtype.Int4{Int: " + valueField + ", Status: statusFromBool(" + statusField + ")}"
	case NULLABLE_TYPE_INT64:
		return "&pgtype.Int8{Int: " + valueField + ", Status: statusFromBool(" + statusField + ")}"
	case NULLABLE_TYPE_JSON:
		return "&pgtype.JSON{Bytes: []byte(" + valueField + "), Status: statusFromBool(" + statusField + ")}"
	case NULLABLE_TYPE_JSONB:
		return "&pgtype.JSONB{Bytes: []byte(" + valueField + "), Status: statusFromBool(" + statusField + ")}"
	case NULLABLE_TYPE_TEXT:
		return "&pgtype.Text{String: " + valueField + ", Status: statusFromBool(" + statusField + ")}"
	case NULLABLE_TYPE_VARCHAR:
		return "&pgtype.Varchar{String: " + valueField + ", Status: statusFromBool(" + statusField + ")}"
	case NULLABLE_TYPE_TIMESTAMP_TZ:
		return "&pgtype.Timestamptz{Time: " + valueField + ", Status: statusFromBool(" + statusField + ")}"
	case NULLABLE_TYPE_TIMESTAMP:
		// Unfortunately, for Timestamp without timezone, we need to convert to
		// UTC location for pgtype.Timestamp to accept the value
		return "&pgtype.Timestamp{Time: utcTime(" + valueField + "), Status: statusFromBool(" + statusField + ")}"
	case NULLABLE_TYPE_DATE:
		return "&pgtype.Date{Time: " + valueField + ", Status: statusFromBool(" + statusField + ")}"
	}

	return "[GenerateNullableTypeStructTemplate: could not find the go nullable type: '" + goNullableType + "']"

}

func GetNullableTypeValueFieldName(goNullableType string) string {

	switch goNullableType {

	case NULLABLE_TYPE_BOOL:
		return "Bool"
	case NULLABLE_TYPE_FLOAT32:
		return "Float"
	case NULLABLE_TYPE_FLOAT64:
		return "Float"
	case NULLABLE_TYPE_NUMERIC:
		return "NumericVal()"
	case NULLABLE_TYPE_INT16:
		return "Int"
	case NULLABLE_TYPE_INT32:
		return "Int"
	case NULLABLE_TYPE_INT64:
		return "Int"
	case NULLABLE_TYPE_JSON:
		return "String()"
	case NULLABLE_TYPE_JSONB:
		return "String()"
	case NULLABLE_TYPE_TEXT:
		return "String"
	case NULLABLE_TYPE_VARCHAR:
		return "String"
	case NULLABLE_TYPE_TIMESTAMP_TZ:
		return "Time"
	case NULLABLE_TYPE_TIMESTAMP:
		return "Time"
	case NULLABLE_TYPE_DATE:
		return "Time"
	}

	return "[GetNullableTypeValueFieldName: could not find the go nullable type: '" + goNullableType + "']"

}

func DecodeIsColumnSequence(columnDefaultValue pgtype.Text) bool {

	if columnDefaultValue.Status == pgtype.Null {
		return false
	}

	if strings.HasPrefix(columnDefaultValue.String, "nextval(") {
		return true
	}

	return false
}

func DecodeNullable(isNullable string) bool {

	if isNullable == "NO" || isNullable == "f" || isNullable == "F" {
		return false
	}

	if isNullable == "YES" || isNullable == "Yes" || isNullable == "yes" || isNullable == "y" || isNullable == "Y" ||
		isNullable == "t" || isNullable == "T" || isNullable == "true" || isNullable == "TRUE" || isNullable == "True" {
		return true
	}

	return false
}

func DecodeMaxLength(maxLength pgtype.Int4) int {

	if maxLength.Status == pgtype.Null {
		return -1
	}

	return int(maxLength.Int)
}
