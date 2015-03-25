package main

/* Select Functions Templates */

/* Templates for the Where method */

const SELECT_TEMPLATE_WHERE = `{{$colCount := len .Columns}}
{{$functionName := "SelectFrom"}}{{$sourceStructName := print "source" .GoFriendlyName}}
// Returns the rows from {{.DbName}}, corresponding to the supplied condition 
// and the respective parameters. The condition must not include the WHERE keyword.
// The cacheOption parameter is one of the FLAG_CACHE_[behaviour] global integer constants.
// FLAG_CACHE_DISABLE has a value of 0, and caching is completely bypassed.
// The rows are converted to a slice of {{.GoFriendlyName}} instances
// If operation fails, it returns nil and the error.
func (utilRef *t{{.GoFriendlyName}}Utils) {{$functionName}}(cacheOption int, condition string, params ...interface{}) ([]{{.GoFriendlyName}},  error) {
						
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "

	if condition == "" {
		return nil, NewModelsErrorLocal(errorPrefix, "No condition specified. Please use SelectAll method to select all rows from {{.DbName}}")
	}
	
	var whereClauseHash string = ""
	var hashErr error = nil	

	if cacheOption > FLAG_CACHE_DISABLE {
		// check the caching options - in case it's FLAG_CACHE_USE and there is cache available no need to go further
		if cacheOption == FLAG_CACHE_USE {
			whereClauseHash, hashErr = GetHashFromConditionAndParams(condition, params...)
			if hashErr != nil {
				return nil, NewModelsError(errorPrefix + "GetHashFromConditionAndParams() error:",hashErr)
			}
			// try to get the rows from cache, if enabled and valid
			if current{{.GoFriendlyName}}RowsFromCache, cacheValid := utilRef.Cache.GetWhere(whereClauseHash) ; cacheValid == true {					
				return current{{.GoFriendlyName}}RowsFromCache, nil
			} 
		}
	}
	
	currentDbHandle := GetDb()
	if currentDbHandle == nil {
		return nil, NewModelsErrorLocal(errorPrefix, "the database handle is nil")
	}

	// define the delete query
	queryBuffer := bytes.Buffer{}
	_, writeErr := queryBuffer.WriteString("{{.GenericSelectQuery}} WHERE ")
	if writeErr != nil {
		return nil, NewModelsError(errorPrefix + "queryBuffer.WriteString error:",writeErr)
	}

	_, writeErr = queryBuffer.WriteString(condition)
	if writeErr != nil {
		return nil, NewModelsError(errorPrefix + "queryBuffer.WriteString (condition param) error:",writeErr)
	}	
	
	var sliceOf{{.GoFriendlyName}} []{{.GoFriendlyName}}
	rows, err := currentDbHandle.Query(queryBuffer.String(), params...)

	if err != nil {
		return nil, NewModelsError(errorPrefix + " fatal error running the query:", err)
	}
	defer rows.Close()

	for rows.Next() {

		// create a new instance of {{.GoFriendlyName}}  {{$instanceVarName := print "current" .GoFriendlyName}}
		
		{{$instanceVarName}} := {{.GoFriendlyName}}{}

		err := rows.Scan({{range $i, $e := .Columns}}&{{$instanceVarName}}.{{$e.GoName}}{{if ne (plus1 $i) $colCount}},{{end}}{{end}})
		if err != nil {
			return nil, NewModelsError(errorPrefix + " error during rows.Scan():", err)
		}
		
		sliceOf{{.GoFriendlyName}} = append(sliceOf{{.GoFriendlyName}}, current{{.GoFriendlyName}})

	}
	err = rows.Err()
	if err != nil {
		return nil, NewModelsError(errorPrefix + " error during rows.Next() iterations:", err)
	}	
	
	// before returning the result, make sure to insert it into cache if instructed
	if cacheOption > FLAG_CACHE_DISABLE {
		
		if cacheOption != FLAG_CACHE_DELETE {			
			utilRef.Cache.SetWhere(whereClauseHash, sliceOf{{.GoFriendlyName}})	
		}
	}	
	
	return sliceOf{{.GoFriendlyName}}, nil
}
`

