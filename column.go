package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"text/template"
)

/* Column Section */

type Column struct {
	Options     *ToolOptions
	DbHandle    *sql.DB
	ParentTable *Table

	Name         string
	Type         string
	MaxLength    int
	DefaultValue sql.NullString
	Nullable     bool
	IsSequence   bool

	IsPK          bool
	IsCompositePK bool

	IsFK bool

	GoName string
	GoType string

	ColumnComment string
}

func (col *Column) GeneratePKGetter(parentTable *Table) []byte {

	col.ParentTable = parentTable

	var fns = template.FuncMap{
		"plus1": func(x int) int {
			return x + 1
		},
	}

	tmpl, err := template.New("pkGetterTemplate").Funcs(fns).Parse(PK_GETTER_TEMPLATE)
	if err != nil {
		log.Fatal("GeneratePKGetter() fatal error running template.New:", err)
	}

	var generatedTemplate bytes.Buffer
	err = tmpl.Execute(&generatedTemplate, col)
	if err != nil {
		log.Fatal("GeneratePKGetter() fatal error running template.Execute:", err)
	}

	fmt.Println("PK Getter structure for column " + col.GoName + " generated.")
	return generatedTemplate.Bytes()

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

func GetGoTypeForColumn(columnType string) (typeReturn string, goTypeToImport string) {

	typeReturn = ""
	goTypeToImport = ""

	switch columnType {
	case "character varying":
		typeReturn = "string"
	case "integer":
		typeReturn = "int"
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

func DecodeIsColumnSequence(columnDefaultValue sql.NullString) bool {

	if columnDefaultValue.Valid == false {
		return false
	}

	if strings.HasPrefix(columnDefaultValue.String, "nextval(") {
		return true
	}

	return false
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
