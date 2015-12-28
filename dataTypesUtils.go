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

func GetGoTypeForColumn(columnType string, nullable bool) (typeReturn string, nullableTypeReturn string, goTypeToImport string) {

	typeReturn = ""
	goTypeToImport = ""
	nullableTypeReturn = ""

	switch columnType {

	case "boolean":
		typeReturn = "bool"
		if nullable {
			nullableTypeReturn = "pgx.NullBool"
		}

	case "character varying", "text":
		typeReturn = "string"
		if nullable {
			nullableTypeReturn = "pgx.NullString"
		}

	case "double precision":
		typeReturn = "float64"
		if nullable {
			nullableTypeReturn = "pgx.NullFloat64"
		}

	case "integer", "serial":
		typeReturn = "int32"
		if nullable {
			nullableTypeReturn = "pgx.NullInt32"
		}

	case "json", "jsonb":
		typeReturn = "string"
		if nullable {
			nullableTypeReturn = "pgx.NullString"
		}

	case "numeric":
		typeReturn = "string"
		if nullable {
			nullableTypeReturn = "pgx.NullString"
		}

	case "uuid":
		typeReturn = "string"
		if nullable {
			nullableTypeReturn = "pgx.NullString"
		}

	case "bigint", "bigserial":
		typeReturn = "int64"
		if nullable {
			nullableTypeReturn = "pgx.NullInt64"
		}

	case "timestamp with time zone", "timestamp without time zone":

		typeReturn = "time.Time"
		goTypeToImport = "time"

		if nullable {
			nullableTypeReturn = "pgx.NullTime"
		}
	}

	return typeReturn, nullableTypeReturn, goTypeToImport
}

func GetGoTypeNullableType(goType string) string {

	switch goType {

	case "bool":
		return "pgx.NullBool"
	case "int32", "serial":
		return "pgx.NullInt32"
	case "int64", "bigserial":
		return "pgx.NullInt64"
	case "string":
		return "pgx.NullString"
	case "time.Time":
		return "pgx.NullTime"
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
