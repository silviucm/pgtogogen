package main

/* Update Functions Templates */

const TABLE_STATIC_UPDATE_TEMPLATE = `{{$colCount := len .Columns}}{{$pkColCount := len .PKColumns}}
{{$functionName := "Update"}}{{$sourceStructName := print "source" .GoFriendlyName}}
// {{$functionName}} attempts to update the rows inside the {{.DbName}} table, based on 
// the supplied condition  and the respective parameters. 
// The condition must not include the WHERE keyword.  Make sure to start the dollar-prefixed 
// params inside the condition from {{plus1 $colCount}}.
// All the fields in the supplied source {{.GoFriendlyName}} pointer will be updated.
// If you need only certain fields to be updated, you will have to create a custom method, 
// or use UpdateWithMask().
// Returns the number of affected rows (zero if no rows found for that condition), 
// and nil error for a successful operation. If the operation fails, it returns 0 and the error.
func (utilRef *t{{.GoFriendlyName}}Utils) {{$functionName}}({{$sourceStructName}} *{{.GoFriendlyName}}, conditionParamsStartAt{{plus1 $colCount}} string, params ...interface{}) (int64,  error) {
						
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "

	if conditionParamsStartAt{{plus1 $colCount}} == "" {
		return 0, NewModelsErrorLocal(errorPrefix, "No condition specified. Please use UpdateAll method to update all rows inside {{.DbName}}")
	}
	
	currentDbHandle := GetDb()
	if currentDbHandle == nil {
		return 0, NewModelsErrorLocal(errorPrefix, "the database handle is nil")
	}

	// define the update query
	queryBuffer := bytes.Buffer{}
	_, writeErr := queryBuffer.WriteString("UPDATE {{.DbName}} SET {{range $i, $e := .Columns}}{{$e.DbName}} = ${{(plus1 $i)}}{{if ne (plus1 $i) $colCount}},{{end}}{{end}} WHERE ")
	if writeErr != nil {
		return 0, NewModelsError(errorPrefix + "queryBuffer.WriteString error:",writeErr)
	}

	_, writeErr = queryBuffer.WriteString(conditionParamsStartAt{{plus1 $colCount}})
	if writeErr != nil {
		return 0, NewModelsError(errorPrefix + "queryBuffer.WriteString (condition param) error:",writeErr)
	}	
	
	instanceValuesSlice := []interface{} { {{range $i, $e := .Columns}}{{if .Nullable}}{{generateNullableTypeStructTemplate .GoNullableType (print $sourceStructName "." $e.GoName) (print $sourceStructName "." $e.GoName "_IsNotNull")}}{{else}}{{$sourceStructName}}.{{$e.GoName}}{{end}}{{if ne (plus1 $i) $colCount}},{{end}}{{end}} }
	
	allParams := append(instanceValuesSlice, params...)	
	
	r, err := currentDbHandle.Exec(queryBuffer.String(), allParams...)
	if err != nil {
		
		{{if gt (len .UniqueConstraints) 0}}if Contains(err.Error(),"SQLSTATE 23505") {
		{{range $e := .UniqueConstraints}}	if Contains(err.Error(),"{{$e.DbName}}") { return 0,Err{{$e.ParentTable.GoFriendlyName}}_UQ_{{$e.DbName}}	}				
		{{end}}
		} {{end}}		
		
		return 0, NewModelsError(errorPrefix + "db.Exec error:",err)
	}
	
	n := r.RowsAffected()
	return n, nil	
	
}
`

