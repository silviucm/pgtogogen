package main

const TABLE_TEMPLATE = `package main

import (
	"database/sql"
	_ "github.com/lib/pq"
	{{range $key, $value := .GoTypesToImport}}"{{$value}}"
	{{end}}	
)

const {{.GoFriendlyName}}_DB_TABLE_NAME string = "{{.TableName}}"

type {{.GoFriendlyName}} struct {
	{{range .Columns}}{{.GoName}} {{.GoType}}
	{{end}}	
}`
