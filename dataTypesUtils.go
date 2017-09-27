package main

import (
	"log"
	"strings"

	"github.com/silviucm/pgx"
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

func GetGoTypeForColumn(columnType string, nullable bool) (typeReturn, nullableTypeReturn, nullableCreateFunc, goTypeToImport string) {

	typeReturn = ""
	goTypeToImport = ""
	nullableTypeReturn = ""
	nullableCreateFunc = "" // For compatibility reasons, see silviucm/pgx/compat.go (e.g. CreateNullString)

	switch columnType {

	case "boolean":
		typeReturn = "bool"
		if nullable {
			nullableTypeReturn = "NullBool"
			nullableCreateFunc = "CreateNullBool"
		}

	case "character varying", "text":
		typeReturn = "string"
		if nullable {
			nullableTypeReturn = "NullString"
			nullableCreateFunc = "CreateNullString"
		}

	case "double precision":
		typeReturn = "float64"
		if nullable {
			nullableTypeReturn = "NullFloat64"
			nullableCreateFunc = "CreateNullFloat64"
		}

	case "integer", "serial":
		typeReturn = "int32"
		if nullable {
			nullableTypeReturn = "NullInt32"
			nullableCreateFunc = "CreateNullInt32"
		}

	case "json", "jsonb":
		typeReturn = "string"
		if nullable {
			nullableTypeReturn = "NullString"
			nullableCreateFunc = "CreateNullString"
		}

	case "numeric":
		typeReturn = "string"
		if nullable {
			nullableTypeReturn = "NullString"
			nullableCreateFunc = "CreateNullString"
		}

	case "uuid":
		typeReturn = "string"
		if nullable {
			nullableTypeReturn = "NullString"
			nullableCreateFunc = "CreateNullString"
		}

	case "bigint", "bigserial":
		typeReturn = "int64"
		if nullable {
			nullableTypeReturn = "NullInt64"
			nullableCreateFunc = "CreateNullInt64"
		}

	case "timestamp with time zone", "timestamp without time zone":

		typeReturn = "time.Time"
		goTypeToImport = "time"

		if nullable {
			nullableTypeReturn = "NullTime"
			nullableCreateFunc = "CreateNullTime"
		}
	}

	return typeReturn, nullableTypeReturn, nullableCreateFunc, goTypeToImport
}

func GetGoTypeNullableType(goType string) string {

	switch goType {

	case "bool":
		return "NullBool"
	case "int32", "serial":
		return "NullInt32"
	case "int64", "bigserial":
		return "NullInt64"
	case "string":
		return "NullString"
	case "time.Time":
		return "NullTime"
	}

	return ""
}

func DecodeIsColumnSequence(columnDefaultValue pgx.NullString) bool {

	if columnDefaultValue.Valid == false {
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

func DecodeMaxLength(maxLength pgx.NullInt32) int {

	if maxLength.Valid == false {
		return -1
	}

	return int(maxLength.Int32)
}