const TABLE_STATIC_UPDATE_TEMPLATE_TX = `{{$colCount := len .Columns}}{{$pkColCount := len .PKColumns}}
{{$functionName := print "Update" .GoFriendlyName}}{{$sourceStructName := print "source" .GoFriendlyName}}
// {{$functionName}} attempts to update the rows inside the {{.DbName}} table, based on 
// the supplied condition  and the respective parameters. 
// The condition must not include the WHERE keyword. Make sure to start the dollar-prefixed 
// params inside the condition from {{plus1 $colCount}}.
// All the fields in the supplied source {{.GoFriendlyName}} pointer will be updated.
// If you need only certain fields to be updated, you will have to create a custom method, 
// or use UpdateWithMask().
// Returns the number of affected rows (zero if no rows found for that condition), 
// and nil error for a successful operation. If the operation fails, it returns 0 and the error.
func (txWrapper *Transaction) {{$functionName}}({{$sourceStructName}} *{{.GoFriendlyName}}, conditionParamsStartAt{{plus1 $colCount}} string, params ...interface{}) (int64,  error) {
						
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "

	if conditionParamsStartAt{{plus1 $colCount}} == "" {
		return 0, NewModelsErrorLocal(errorPrefix, "No condition specified. Please use UpdateAll method to update all rows inside {{.DbName}}")
	}
	
	if txWrapper == nil { return 0, NewModelsErrorLocal(errorPrefix, "the transaction wrapper is nil") }
	if txWrapper.Tx == nil { return 0, NewModelsErrorLocal(errorPrefix, "the transaction object is nil") }

	// define the update query
	queryBuffer := bytes.Buffer{}
	_, writeErr := queryBuffer.WriteString("UPDATE {{.DbName}} SET {{range $i, $e := .Columns}}{{$e.DbName}} = ${{(plus1 $i)}}{{if ne (plus1 $i) $colCount}},{{end}}{{end}} WHERE ")
	if writeErr != nil {
		return 0, NewModelsError(errorPrefix + "queryBuffer.WriteString error:",writeErr)
	}

	_, writeErr = queryBuffer.WriteString(conditionParamsStartAt{{plus1 $colCount}})
	if writeErr != nil {
		return 0, NewModelsError(errorPrefix + "queryBuffer.WriteString (condition param) error:",writeErr)
	}	
	
	instanceValuesSlice := []interface{} { {{range $i, $e := .Columns}}{{if .Nullable}}{{generateNullableTypeStructTemplate .GoNullableType (print $sourceStructName "." $e.GoName) (print $sourceStructName "." $e.GoName "_IsNotNull")}}{{else}}{{$sourceStructName}}.{{$e.GoName}}{{end}}{{if ne (plus1 $i) $colCount}},{{end}}{{end}} }
	
	allParams := append(instanceValuesSlice, params...)
	
	r, err := txWrapper.Tx.Exec(queryBuffer.String(), allParams...)
	if err != nil {
		
		{{if gt (len .UniqueConstraints) 0}}if Contains(err.Error(),"SQLSTATE 23505") {
		{{range $e := .UniqueConstraints}}	if Contains(err.Error(),"{{$e.DbName}}") { return 0,Err{{$e.ParentTable.GoFriendlyName}}_UQ_{{$e.DbName}}	}				
		{{end}}
		} {{end}}		
		
		return 0, NewModelsError(errorPrefix + "db.Exec error:",err)
	}
	
	n := r.RowsAffected()
	return n, nil	
	
}
`

