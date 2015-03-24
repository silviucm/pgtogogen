package main

/* Insert Functions Templates */

const TABLE_STATIC_INSERT_TEMPLATE_ATOMIC = `{{$colCount := len .Columns}}{{$pkColCount := len .PKColumns}}
{{$functionName := "Insert"}}{{$sourceStructName := print "source" .GoFriendlyName}}
// Inserts a new row into the {{.DbName}} table, using the values
// inside the pointer to a {{.GoFriendlyName}} structure passed to it.
// Returns back the pointer to the structure with all the fields, including the PK fields.
// If operation fails, it returns nil and the error
func (utilRef *t{{.GoFriendlyName}}Utils) {{$functionName}}({{$sourceStructName}} *{{.GoFriendlyName}}) (*{{.GoFriendlyName}},  error) {
						
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

	if {{$sourceStructName}}.PgToGo_SetDateTimeFieldsToNowForNewRecords {
		{{range $i, $e := .Columns}}{{if eq .GoType "time.Time"}}{{$sourceStructName}}.{{$e.GoName}}=Now(){{end}}{{end}}
	}

	if {{$sourceStructName}}.PgToGo_SetGuidFieldsToNewGuidsNewRecords {
		{{range $i, $e := .Columns}}{{if .IsGuid }}{{$sourceStructName}}.{{$e.GoName}}=NewGuid(){{end}}{{end}}
	}

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
			{{if gt (len .UniqueConstraints) 0}}if Contains(err.Error(),"SQLSTATE 23505") {
			{{range $e := .UniqueConstraints}}	if Contains(err.Error(),"{{$e.DbName}}") { return nil,Err{{$e.ParentTable.GoFriendlyName}}_UQ_{{$e.DbName}}	}				
			{{end}}
			} {{end}}
            return nil, NewModelsError(errorPrefix + "fatal error running the query:",err)
    default:
           	// populate the returning ids inside the returnStructure pointer
			{{range .PKColumns}}{{$sourceStructName}}.{{.GoName}} = param{{.GoName}}
			{{end}}

			// return the structure
			return {{$sourceStructName}}, nil
    }			
}

{{$functionName := "Insert"}}{{$sourceInstanceStructName := print "source" .GoFriendlyName}}
// Inserts a new row into the {{.DbName}} table, corresponding to the provided {{$sourceInstanceStructName}}
// Returns back the pointer to the structure with all the fields, including the PK fields.
// If operation fails, it returns nil and the error
func ({{$sourceInstanceStructName}} *{{.GoFriendlyName}}) {{$functionName}}() (*{{.GoFriendlyName}},  error) {
	
	return Tables.{{.GoFriendlyName}}.Insert({{$sourceInstanceStructName}})
}
`

const TABLE_STATIC_INSERT_TEMPLATE_TX = `{{$colCount := len .Columns}}{{$pkColCount := len .PKColumns}}
{{$functionName := print "Insert" .GoFriendlyName}}{{$sourceStructName := print "source" .GoFriendlyName}}
// Inserts a new row into the {{.DbName}} table, within the supplied transaction wrapper,
// using the pointer to a {{.GoFriendlyName}} structure passed to it.
// Returns back the pointer to the structure with all the fields, including the PK fields.
// If operation fails, it returns nil and the error. It does not rollback the transaction itself.
func (txWrapper *Transaction) {{$functionName}}({{$sourceStructName}} *{{.GoFriendlyName}}) (*{{.GoFriendlyName}},  error) {
						
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "

	if source{{.GoFriendlyName}} == nil {
		return nil, NewModelsErrorLocal(errorPrefix, "the source{{.GoFriendlyName}} pointer is nil")
	}
	
	if txWrapper == nil { return nil, NewModelsErrorLocal(errorPrefix, "the transaction wrapper is nil") }
	if txWrapper.Tx == nil { return nil, NewModelsErrorLocal(errorPrefix, "the transaction object is nil") }

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

	if {{$sourceStructName}}.PgToGo_SetDateTimeFieldsToNowForNewRecords {
		{{range $i, $e := .Columns}}{{if eq .GoType "time.Time"}}{{$sourceStructName}}.{{$e.GoName}}=Now(){{end}}{{end}}
	}

	if {{$sourceStructName}}.PgToGo_SetGuidFieldsToNewGuidsNewRecords {
		{{range $i, $e := .Columns}}{{if .IsGuid }}{{$sourceStructName}}.{{$e.GoName}}=NewGuid(){{end}}{{end}}
	}

	// define the values to be passed, from the structure
	var  {{.ColumnsString}} = {{range $i, $e := .Columns}}{{$sourceStructName}}.{{$e.GoName}}{{if ne (plus1 $i) $colCount}},{{end}}{{end}}
	
	// this will print only if debug mode enabled
	Debug("Insert Query:", query)
	
	if {{$sourceStructName}}.PgToGo_IgnorePKValuesWhenInsertingAndUseSequence {
		err = txWrapper.Tx.QueryRow(query, {{.ColumnsStringNoPK}}).Scan({{range $i, $e := .PKColumns}}&param{{.GoName}}{{if ne (plus1 $i) $pkColCount}},{{end}}{{end}})		
	} else {
		err = txWrapper.Tx.QueryRow(query, {{.ColumnsString}}).Scan({{range $i, $e := .PKColumns}}&param{{.GoName}}{{if ne (plus1 $i) $pkColCount}},{{end}}{{end}})
	}
		
    switch {
    case err == ErrNoRows:
            // no such row found, return nil and nil
			return nil, nil
    case err != nil:
			{{if gt (len .UniqueConstraints) 0}}if Contains(err.Error(),"SQLSTATE 23505") {
			{{range $e := .UniqueConstraints}}	if Contains(err.Error(),"{{$e.DbName}}") { return nil,Err{{$e.ParentTable.GoFriendlyName}}_UQ_{{$e.DbName}}	}				
			{{end}}
			} {{end}}	
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
