package main

/* Select Functions Templates */

/* Templates for the Where method */

const SELECT_TEMPLATE_WHERE = `{{$colCount := len .Columns}}
{{$functionName := "Where"}}{{$sourceStructName := print "source" .GoFriendlyName}}
// Returns the rows from {{.DbName}}, corresponding to the supplied condition 
// and the respective parameters. The condition must not include the WHERE keyword.
// The rows are converted to a slice of {{.GoFriendlyName}} instances
// If operation fails, it returns nil and the error.
func (utilRef *t{{.GoFriendlyName}}Utils) {{$functionName}}(condition string, params ...interface{}) ([]{{.GoFriendlyName}},  error) {
						
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "

	if condition == "" {
		return nil, NewModelsErrorLocal(errorPrefix, "No condition specified. Please use SelectAll method to select all rows from {{.DbName}}")
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
	
	return sliceOf{{.GoFriendlyName}}, nil
}
`

const SELECT_TEMPLATE_WHERE_TX = `{{$colCount := len .Columns}}
{{$functionName := print "Where" .GoFriendlyName}}{{$sourceStructName := print "source" .GoFriendlyName}}
// Returns the rows from {{.DbName}}, corresponding to the supplied condition 
// and the respective parameters. The condition must not include the WHERE keyword.
// The rows are converted to a slice of {{.GoFriendlyName}} instances
// If operation fails, it returns nil and the error.
func (txWrapper *Transaction) {{$functionName}}(condition string, params ...interface{}) ([]{{.GoFriendlyName}},  error) {
						
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "

	if condition == "" {
		return nil, NewModelsErrorLocal(errorPrefix, "No condition specified. Please use SelectAll method to select all rows from {{.DbName}}")
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