const TABLE_STATIC_UPDATE_WITH_MASK = `{{$colCount := len .Columns}}{{$pkColCount := len .PKColumns}}
{{$functionName := "UpdateWithMask"}}{{$sourceStructName := print "source" .GoFriendlyName}}
// {{$functionName}} attempts to update the rows inside the {{.DbName}} table, based on 
// the supplied condition  and the respective parameters. 
// The condition must not include the WHERE keyword.  Make sure to start the dollar-prefixed params 
// inside the condition from the number of elements supplied in the update mask, plus one.
// Only the fields in the supplied mask slice of strings will be updated. 
// If the mask is nil, all fields will be updated.
// Returns the number of affected rows (zero if no rows found for that condition), and nil error 
// in case of a successful operation. If the operation fails, it returns 0 and the error.
func (utilRef *t{{.GoFriendlyName}}Utils) {{$functionName}}({{$sourceStructName}} *{{.GoFriendlyName}}, updateMask []string, condition string, params ...interface{}) (int64,  error) {
						
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "

	if condition == "" {
		return 0, NewModelsErrorLocal(errorPrefix, "No condition specified. Please use UpdateAll method to update all rows inside {{.DbName}}")
	}

	if updateMask == nil {
		return 0, NewModelsErrorLocal(errorPrefix, "No update mask specified. Please use Update or UpdateAll method to update all fields.")
	}
	if len(updateMask) == 0 {
		return 0, NewModelsErrorLocal(errorPrefix, "No update mask specified. Please use Update or UpdateAll method to update all fields.")
	}
	
	currentDbHandle := GetDb()
	if currentDbHandle == nil {
		return 0, NewModelsErrorLocal(errorPrefix, "the database handle is nil")
	}

	// define the update query
	queryBuffer := bytes.Buffer{}
	_, writeErr := queryBuffer.WriteString("UPDATE {{.DbName}} SET ")
	if writeErr != nil {
		return 0, NewModelsError(errorPrefix + "queryBuffer.WriteString error:",writeErr)
	}
	
	var instanceValuesSlice []interface{}
	for i,e := range updateMask {
		
		_, writeErr = queryBuffer.WriteString(utilRef.ToDbFieldName(e))
		if writeErr != nil {
			return 0, NewModelsError(errorPrefix + "queryBuffer.WriteString error (inside range updateMask):",writeErr)
		}

		_, writeErr = queryBuffer.WriteString("=$")
		if writeErr != nil {
			return 0, NewModelsError(errorPrefix + "queryBuffer.WriteString error (inside range updateMask):",writeErr)
		}

		_, writeErr = queryBuffer.WriteString(Itoa(i+1))
		if writeErr != nil {
			return 0, NewModelsError(errorPrefix + "queryBuffer.WriteString error (inside range updateMask):",writeErr)
		}

		// add a comma if not the final set
		if len(updateMask) != (i + 1) {
			_, writeErr = queryBuffer.WriteString(",")
			if writeErr != nil {
				return 0, NewModelsError(errorPrefix + "queryBuffer.WriteString error (inside range updateMask):",writeErr)
			}
		}
				
		{{range $i, $e := .Columns}}if e == "{{$e.GoName}}" || e == "{{$e.DbName}}" { instanceValuesSlice = append(instanceValuesSlice, {{$sourceStructName}}.{{$e.GoName}}) }			
		{{end}}
		
	}
	
	_, writeErr = queryBuffer.WriteString(" WHERE ")
	if writeErr != nil {
		return 0, NewModelsError(errorPrefix + "queryBuffer.WriteString error:",writeErr)
	}	

	_, writeErr = queryBuffer.WriteString(condition)
	if writeErr != nil {
		return 0, NewModelsError(errorPrefix + "queryBuffer.WriteString (condition param) error:",writeErr)
	}	
	
	// append the condition's params to the ones of the setters
	allParams := append(instanceValuesSlice, params...)			
	
	r, err := currentDbHandle.Exec(queryBuffer.String(), allParams...)
	if err != nil {
		
		{{if gt (len .UniqueConstraints) 0}}if Contains(err.Error(),"SQLSTATE 23505") {
		{{range $e := .UniqueConstraints}}	if Contains(err.Error(),"{{$e.DbName}}") { return 0,Err{{$e.ParentTable.GoFriendlyName}}_UQ_{{$e.DbName}}	}				
		{{end}}
		} {{end}}		
		
		return 0, NewModelsError(errorPrefix + "db.Exec error:",err)
	}
	
	n := r.RowsAffected()
	return n, nil	
	
}
`

