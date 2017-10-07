package main

/* Select Single Rows by Columns */

/* BEGIN: Primary Key Getter Templates */

const PK_GETTER_TEMPLATE_ATOMIC = `{{$colCount := len .ParentTable.Columns}}{{$pkColCount := len .ParentTable.PKColumns}}{{$functionName := "GetBy"}}
// Queries the database for a single row based on the specified single or multi-column primary key.
// Returns a pointer to a {{.ParentTable.GoFriendlyName}} structure if a record was found,
// otherwise it returns nil.
func (utilRef *t{{.ParentTable.GoFriendlyName}}Utils) {{$functionName}}` +
	`{{if gt $pkColCount 1}}` +
	`{{range $i, $e := .ParentTable.PKColumns}}{{$e.GoName}}{{if ne (plus1 $i) $pkColCount}}And{{end}}{{end}}(` +
	`{{else}}{{range $i, $e := .ParentTable.PKColumns}}{{$e.GoName}}{{end}}(` +
	`{{end}}` +
	`{{range $i, $e := .ParentTable.PKColumns}}input{{$e.GoName}} {{$e.GoType}} {{if ne (plus1 $i) $pkColCount}},{{end}}{{end}})` +
	` (returnStruct *{{.ParentTable.GoFriendlyName}}, err error) {
	
	returnStruct = nil
	err = nil
	
	var errorPrefix = "{{.ParentTable.GoFriendlyName}}GetBy{{.GoName}}() ERROR: "
	
	currentDbHandle := GetDb()
	if currentDbHandle == nil {
		return nil, NewModelsErrorLocal(errorPrefix, "the database handle is nil")
	}

	// define receiving params for the row iteration
	{{range $e := .ParentTable.Columns}}{{if .Nullable}}var param{{.GoName}} {{$e.GoNullableType}}
	{{else}}var param{{.GoName}} {{.GoType}}
	{{end}}{{end}}

	// define the select query
	var query = "{{.ParentTable.GenericSelectQuery}} WHERE {{range $i, $e := .ParentTable.PKColumns}}{{.DbName}} = ${{print (plus1 $i)}}{{if ne (plus1 $i) $pkColCount}} AND {{end}}{{end}}";

	// we are aiming for a single row so we will use Query Row	
	err = currentDbHandle.QueryRow(query, ` +
	`{{range $i, $e := .ParentTable.PKColumns}}input{{$e.GoName}}{{if ne (plus1 $i) $pkColCount}},{{end}}{{end}}` +
	`).Scan({{range $i, $e := .ParentTable.Columns}}&param{{$e.GoName}}{{if ne (plus1 $i) $colCount}},{{end}}{{end}})
    switch {
    case err == ErrNoRows:
            // no such row found, return nil and nil
			return nil, nil
    case err != nil:
            return nil, NewModelsError(errorPrefix + "fatal error running the query:", err)
    default:
           	// create the return structure as a pointer of the type
			returnStruct = &{{.ParentTable.GoFriendlyName}}{
				{{range .ParentTable.Columns}}{{if not .Nullable}}{{.GoName}}: param{{.GoName}},
				{{end}}{{end}}
			}
			{{range $e := .ParentTable.Columns}}{{if $e.Nullable}}returnStruct.Set{{.GoName}}(param{{$e.GoName}}.{{getNullableTypeValueFieldName $e.GoNullableType}}, boolFromStatus(param{{$e.GoName}}.Status))
			{{end}}{{end}}			
			// return the structure
			return returnStruct, nil
    }			
}
`

