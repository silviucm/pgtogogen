package main

import "text/template"

/* Template helper functions */
var fns = template.FuncMap{
	"plus1": func(x int) int {
		return x + 1
	},
}

/* Base Templates */

const BASE_TEMPLATE = `package {{.PackageName}}

/* *********************************************************** */
/* This file was automatically generated by pgtogogen.         */
/* Do not modify this file unless you know what you are doing. */
/* *********************************************************** */

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
`

/* Tables */

const TABLE_TEMPLATE = `package {{.Options.PackageName}}

/* *********************************************************** */
/* This file was automatically generated by pgtogogen.         */
/* Do not modify this file unless you know what you are doing. */
/* *********************************************************** */

import (
	{{range $key, $value := .GoTypesToImport}}"{{$value}}"
	{{end}}	
)

const {{.GoFriendlyName}}_DB_TABLE_NAME string = "{{.TableName}}"

type {{.GoFriendlyName}} struct {
	{{range .Columns}}{{.GoName}} {{.GoType}} // IsPK: {{.IsPK}} , IsCompositePK: {{.IsCompositePK}}, IsFK: {{.IsFK}}
	{{end}}	
}`

const TABLE_TEMPLATE_CUSTOM = `package {{.Options.PackageName}}

/* *********************************************************** **/
/* This file is generated by pgtogogen FIRST-TIME ONLY.         */
/* It will not subsequently overwrite it if it already exists.  */
/* Use this file to create your custom extension functionality. */
/* ************************************************************ */

/*
import (
	{{range $key, $value := .GoTypesToImport}}"{{$value}}"
	{{end}}	
)
*/

`

/* Insert, Update Methods */

const TABLE_INSERT_TEMPLATE = `{{$colCount := len .Columns}}{{$pkColCount := len .PKColumns}}{{$functionName := print .GoFriendlyName "Insert"}}
// Inserts a new row into the {{.TableName}} table, using the values
// inside the pointer to a {{.GoFriendlyName}} structure passed to it.
// Returns back the pointer to the structure with all the fields, including the PK fields.
// If operation fails, it returns nil and the error
func {{$functionName}}(new{{.GoFriendlyName}} *{{.GoFriendlyName}}) (returnStruct *{{.GoFriendlyName}}, err error) {
	
	returnStruct = nil
	err = nil
	
	var errorPrefix = "{{$functionName}}() ERROR: "
	
	currentDbHandle := GetDb()
	if currentDbHandle == nil {
		return nil, NewModelsError(errorPrefix + "the database handle is nil")
	}

	// define returning PK params for the insert query row execution
	{{range .PKColumns}}var param{{.GoName}} {{.GoType}}
	{{end}}

	// define the select query
	var query = "{{.GenericInsertQuery}} RETURNING {{.PKColumnsString}}";

	// pq does not support the LastInsertId() method of the Result type in database/sql. 
	// To return the identifier of an INSERT (or UPDATE or DELETE), use the Postgres RETURNING clause 
	// with a standard Query or QueryRow call
	err = currentDbHandle.QueryRow(query, {{range $i, $e := .Columns}}returnStruct.{{.GoName}}{{if ne (plus1 $i) $colCount}},{{end}}{{end}}).Scan({{range $i, $e := .PKColumns}}&param{{.GoName}}{{if ne (plus1 $i) $pkColCount}},{{end}}{{end}})
    switch {
    case err == sql.ErrNoRows:
            // no such row found, return nil and nil
			return nil, nil
    case err != nil:
            return nil, NewModelsError(errorPrefix + "fatal error running the query:" + err.Error())
    default:
           	// populate the returning ids inside the returnStructure pointer
			{{range .PKColumns}}returnStruct.{{.GoName}} = param{{.GoName}}
			{{end}}

			// return the structure
			return returnStruct, nil
    }			
}
`

/* Columns */

const PK_GETTER_TEMPLATE = `{{$colCount := len .ParentTable.Columns}}
// Queries the database for a single row based on the specified {{.GoName}} value.
// Returns a pointer to a {{.ParentTable.GoFriendlyName}} structure if a record was found,
// otherwise it returns nil.
func {{.ParentTable.GoFriendlyName}}GetBy{{.GoName}}(inputParam{{.GoName}} {{.GoType}}) (returnStruct *{{.ParentTable.GoFriendlyName}}, err error) {
	
	returnStruct = nil
	err = nil
	
	var errorPrefix = "{{.ParentTable.GoFriendlyName}}GetBy{{.GoName}}() ERROR: "
	
	currentDbHandle := GetDb()
	if currentDbHandle == nil {
		return nil, NewModelsError(errorPrefix + "the database handle is nil")
	}

	// define receiving params for the row iteration
	{{range .ParentTable.Columns}}var param{{.GoName}} {{.GoType}}
	{{end}}

	// define the select query
	var query = "{{.ParentTable.GenericSelectQuery}} WHERE {{.Name}} = $1";

	// we are aiming for a single row so we will use Query Row	
	err = currentDbHandle.QueryRow(query, inputParam{{.GoName}}).Scan({{range $i, $e := .ParentTable.Columns}}&param{{$e.GoName}}{{if ne (plus1 $i) $colCount}},{{end}}{{end}})
    switch {
    case err == sql.ErrNoRows:
            // no such row found, return nil and nil
			return nil, nil
    case err != nil:
            return nil, NewModelsError(errorPrefix + "fatal error running the query:" + err.Error())
    default:
           	// create the return structure as a pointer of the type
			returnStruct = &{{.ParentTable.GoFriendlyName}}{
				{{range .ParentTable.Columns}}{{.GoName}}: param{{.GoName}},
				{{end}}
			}
			// return the structure
			return returnStruct, nil
    }			
}
`

const PK_SELECT_TEMPLATE = `{{$colCount := len .ParentTable.Columns}}
func {{.ParentTable.GoFriendlyName}}GetBy{{.GoName}}(inputParam{{.GoName}} {{.GoType}}) (returnStruct *{{.ParentTable.GoFriendlyName}}, err error) {
	
	returnStruct = nil
	err = nil
	
	var errorPrefix = "{{.ParentTable.GoFriendlyName}}GetBy{{.GoName}}() ERROR: "
	
	currentDbHandle := GetDb()
	if currentDbHandle == nil {
		return nil, NewModelsError(errorPrefix + "the database handle is nil")
	}

	// define receiving params for the row iteration
	{{range .ParentTable.Columns}}var param{{.GoName}} {{.GoType}}
	{{end}}

	// define the select query
	var query = "{{.ParentTable.GenericSelectQuery}} FROM {{.ParentTable.TableName}} WHERE {{.Name}} = $1";

	rows, err := currentDbHandle.Query(query, inputParam{{.GoName}})

	if err != nil {
		return nil, NewModelsError(errorPrefix + "fatal error running the query:" + err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan({{range $i, $e := .ParentTable.Columns}}&param{{$e.GoName}}{{if ne (plus1 $i) $colCount}},{{end}}{{end}})
		if err != nil {
			return nil, NewModelsError(errorPrefix + "fatal error scanning the fields in the current row:" + err.Error())
		}		

		// create the return structure as a pointer of the type
		returnStruct = &{{.ParentTable.GoFriendlyName}}{
			{{range .ParentTable.Columns}}{{.GoName}}: param{{.GoName}},
			{{end}}
		}		

	}
	err = rows.Err()
	if err != nil {
		return nil, NewModelsError(errorPrefix + "fatal generic rows error:" + err.Error())
	}
	
	// return the structure
	return returnStruct, err	
}
`
