package main

/* Count, Single, First, Last Functions Templates */

/* ****************************************************** */
/* BEGIN: Atomic (non-transaction) Select Extra Templates */
/* ****************************************************** */

const SELECT_TEMPLATE_COUNT = `{{$functionName := "Count"}}
// Returns the number of rows from {{.DbName}}
// This version is accurate, but can be slow. For a faster version, user CountImprecise.
// If an error occures, it returns -1 and the error.
func (utilRef *t{{.GoFriendlyName}}Utils) {{$functionName}}() (int64,  error) {
						
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "	
	
	currentDbHandle := GetDb()
	if currentDbHandle == nil {
		return -1, NewModelsErrorLocal(errorPrefix, "the database handle is nil")
	}

	// define the select query
	var query string = "SELECT COUNT(*) FROM {{.DbName}}"
	var totalRows int64	

	err := currentDbHandle.QueryRow(query).Scan(&totalRows)
	if err != nil {
		return -1, NewModelsError(errorPrefix + " error during QueryRow() or Scan():", err)
	}

	return totalRows, nil
}	

{{$functionName := "CountImprecise"}}
// Returns the number of rows from {{.DbName}}
// This version is less accurate, but much faster. It depends on the table being
// vacuum-analyzed regularly. With autovacuum results are quite accurate
func (utilRef *t{{.GoFriendlyName}}Utils) {{$functionName}}() (int64,  error) {
						
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "	
	
	currentDbHandle := GetDb()
	if currentDbHandle == nil {
		return -1, NewModelsErrorLocal(errorPrefix, "the database handle is nil")
	}

	// define the select query
	var query string = "SELECT reltuples FROM pg_class WHERE oid = '{{.Options.DbSchema}}.{{.DbName}}'::regclass;"
	
	// the reltuples is real (oid 700) so we need to retrieve it using a float32 value
	var totalRows float32
	
	err := currentDbHandle.QueryRow(query).Scan(&totalRows)
	if err != nil {
		return -1, NewModelsError(errorPrefix + " error during QueryRow() or Scan():", err)
	}
	
	return int64(totalRows), nil
}	
`

/* BEGIN: Single Templates Section */

const CONST_SELECT_TEMPLATE_SINGLE = `{{$colCount := len .Columns}}
// Returns the a single record from {{.DbName}} based on the specified condition.
// If no record is found, nil is returned. If more than one record is found, an 
// If an error occures, the function returns nil and the error.
func {{if eq $utilOrTransactionDbHandle "currentDbHandle"}}(utilRef *t{{.GoFriendlyName}}Utils){{else}}(txWrapper *Transaction){{end}}` +
	` {{$functionName}}(condition string, params ...interface{}) (*{{.GoFriendlyName}},  error) {
						
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "	
	
	{{if eq $utilOrTransactionDbHandle "currentDbHandle"}}
	currentDbHandle := GetDb()
	if currentDbHandle == nil {
		return nil, NewModelsErrorLocal(errorPrefix, "the database handle is nil")
	} 
	{{else}}
	if txWrapper == nil { return nil, NewModelsErrorLocal(errorPrefix, "the transaction wrapper is nil") }
	if txWrapper.Tx == nil { return nil, NewModelsErrorLocal(errorPrefix, "the transaction object is nil") }
	{{end}}

	// define the select query
	var queryParts []string
	
	queryParts = append(queryParts, "{{.GenericSelectQuery}} WHERE ")
	queryParts = append(queryParts, condition)
		
	rows, err := {{$utilOrTransactionDbHandle}}.Query(JoinStringParts(queryParts,""), params...)
	
	if err != nil {
		return nil, NewModelsError(errorPrefix + " fatal error running the query:", err)
	}
	defer rows.Close()

	var instanceOf{{.GoFriendlyName}} *{{.GoFriendlyName}}

	// BEGIN: if any nullable fields, create temporary nullable variables to receive null values
	{{range $i, $e := .Columns}}{{if .Nullable}}var nullable{{$e.GoName}} {{$e.GoNullableType}} 
	{{end}}{{end}}
	// END: if any nullable fields, create temporary nullable variables to receive null values

	var iteration int = 0

	for rows.Next() {

		if iteration > 0 {
			return nil, ErrTooManyRows	
		}

		// create a new instance of {{.GoFriendlyName}}  {{$instanceVarName := print "current" .GoFriendlyName}}
		
		{{$instanceVarName}} := {{.GoFriendlyName}}{}

		err := rows.Scan({{range $i, $e := .Columns}}{{if .Nullable}}&nullable{{$e.GoName}}{{else}}&{{$instanceVarName}}.{{$e.GoName}}{{end}}{{if ne (plus1 $i) $colCount}},{{end}}{{end}})
		if err != nil {
			return nil, NewModelsError(errorPrefix + " error during rows.Scan():", err)
		}
		
		// BEGIN: assign any nullable values to the nullable fields inside the struct appropriately
		{{range $i, $e := .Columns}}{{if .Nullable}} {{$instanceVarName}}.Set{{.GoName}}(nullable{{$e.GoName}}.GetValue(), nullable{{$e.GoName}}.Valid)
		{{end}}{{end}}
		// END: assign any nullable values to the nullable fields inside the struct appropriately			
		
		instanceOf{{.GoFriendlyName}} = &{{$instanceVarName}}
		iteration = iteration + 1

	}

	err = rows.Err()
	if err != nil {
		return nil, NewModelsError(errorPrefix + " error during single row fetching:", err)
	}

	return instanceOf{{.GoFriendlyName}}, nil
}`

const SELECT_TEMPLATE_SINGLE_ATOMIC = `{{$utilOrTransactionDbHandle := "currentDbHandle"}}{{$functionName := "Single"}}` + CONST_SELECT_TEMPLATE_SINGLE
const SELECT_TEMPLATE_SINGLE_TX = `{{$utilOrTransactionDbHandle := "txWrapper.Tx"}}{{$functionName :=  print "Single" .GoFriendlyName}}` + CONST_SELECT_TEMPLATE_SINGLE