const SELECT_TEMPLATE_WHERE_TX = `{{$colCount := len .Columns}}
{{$functionName := print "SelectFrom" .GoFriendlyName}}{{$sourceStructName := print "source" .GoFriendlyName}}
// Returns the rows from {{.DbName}}, corresponding to the supplied condition 
// and the respective parameters. The condition must not include the WHERE keyword.
// The cacheOption parameter is one of the FLAG_CACHE_[behaviour] global integer constants.
// FLAG_CACHE_DISABLE has a value of 0, and caching is completely bypassed.
// The rows are converted to a slice of {{.GoFriendlyName}} instances
// If operation fails, it returns nil and the error.
func (txWrapper *Transaction) {{$functionName}}(cacheOption int, condition string, params ...interface{}) ([]{{.GoFriendlyName}},  error) {
						
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "

	if condition == "" {
		return nil, NewModelsErrorLocal(errorPrefix, "No condition specified. Please use SelectAll method to select all rows from {{.DbName}}")
	}

	var whereClauseHash string = ""
	var hashErr error = nil	

	if cacheOption > FLAG_CACHE_DISABLE {
		// check the caching options - in case it's FLAG_CACHE_USE and there is cache available no need to go further
		if cacheOption == FLAG_CACHE_USE {
			whereClauseHash, hashErr = GetHashFromConditionAndParams(condition, params...)
			if hashErr != nil {
				return nil, NewModelsError(errorPrefix + "GetHashFromConditionAndParams() error:",hashErr)
			}
			// try to get the rows from cache, if enabled and valid
			if current{{.GoFriendlyName}}RowsFromCache, cacheValid := {{if .IsTable}}Tables{{else}}Views{{end}}.{{.GoFriendlyName}}.Cache.GetWhere(whereClauseHash) ; cacheValid == true {					
				return current{{.GoFriendlyName}}RowsFromCache, nil
			} 
		}
	}
	
	if txWrapper == nil { return nil, NewModelsErrorLocal(errorPrefix, "the transaction wrapper is nil") }
	if txWrapper.Tx == nil { return nil, NewModelsErrorLocal(errorPrefix, "the transaction object is nil") }	

	// define the delete query
	queryBuffer := bytes.Buffer{}
	_, writeErr := queryBuffer.WriteString("{{.GenericSelectQuery}} WHERE ")
	if writeErr != nil {
		return nil, NewModelsError(errorPrefix + "queryBuffer.WriteString error:",writeErr)
	}

	_, writeErr = queryBuffer.WriteString(condition)
	if writeErr != nil {
		return nil, NewModelsError(errorPrefix + "queryBuffer.WriteString (condition param) error:",writeErr)
	}	
	
	var sliceOf{{.GoFriendlyName}} []{{.GoFriendlyName}}
	rows, err := txWrapper.Tx.Query(queryBuffer.String(), params...)

	if err != nil {
		return nil, NewModelsError(errorPrefix + " fatal error running the query:", err)
	}
	defer rows.Close()

	for rows.Next() {

		// create a new instance of {{.GoFriendlyName}}  {{$instanceVarName := print "current" .GoFriendlyName}}
		
		{{$instanceVarName}} := {{.GoFriendlyName}}{}

		err := rows.Scan({{range $i, $e := .Columns}}&{{$instanceVarName}}.{{$e.GoName}}{{if ne (plus1 $i) $colCount}},{{end}}{{end}})
		if err != nil {
			return nil, NewModelsError(errorPrefix + " error during rows.Scan():", err)
		}
		
		sliceOf{{.GoFriendlyName}} = append(sliceOf{{.GoFriendlyName}}, current{{.GoFriendlyName}})

	}
	err = rows.Err()
	if err != nil {
		return nil, NewModelsError(errorPrefix + " error during rows.Next() iterations:", err)
	}

	// before returning the result, make sure to insert it into cache if instructed
	if cacheOption > FLAG_CACHE_DISABLE {
		
		if cacheOption != FLAG_CACHE_DELETE {			
			{{if .IsTable}}Tables{{else}}Views{{end}}.{{.GoFriendlyName}}.Cache.SetWhere(whereClauseHash, sliceOf{{.GoFriendlyName}})	
		}
	}		
	
	return sliceOf{{.GoFriendlyName}}, nil
}
`

/* Templates for the SelectAll method */