const PK_GETTER_TEMPLATE_TX = `{{$colCount := len .ParentTable.Columns}}{{$pkColCount := len .ParentTable.PKColumns}}{{$functionName := print "Get" .ParentTable.GoFriendlyName "By"}}
// Queries the database for a single row based on the specified single or multi-column primary key.
// Returns a pointer to a {{.ParentTable.GoFriendlyName}} structure if a record was found,
// otherwise it returns nil.
func (txWrapper *Transaction) {{$functionName}}` +
	`{{if gt $pkColCount 1}}` +
	`{{range $i, $e := .ParentTable.PKColumns}}{{$e.GoName}}{{if ne (plus1 $i) $pkColCount}}And{{end}}{{end}}(` +
	`{{else}}{{range $i, $e := .ParentTable.PKColumns}}{{$e.GoName}}{{end}}(` +
	`{{end}}` +
	`{{range $i, $e := .ParentTable.PKColumns}}input{{$e.GoName}} {{$e.GoType}} {{if ne (plus1 $i) $pkColCount}},{{end}}{{end}})` +
	` (returnStruct *{{.ParentTable.GoFriendlyName}}, err error) {
	
	returnStruct = nil
	err = nil
	
	var errorPrefix = "{{.ParentTable.GoFriendlyName}}GetBy{{.GoName}}() ERROR: "
	
	if txWrapper == nil { return nil, NewModelsErrorLocal(errorPrefix, "the transaction wrapper is nil") }
	if txWrapper.Tx == nil { return nil, NewModelsErrorLocal(errorPrefix, "the transaction object is nil") }	

	// define receiving params for the row iteration
	{{range $e := .ParentTable.Columns}}{{if .Nullable}}var param{{.GoName}} {{$e.GoNullableType}}
	{{else}}var param{{.GoName}} {{.GoType}}
	{{end}}{{end}}

	// define the select query
	var query = "{{.ParentTable.GenericSelectQuery}} WHERE {{range $i, $e := .ParentTable.PKColumns}}{{.DbName}} = ${{print (plus1 $i)}}{{if ne (plus1 $i) $pkColCount}} AND {{end}}{{end}}";

	// we are aiming for a single row so we will use Query Row	
	err = txWrapper.Tx.QueryRow(query, ` +
	`{{range $i, $e := .ParentTable.PKColumns}}input{{$e.GoName}}{{if ne (plus1 $i) $pkColCount}},{{end}}{{end}}` +
	`).Scan({{range $i, $e := .ParentTable.Columns}}&param{{$e.GoName}}{{if ne (plus1 $i) $colCount}},{{end}}{{end}})
    switch {
    case err == ErrNoRows:
            // no such row found, return nil and nil
			return nil, nil
    case err != nil:
            return nil, NewModelsError(errorPrefix + "fatal error running the query:", err)
    default:
           	// create the return structure as a pointer of the type
			returnStruct = &{{.ParentTable.GoFriendlyName}}{
				{{range .ParentTable.Columns}}{{if not .Nullable}}{{.GoName}}: param{{.GoName}},
				{{end}}{{end}}
			}
			{{range $e := .ParentTable.Columns}}{{if $e.Nullable}}returnStruct.Set{{.GoName}}(param{{$e.GoName}}.{{getNullableTypeValueFieldName $e.GoNullableType}}, boolFromStatus(param{{$e.GoName}}.Status))
			{{end}}{{end}}			
			// return the structure
			return returnStruct, nil
    }			
}
`

/* BEGIN: Unique Constraints Getter Templates */

const UQ_GETTER_TEMPLATE_ATOMIC = `{{$colCount := len .ParentTable.Columns}}{{$uqColCount := len .Columns}}{{$functionName := "GetByUnique"}}
// Queries the database for a single row based on the specified single or multi-column unique constraint.
// Returns a pointer to a {{.ParentTable.GoFriendlyName}} structure if a record was found,
// otherwise it returns nil.
func (utilRef *t{{.ParentTable.GoFriendlyName}}Utils) {{$functionName}}` +
	`{{if gt $uqColCount 1}}` +
	`{{range $i, $e := .Columns}}{{$e.GoName}}{{if ne (plus1 $i) $uqColCount}}And{{end}}{{end}}(` +
	`{{else}}{{range $i, $e := .Columns}}{{$e.GoName}}{{end}}(` +
	`{{end}}` +
	`{{range $i, $e := .Columns}}input{{$e.GoName}} {{$e.GoType}} {{if ne (plus1 $i) $uqColCount}},{{end}}{{end}})` +
	` (returnStruct *{{.ParentTable.GoFriendlyName}}, err error) {
	
	returnStruct = nil
	err = nil
	
	var errorPrefix = "{{.ParentTable.GoFriendlyName}}GetByUnique ERROR: "
	
	currentDbHandle := GetDb()
	if currentDbHandle == nil {
		return nil, NewModelsErrorLocal(errorPrefix, "the database handle is nil")
	}

	// define receiving params for the row iteration
	{{range $e := .ParentTable.Columns}}{{if .Nullable}}var param{{.GoName}} {{$e.GoNullableType}}
	{{else}}var param{{.GoName}} {{.GoType}}
	{{end}}{{end}}

	// define the select query
	var query = "{{.ParentTable.GenericSelectQuery}} WHERE {{range $i, $e := .Columns}}{{.DbName}} = ${{print (plus1 $i)}}{{if ne (plus1 $i) $uqColCount}} AND {{end}}{{end}}";

	// we are aiming for a single row so we will use Query Row	
	err = currentDbHandle.QueryRow(query, ` +
	`{{range $i, $e := .Columns}}input{{$e.GoName}}{{if ne (plus1 $i) $uqColCount}},{{end}}{{end}}` +
	`).Scan({{range $i, $e := .ParentTable.Columns}}&param{{$e.GoName}}{{if ne (plus1 $i) $colCount}},{{end}}{{end}})
    switch {
    case err == ErrNoRows:
            // no such row found, return nil and nil
			return nil, nil
    case err != nil:
            return nil, NewModelsError(errorPrefix + "fatal error running the query:", err)
    default:
           	// create the return structure as a pointer of the type
			returnStruct = &{{.ParentTable.GoFriendlyName}}{
				{{range .ParentTable.Columns}}{{if not .Nullable}}{{.GoName}}: param{{.GoName}},
				{{end}}{{end}}
			}
			{{range $e := .ParentTable.Columns}}{{if $e.Nullable}}returnStruct.Set{{.GoName}}(param{{$e.GoName}}.{{getNullableTypeValueFieldName $e.GoNullableType}}, boolFromStatus(param{{$e.GoName}}.Status))
			{{end}}{{end}}			
			// return the structure
			return returnStruct, nil
    }			
}
`