const TABLE_STATIC_UPDATE_WITH_MASK_TX = `{{$colCount := len .Columns}}{{$pkColCount := len .PKColumns}}
{{$functionName := print "UpdateWithMask" .GoFriendlyName}}{{$sourceStructName := print "source" .GoFriendlyName}}
// {{$functionName}} attempts to update the rows inside the {{.DbName}} table, based on 
// the supplied condition  and the respective parameters. The condition must not include 
// the WHERE keyword.  Make sure to start the dollar-prefixed params inside the condition 
// from the number of elements supplied in the update mask, plus one.
// Only the fields in the supplied mask slice of strings will be updated. If the mask is nil, 
// all fields will be updated.
// Returns the number of affected rows (zero if no rows found for that condition), 
// and nil error for a successful operation. If operation fails, it returns 0 and the error.
func (txWrapper *Transaction) {{$functionName}}({{$sourceStructName}} *{{.GoFriendlyName}}, updateMask []string, condition string, params ...interface{}) (int64,  error) {
						
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "

	if condition == "" {
		return 0, NewModelsErrorLocal(errorPrefix, "No condition specified. Please use UpdateAll method to update all rows inside {{.DbName}}")
	}

	if updateMask == nil {
		return 0, NewModelsErrorLocal(errorPrefix, "No update mask specified. Please use Update or UpdateAll method to update all fields.")
	}
	if len(updateMask) == 0 {
		return 0, NewModelsErrorLocal(errorPrefix, "No update mask specified. Please use Update or UpdateAll method to update all fields.")
	}
	
	if txWrapper == nil { return 0, NewModelsErrorLocal(errorPrefix, "the transaction wrapper is nil") }
	if txWrapper.Tx == nil { return 0, NewModelsErrorLocal(errorPrefix, "the transaction object is nil") }


	// define the update query
	queryBuffer := bytes.Buffer{}
	_, writeErr := queryBuffer.WriteString("UPDATE {{.DbName}} SET ")
	if writeErr != nil {
		return 0, NewModelsError(errorPrefix + "queryBuffer.WriteString error:",writeErr)
	}
	
	var instanceValuesSlice []interface{}
	for i,e := range updateMask {
		
		_, writeErr = queryBuffer.WriteString(Tables.{{.GoFriendlyName}}.ToDbFieldName(e))
		if writeErr != nil {
			return 0, NewModelsError(errorPrefix + "queryBuffer.WriteString error (inside range updateMask):",writeErr)
		}

		_, writeErr = queryBuffer.WriteString("=$")
		if writeErr != nil {
			return 0, NewModelsError(errorPrefix + "queryBuffer.WriteString error (inside range updateMask):",writeErr)
		}

		_, writeErr = queryBuffer.WriteString(Itoa(i+1))
		if writeErr != nil {
			return 0, NewModelsError(errorPrefix + "queryBuffer.WriteString error (inside range updateMask):",writeErr)
		}

		// add a comma if not the final set
		if len(updateMask) != (i + 1) {
			_, writeErr = queryBuffer.WriteString(",")
			if writeErr != nil {
				return 0, NewModelsError(errorPrefix + "queryBuffer.WriteString error (inside range updateMask):",writeErr)
			}
		}
				
		{{range $i, $e := .Columns}}if e == "{{$e.GoName}}" || e == "{{$e.DbName}}" { instanceValuesSlice = append(instanceValuesSlice, {{$sourceStructName}}.{{$e.GoName}}) }			
		{{end}}
		
	}
	
	_, writeErr = queryBuffer.WriteString(" WHERE ")
	if writeErr != nil {
		return 0, NewModelsError(errorPrefix + "queryBuffer.WriteString error:",writeErr)
	}	

	_, writeErr = queryBuffer.WriteString(condition)
	if writeErr != nil {
		return 0, NewModelsError(errorPrefix + "queryBuffer.WriteString (condition param) error:",writeErr)
	}	
	
	// append the condition's params to the ones of the setters
	allParams := append(instanceValuesSlice, params...)			
	
	r, err := txWrapper.Tx.Exec(queryBuffer.String(), allParams...)
	if err != nil {
		
		{{if gt (len .UniqueConstraints) 0}}if Contains(err.Error(),"SQLSTATE 23505") {
		{{range $e := .UniqueConstraints}}	if Contains(err.Error(),"{{$e.DbName}}") { return 0,Err{{$e.ParentTable.GoFriendlyName}}_UQ_{{$e.DbName}}	}				
		{{end}}
		} {{end}}		
		
		return 0, NewModelsError(errorPrefix + "db.Exec error:",err)
	}
	
	n := r.RowsAffected()
	return n, nil	
	
}
`