const SELECT_TEMPLATE_ALL = `{{$colCount := len .Columns}}
{{$functionName := "SelectAll"}}{{$sourceStructName := print "source" .GoFriendlyName}}
// Returns all the rows from {{.DbName}}.
// The rows are converted to a slice of {{.GoFriendlyName}} instances
// If operation fails, it returns nil and the error.
func (utilRef *t{{.GoFriendlyName}}Utils) {{$functionName}}() ([]{{.GoFriendlyName}},  error) {
						
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "

	currentDbHandle := GetDb()
	if currentDbHandle == nil {
		return nil, NewModelsErrorLocal(errorPrefix, "the database handle is nil")
	}

	// try to get the rows from cache, if enabled and valid
	if all{{.GoFriendlyName}}RowsFromCache, cacheValid := utilRef.Cache.GetAllRows() ; cacheValid == true {		
		return all{{.GoFriendlyName}}RowsFromCache, nil
	}

	var sliceOf{{.GoFriendlyName}} []{{.GoFriendlyName}}
	rows, err := currentDbHandle.Query("{{.GenericSelectQuery}}")

	if err != nil {
		return nil, NewModelsError(errorPrefix + " fatal error running the query:", err)
	}
	defer rows.Close()

	for rows.Next() {

		// create a new instance of {{.GoFriendlyName}}  {{$instanceVarName := print "current" .GoFriendlyName}}
		
		{{$instanceVarName}} := {{.GoFriendlyName}}{}

		err := rows.Scan({{range $i, $e := .Columns}}&{{$instanceVarName}}.{{$e.GoName}}{{if ne (plus1 $i) $colCount}},{{end}}{{end}})
		if err != nil {
			return nil, NewModelsError(errorPrefix + " error during rows.Scan():", err)
		}
		
		sliceOf{{.GoFriendlyName}} = append(sliceOf{{.GoFriendlyName}}, current{{.GoFriendlyName}})

	}
	err = rows.Err()
	if err != nil {
		return nil, NewModelsError(errorPrefix + " error during rows.Next() iterations:", err)
	}	
	
	return sliceOf{{.GoFriendlyName}}, nil
}
`

const SELECT_TEMPLATE_ALL_TX = `{{$colCount := len .Columns}}
{{$functionName := print "SelectAll" .GoFriendlyName}}{{$sourceStructName := print "source" .GoFriendlyName}}
// Returns all the rows from {{.DbName}}.
// The rows are converted to a slice of {{.GoFriendlyName}} instances
// If operation fails, it returns nil and the error.
func (txWrapper *Transaction) {{$functionName}}() ([]{{.GoFriendlyName}},  error) {
						
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "

	if txWrapper == nil { return nil, NewModelsErrorLocal(errorPrefix, "the transaction wrapper is nil") }
	if txWrapper.Tx == nil { return nil, NewModelsErrorLocal(errorPrefix, "the transaction object is nil") }	

	// try to get the rows from cache, if enabled and valid
	if all{{.GoFriendlyName}}RowsFromCache, cacheValid := {{if .IsTable}}Tables{{else}}Views{{end}}.{{.GoFriendlyName}}.Cache.GetAllRows() ; cacheValid == true {		
		return all{{.GoFriendlyName}}RowsFromCache, nil
	}

	var sliceOf{{.GoFriendlyName}} []{{.GoFriendlyName}}
	rows, err := txWrapper.Tx.Query("{{.GenericSelectQuery}}")

	if err != nil {
		return nil, NewModelsError(errorPrefix + " fatal error running the query:", err)
	}
	defer rows.Close()

	for rows.Next() {

		// create a new instance of {{.GoFriendlyName}}  {{$instanceVarName := print "current" .GoFriendlyName}}
		
		{{$instanceVarName}} := {{.GoFriendlyName}}{}

		err := rows.Scan({{range $i, $e := .Columns}}&{{$instanceVarName}}.{{$e.GoName}}{{if ne (plus1 $i) $colCount}},{{end}}{{end}})
		if err != nil {
			return nil, NewModelsError(errorPrefix + " error during rows.Scan():", err)
		}
		
		sliceOf{{.GoFriendlyName}} = append(sliceOf{{.GoFriendlyName}}, current{{.GoFriendlyName}})

	}
	err = rows.Err()
	if err != nil {
		return nil, NewModelsError(errorPrefix + " error during rows.Next() iterations:", err)
	}	
	
	return sliceOf{{.GoFriendlyName}}, nil
}
`
