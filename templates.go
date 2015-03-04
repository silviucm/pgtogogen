package main

/* Base */

const BASE_TEMPLATE = `package {{.PackageName}}

import (
	"database/sql"
	_ "github.com/lib/pq"	
	"errors"
)

var dbHandle *sql.DB

func GetDb() *sql.DB {
	return dbHandle
}

func NewModelsError(errorMsg string) error {
	return errors.New(errorMsg)
}
`

/* Tables */

const TABLE_TEMPLATE = `package {{.Options.PackageName}}

import (
	{{range $key, $value := .GoTypesToImport}}"{{$value}}"
	{{end}}	
)

const {{.GoFriendlyName}}_DB_TABLE_NAME string = "{{.TableName}}"

type {{.GoFriendlyName}} struct {
	{{range .Columns}}{{.GoName}} {{.GoType}} // IsPK: {{.IsPK}} , IsCompositePK: {{.IsCompositePK}}, IsFK: {{.IsFK}}
	{{end}}	
}`

/* Columns */

const PK_GETTER_TEMPLATE = `

func {{.ParentTable.GoFriendlyName}}GetBy{{.GoName}}(param{{.GoName}} {{.GoType}}) ({{.ParentTable.GoFriendlyName}}, error) {
	
	if GetDb() == nil {
		return NewModelsError("{{.ParentTable.GoFriendlyName}}GetBy{{.GoName}}() ERROR: the database handle is nil")
	}
}
`
