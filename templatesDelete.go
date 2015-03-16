package main

/* Insert Functions Templates */

const TABLE_STATIC_DELETE_TEMPLATE = `{{$colCount := len .Columns}}{{$pkColCount := len .PKColumns}}
{{$functionName := "Delete"}}{{$sourceStructName := print "source" .GoFriendlyName}}
// Deletes the row from the {{.TableName}} table, corresponding to the supplied condition 
// and the respective parameters. The condition must not include the WHERE keyword.
// Returns the number of deleted rows (zero if no rows found for that condition), and nil error for a successful operation.
// If operation fails, it returns zero and the error.
func (utilRef *t{{.GoFriendlyName}}Utils) {{$functionName}}(condition string, params ...interface{}) (int64,  error) {
						
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "

	if condition == "" {
		return 0, NewModelsErrorLocal(errorPrefix, "No condition specified. Please use DeleteAll method to delete all rows from {{.TableName}}")
	}
	
	currentDbHandle := GetDb()
	if currentDbHandle == nil {
		return 0, NewModelsErrorLocal(errorPrefix, "the database handle is nil")
	}

	// define the delete query
	queryBuffer := bytes.Buffer{}
	_, writeErr := queryBuffer.WriteString("DELETE FROM {{.TableName}} WHERE ")
	if writeErr != nil {
		return 0, NewModelsError(errorPrefix + "queryBuffer.WriteString error:",writeErr)
	}

	_, writeErr = queryBuffer.WriteString(condition)
	if writeErr != nil {
		return 0, NewModelsError(errorPrefix + "queryBuffer.WriteString (condition param) error:",writeErr)
	}	
	
	r, err := currentDbHandle.Exec(queryBuffer.String(), params...)
	if err != nil {
		return 0, NewModelsError(errorPrefix + "db.Exec error:",err)
	}
	
	n := r.RowsAffected()
	return n, nil	
	
}
`

const TABLE_STATIC_DELETE_TEMPLATE_TX = `{{$colCount := len .Columns}}{{$pkColCount := len .PKColumns}}
{{$functionName := print "Delete" .GoFriendlyName}}{{$sourceStructName := print "source" .GoFriendlyName}}
// Deletes the row from the {{.TableName}} table, corresponding to the supplied condition 
// and the respective parameters. The condition must not include the WHERE keyword.
// Returns the number of deleted rows (zero if no rows found for that condition), and nil error for a successful operation.
// If operation fails, it returns zero and the error.
func (txWrapper *Transaction) {{$functionName}}(condition string, params ...interface{}) (int64,  error) {
						
	var errorPrefix = "txWrapper.{{$functionName}}() ERROR: "

	if condition == "" {
		return 0, NewModelsErrorLocal(errorPrefix, "No condition specified. Please use DeleteAll method to delete all rows from {{.TableName}}")
	}
	
	if txWrapper == nil { return 0, NewModelsErrorLocal(errorPrefix, "the transaction wrapper is nil") }
	if txWrapper.Tx == nil { return 0, NewModelsErrorLocal(errorPrefix, "the transaction object is nil") }

	// define the delete query
	queryBuffer := bytes.Buffer{}
	_, writeErr := queryBuffer.WriteString("DELETE FROM {{.TableName}} WHERE ")
	if writeErr != nil {
		return 0, NewModelsError(errorPrefix + "queryBuffer.WriteString error:",writeErr)
	}

	_, writeErr = queryBuffer.WriteString(condition)
	if writeErr != nil {
		return 0, NewModelsError(errorPrefix + "queryBuffer.WriteString (condition param) error:",writeErr)
	}	
	
	r, err := txWrapper.Tx.Exec(queryBuffer.String(), params...)
	if err != nil {
		return 0, NewModelsError(errorPrefix + "db.Exec error:",err)
	}
	
	n := r.RowsAffected()
	return n, nil	
	
}
`

const TABLE_STATIC_DELETE_ALL_TEMPLATE = `{{$colCount := len .Columns}}{{$pkColCount := len .PKColumns}}
{{$functionName := "DeleteAll"}}{{$sourceStructName := print "source" .GoFriendlyName}}
// Deletes all existing rows from the {{.TableName}} table.
// Returns the number of deleted rows (zero if no rows found), and nil error for a successful operation.
// If operation fails, it returns zero and the error.
func (utilRef *t{{.GoFriendlyName}}Utils) {{$functionName}}() (int64,  error) {
						
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "
	
	currentDbHandle := GetDb()
	if currentDbHandle == nil {
		return 0, NewModelsErrorLocal(errorPrefix, "the database handle is nil")
	}
	
	r, err := currentDbHandle.Exec("DELETE FROM {{.TableName}}")
	if err != nil {
		return 0, NewModelsError(errorPrefix + "currentDbHandle.Exec error:",err)
	}
	
	n := r.RowsAffected()
	return n, nil	
	
}
`

