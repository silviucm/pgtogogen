package main

/* Insert Functions Templates */

const TABLE_STATIC_INSERT_TEMPLATE = `{{$colCount := len .Columns}}{{$pkColCount := len .PKColumns}}
{{$functionName := "Insert"}}{{$sourceStructName := print "source" .GoFriendlyName}}
// Inserts a new row into the {{.TableName}} table, using the values
// inside the pointer to a {{.GoFriendlyName}} structure passed to it.
// Returns back the pointer to the structure with all the fields, including the PK fields.
// If operation fails, it returns nil and the error
func (util *t{{.GoFriendlyName}}Utils) {{$functionName}}({{$sourceStructName}} *{{.GoFriendlyName}}) (*{{.GoFriendlyName}},  error) {
						
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "

	if source{{.GoFriendlyName}} == nil {
		return nil, NewModelsErrorLocal(errorPrefix, "the source{{.GoFriendlyName}} pointer is nil")
	}
	
	currentDbHandle := GetDb()
	if currentDbHandle == nil {
		return nil, NewModelsErrorLocal(errorPrefix, "the database handle is nil")
	}

	// define returning PK params for the insert query row execution
	{{range .PKColumns}}var param{{.GoName}} {{.GoType}}
	{{end}}

	// define the select query
	var insertQueryAllColumns = "{{.GenericInsertQuery}} RETURNING {{.PKColumnsString}}";
	var insertQueryNoPKColumns = "{{.GenericInsertQueryNoPK}} RETURNING {{.PKColumnsString}}";
	
	var query string = insertQueryAllColumns
	
	if {{$sourceStructName}}.PgToGo_IgnorePKValuesWhenInsertingAndUseSequence {
		query = insertQueryNoPKColumns
	}

	// pq does not support the LastInsertId() method of the Result type in database/sql. 
	// To return the identifier of an INSERT (or UPDATE or DELETE), use the Postgres RETURNING clause 
	// with a standard Query or QueryRow call
	
	var err error

	// define the values to be passed, from the structure
	var  {{.ColumnsString}} = {{range $i, $e := .Columns}}{{$sourceStructName}}.{{$e.GoName}}{{if ne (plus1 $i) $colCount}},{{end}}{{end}}
	
	// this will print only if debug mode enabled
	Debug("Insert Query:", query)
	
	if {{$sourceStructName}}.PgToGo_IgnorePKValuesWhenInsertingAndUseSequence {
		err = currentDbHandle.QueryRow(query, {{.ColumnsStringNoPK}}).Scan({{range $i, $e := .PKColumns}}&param{{.GoName}}{{if ne (plus1 $i) $pkColCount}},{{end}}{{end}})		
	} else {
		err = currentDbHandle.QueryRow(query, {{.ColumnsString}}).Scan({{range $i, $e := .PKColumns}}&param{{.GoName}}{{if ne (plus1 $i) $pkColCount}},{{end}}{{end}})
	}
		
    switch {
    case err == ErrNoRows:
            // no such row found, return nil and nil
			return nil, nil
    case err != nil:
            return nil, NewModelsError(errorPrefix + "fatal error running the query:",err)
    default:
           	// populate the returning ids inside the returnStructure pointer
			{{range .PKColumns}}{{$sourceStructName}}.{{.GoName}} = param{{.GoName}}
			{{end}}

			// return the structure
			return {{$sourceStructName}}, nil
    }			
}
`