const UQ_GETTER_TEMPLATE_TX = `{{$colCount := len .ParentTable.Columns}}{{$uqColCount := len .Columns}}{{$functionName := print "Get" .ParentTable.GoFriendlyName "ByUnique"}}
// Queries the database for a single row based on the specified single or multi-column unique constraints.
// Returns a pointer to a {{.ParentTable.GoFriendlyName}} structure if a record was found,
// otherwise it returns nil.
func (txWrapper *Transaction) {{$functionName}}` +
	`{{if gt $uqColCount 1}}` +
	`{{range $i, $e := .Columns}}{{$e.GoName}}{{if ne (plus1 $i) $uqColCount}}And{{end}}{{end}}(` +
	`{{else}}{{range $i, $e := .Columns}}{{$e.GoName}}{{end}}(` +
	`{{end}}` +
	`{{range $i, $e := .Columns}}input{{$e.GoName}} {{$e.GoType}} {{if ne (plus1 $i) $uqColCount}},{{end}}{{end}})` +
	` (returnStruct *{{.ParentTable.GoFriendlyName}}, err error) {
	
	returnStruct = nil
	err = nil
	
	var errorPrefix = "{{.ParentTable.GoFriendlyName}}GetByUniqueTx ERROR: "
	
	if txWrapper == nil { return nil, NewModelsErrorLocal(errorPrefix, "the transaction wrapper is nil") }
	if txWrapper.Tx == nil { return nil, NewModelsErrorLocal(errorPrefix, "the transaction object is nil") }	

	// define receiving params for the row iteration
	{{range $e := .ParentTable.Columns}}{{if .Nullable}}var param{{.GoName}} {{$e.GoNullableType}}
	{{else}}var param{{.GoName}} {{.GoType}}
	{{end}}{{end}}

	// define the select query
	var query = "{{.ParentTable.GenericSelectQuery}} WHERE {{range $i, $e := .Columns}}{{.DbName}} = ${{print (plus1 $i)}}{{if ne (plus1 $i) $uqColCount}} AND {{end}}{{end}}";

	// we are aiming for a single row so we will use Query Row	
	err = txWrapper.Tx.QueryRow(query, ` +
	`{{range $i, $e := .Columns}}input{{$e.GoName}}{{if ne (plus1 $i) $uqColCount}},{{end}}{{end}}` +
	`).Scan({{range $i, $e := .ParentTable.Columns}}&param{{$e.GoName}}{{if ne (plus1 $i) $colCount}},{{end}}{{end}})
    switch {
    case err == ErrNoRows:
            // no such row found, return nil and nil
			return nil, nil
    case err != nil:
            return nil, NewModelsError(errorPrefix + "fatal error running the query:", err)
    default:
           	// create the return structure as a pointer of the type
			returnStruct = &{{.ParentTable.GoFriendlyName}}{
				{{range .ParentTable.Columns}}{{if not .Nullable}}{{.GoName}}: param{{.GoName}},
				{{end}}{{end}}
			}
			{{range $e := .ParentTable.Columns}}{{if $e.Nullable}}returnStruct.Set{{.GoName}}(param{{$e.GoName}}.{{getNullableTypeValueFieldName $e.GoNullableType}}, boolFromStatus(param{{$e.GoName}}.Status))
			{{end}}{{end}}			
			// return the structure
			return returnStruct, nil
    }			
}
`