const TABLE_STATIC_DELETE_ALL_TEMPLATE_TX = `{{$colCount := len .Columns}}{{$pkColCount := len .PKColumns}}
{{$functionName := print "DeleteAll" .GoFriendlyName}}{{$sourceStructName := print "source" .GoFriendlyName}}
// Deletes all existing rows from the {{.TableName}} table.
// Returns the number of deleted rows (zero if no rows found), and nil error for a successful operation.
// If operation fails, it returns zero and the error.
func (txWrapper *Transaction) {{$functionName}}() (int64,  error) {
						
	var errorPrefix = "txWrapper.{{$functionName}}() ERROR: "
	
	if txWrapper == nil { return 0, NewModelsErrorLocal(errorPrefix, "the transaction wrapper is nil") }
	if txWrapper.Tx == nil { return 0, NewModelsErrorLocal(errorPrefix, "the transaction object is nil") }
	
	r, err := txWrapper.Tx.Exec("DELETE FROM {{.TableName}}")
	if err != nil {
		return 0, NewModelsError(errorPrefix + "currentDbHandle.Exec error:",err)
	}
	
	n := r.RowsAffected()
	return n, nil	
	
}
`

const TABLE_STATIC_DELETE_INSTANCE_TEMPLATE = `{{$colCount := len .Columns}}{{$pkColCount := len .PKColumns}}
{{$functionName := "DeleteInstance"}}{{$sourceStructName := print "source" .GoFriendlyName}}
// Deletes the row from the {{.TableName}} table, corresponding to the primary key fields
// inside the {{$sourceStructName}} parameter.
// Returns true if the row was deleted, or false and nil error if no such PK value was found in the database.
// If operation fails, it returns nil and the error.
func (utilRef *t{{.GoFriendlyName}}Utils) {{$functionName}}({{$sourceStructName}} *{{.GoFriendlyName}}) (bool,  error) {
						
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "

	if {{$sourceStructName}} == nil {
		return false, NewModelsErrorLocal(errorPrefix, "the {{$sourceStructName}} pointer is nil")
	}	

	// define the condition based on the PK columns
	var deleteInstanceQueryCondition string = "	{{range $i, $e := .PKColumns}}{{.Name}} = ${{print (plus1 $i)}}{{if ne (plus1 $i) $pkColCount}},{{end}}{{end}}"

	rowCount, err := Tables.{{.GoFriendlyName}}.Delete(deleteInstanceQueryCondition, {{range $i, $e := .PKColumns}}{{$sourceStructName}}.{{$e.GoName}}{{if ne (plus1 $i) $colCount}},{{end}}{{end}})
	if err != nil {
		return false, NewModelsError(errorPrefix,err)
	}
	
	if rowCount > 1 {
		return false, NewModelsError(errorPrefix + ": FATAL ERROR: Too many rows deleted !! : ",err)
	}	
	
	return rowCount == 1, nil
	
}
`

const TABLE_STATIC_DELETE_INSTANCE_TEMPLATE_TX = `{{$colCount := len .Columns}}{{$pkColCount := len .PKColumns}}
{{$functionName := print "DeleteInstance" .GoFriendlyName}}{{$sourceStructName := print "source" .GoFriendlyName}}
// Deletes the row from the {{.TableName}} table, corresponding to the primary key fields
// inside the {{$sourceStructName}} parameter.
// Returns true if the row was deleted, or false and nil error if no such PK value was found in the database.
// If operation fails, it returns nil and the error.
func (txWrapper *Transaction) {{$functionName}}({{$sourceStructName}} *{{.GoFriendlyName}}) (bool,  error) {
						
	var errorPrefix = "txWrapper.{{$functionName}}() ERROR: "

	if {{$sourceStructName}} == nil {
		return false, NewModelsErrorLocal(errorPrefix, "the {{$sourceStructName}} pointer is nil")
	}	

	if txWrapper == nil { return false, NewModelsErrorLocal(errorPrefix, "the transaction wrapper is nil") }
	if txWrapper.Tx == nil { return false, NewModelsErrorLocal(errorPrefix, "the transaction object is nil") }
		

	// define the condition based on the PK columns
	var deleteInstanceQueryCondition string = "	{{range $i, $e := .PKColumns}}{{.Name}} = ${{print (plus1 $i)}}{{if ne (plus1 $i) $pkColCount}},{{end}}{{end}}"

	rowCount, err := txWrapper.Delete{{.GoFriendlyName}}(deleteInstanceQueryCondition, {{range $i, $e := .PKColumns}}{{$sourceStructName}}.{{$e.GoName}}{{if ne (plus1 $i) $colCount}},{{end}}{{end}})
	if err != nil {
		return false, NewModelsError(errorPrefix,err)
	}
	
	if rowCount > 1 {
		return false, NewModelsError(errorPrefix + ": FATAL ERROR: Too many rows deleted !! : ",err)
	}	
	
	return rowCount == 1, nil
	
}
`