const TABLE_INSTANCE_UPDATE_TEMPLATE = `{{if lt 0 (len .PKColumns)}}{{$colCount := len .Columns}}{{$pkColCount := len .PKColumns}}
{{$functionName := "Update"}}{{$sourceStructName := print "source" .GoFriendlyName}}
// {{$functionName}} attempts to update the row inside the {{.DbName}} table, corresponding 
// to the PK of the current {{.GoFriendlyName}} instance.
// All the fields in the supplied source {{.GoFriendlyName}} pointer will be updated.
// If you need only certain fields to be updated, you will have to create a custom method, 
// or use UpdateWithMask().
// Returns nil error for a successful operation. If operation fails, it returns the error. 
// If more than one row gets updated, it will return an error.
func ({{$sourceStructName}} *{{.GoFriendlyName}}) {{$functionName}}() error {
						
	var errorPrefix = "instance of {{.GoFriendlyName}}.{{$functionName}}() ERROR: "
	
	currentDbHandle := GetDb()
	if currentDbHandle == nil {
		return NewModelsErrorLocal(errorPrefix, "the database handle is nil")
	}

	// define the update query
	queryBuffer := bytes.Buffer{}
	_, writeErr := queryBuffer.WriteString("UPDATE {{.DbName}} SET {{range $i, $e := .Columns}}{{$e.DbName}} = ${{(plus1 $i)}}{{if ne (plus1 $i) $colCount}},{{end}}{{end}} WHERE ")
	if writeErr != nil {
		return NewModelsError(errorPrefix + "queryBuffer.WriteString error:",writeErr)
	}

	_, writeErr = queryBuffer.WriteString("{{range $i, $e := .PKColumns}}{{$e.DbName}}=${{plus (plus1 $i) $colCount}}{{if ne (plus1 $i) $pkColCount}} AND {{end}}{{end}}")
	if writeErr != nil {
		return NewModelsError(errorPrefix + "queryBuffer.WriteString (instance condition param) error:",writeErr)
	}	
	
	instanceValuesSlice := []interface{} { {{range $i, $e := .Columns}}{{if .Nullable}}{{generateNullableTypeStructTemplate .GoNullableType (print $sourceStructName "." $e.GoName) (print $sourceStructName "." $e.GoName "_IsNotNull")}}{{else}}{{$sourceStructName}}.{{$e.GoName}}{{end}}{{if ne (plus1 $i) $colCount}},{{end}}{{end}}, {{range $i, $e := .PKColumns}}{{$sourceStructName}}.{{$e.GoName}}{{if ne (plus1 $i) $pkColCount}},{{end}}{{end}}  }
	
	r, err := currentDbHandle.Exec(queryBuffer.String(), instanceValuesSlice...)
	if err != nil {
		
		{{if gt (len .UniqueConstraints) 0}}if Contains(err.Error(),"SQLSTATE 23505") {
		{{range $e := .UniqueConstraints}}	if Contains(err.Error(),"{{$e.DbName}}") { return Err{{$e.ParentTable.GoFriendlyName}}_UQ_{{$e.DbName}}	}				
		{{end}}
		} {{end}}		
		
		return NewModelsError(errorPrefix + "db.Exec error:",err)
	}
	
	n := r.RowsAffected()
	if n > 1 {
		return  NewModelsErrorLocal(errorPrefix, "More than one record was updated: " + Itoa(int(n)))
	}
	return nil	
	
}{{end}}
`

