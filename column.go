package main

import (
	"database/sql"
	"log"
	"strings"
)

/* Column Section */

type Column struct {
	Options  *ToolOptions
	DbHandle *sql.DB

	Name         string
	Type         string
	MaxLength    int
	DefaultValue sql.NullString
	Nullable     bool

	IsPK          bool
	IsCompositePK bool

	IsFK bool

	GoName string
	GoType string

	ColumnComment string
}

/* Util methods */

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

func GetGoTypeForColumn(udtType string) (typeReturn string, goTypeToImport string) {

	typeReturn = ""
	goTypeToImport = ""

	switch udtType {
	case "character varying":
		typeReturn = "string"
	case "integer":
		typeReturn = "int"
	case "boolean":
		typeReturn = "bool"
	case "uuid":
		typeReturn = "string"
	case "biging":
		typeReturn = "int64"
	case "timestamp with time zone":
		typeReturn = "time.Time"
		goTypeToImport = "time"
	}

	return typeReturn, goTypeToImport
}

func DecodeNullable(isNullable string) bool {

	if isNullable == "NO" {
		return false
	}

	if isNullable == "YES" || isNullable == "Yes" || isNullable == "yes" {
		return true
	}

	return false
}

func DecodeMaxLength(maxLength sql.NullInt64) int {

	if maxLength.Valid == false {
		return -1
	}

	return int(maxLength.Int64)
}
