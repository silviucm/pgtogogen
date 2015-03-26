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
