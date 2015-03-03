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
	DefaultValue sql.NullString
	Nullable     bool

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

func GetGoTypeForColumn(udtType string) string {

	var correspondingGoType = ""

	switch udtType {
	case "varchar":
		return "string"
	case "int4":
		return "int"
	}

	return correspondingGoType
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