const TABLE_INSTANCE_UPDATE_TEMPLATE_TX = `{{if lt 0 (len .PKColumns)}}{{$colCount := len .Columns}}{{$pkColCount := len .PKColumns}}
{{$functionName := print "UpdateSingleInstance" .GoFriendlyName}}{{$sourceStructName := print "source" .GoFriendlyName}}
// {{$functionName}} attempts to update the row inside the {{.DbName}} table, corresponding to 
// the PK of the current {{.GoFriendlyName}} instance.
// All the fields in the supplied source {{.GoFriendlyName}} pointer will be updated.
// If you need only certain fields to be updated, you will have to create a custom method, 
// or use UpdateWithMask().
// Returns nil error for a successful operation. If operation fails, it returns the error. 
// If more than one row gets updated, it will return an error.
func (txWrapper *Transaction) {{$functionName}}({{$sourceStructName}} *{{.GoFriendlyName}}) error {
						
	var errorPrefix = "instance of {{.GoFriendlyName}}.{{$functionName}}() ERROR: "
	
	if txWrapper == nil { return NewModelsErrorLocal(errorPrefix, "the transaction wrapper is nil") }
	if txWrapper.Tx == nil { return NewModelsErrorLocal(errorPrefix, "the transaction object is nil") }


	// define the update query
	queryBuffer := bytes.Buffer{}
	_, writeErr := queryBuffer.WriteString("UPDATE {{.DbName}} SET {{range $i, $e := .Columns}}{{$e.DbName}} = ${{(plus1 $i)}}{{if ne (plus1 $i) $colCount}},{{end}}{{end}} WHERE ")
	if writeErr != nil {
		return NewModelsError(errorPrefix + "queryBuffer.WriteString error:",writeErr)
	}

	_, writeErr = queryBuffer.WriteString("{{range $i, $e := .PKColumns}}{{$e.DbName}}=${{plus (plus1 $i) $colCount}}{{if ne (plus1 $i) $pkColCount}} AND {{end}}{{end}}")
	if writeErr != nil {
		return NewModelsError(errorPrefix + "queryBuffer.WriteString (instance condition param) error:",writeErr)
	}	
	
	instanceValuesSlice := []interface{} { {{range $i, $e := .Columns}}{{if .Nullable}}{{generateNullableTypeStructTemplate .GoNullableType (print $sourceStructName "." $e.GoName) (print $sourceStructName "." $e.GoName "_IsNotNull")}}{{else}}{{$sourceStructName}}.{{$e.GoName}}{{end}}{{if ne (plus1 $i) $colCount}},{{end}}{{end}}, {{range $i, $e := .PKColumns}}{{$sourceStructName}}.{{$e.GoName}}{{if ne (plus1 $i) $pkColCount}},{{end}}{{end}}  }
	
	r, err := txWrapper.Tx.Exec(queryBuffer.String(), instanceValuesSlice...)
	if err != nil {
		
		{{if gt (len .UniqueConstraints) 0}}if Contains(err.Error(),"SQLSTATE 23505") {
		{{range $e := .UniqueConstraints}}	if Contains(err.Error(),"{{$e.DbName}}") { return Err{{$e.ParentTable.GoFriendlyName}}_UQ_{{$e.DbName}}	}				
		{{end}}
		} {{end}}		
		
		return NewModelsError(errorPrefix + "db.Exec error:",err)
	}
	
	n := r.RowsAffected()
	if n > 1 {
		return  NewModelsErrorLocal(errorPrefix, "More than one record was updated: " + Itoa(int(n)))
	}
	return nil	
	
}{{end}}
`
