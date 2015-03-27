package main

/* Select Functions Templates */

/* ************************************************ */
/* BEGIN: Atomic (non-transaction) Select Templates */
/* ************************************************ */

const COMMON_CODE_SELECT_QUERY_WHERE = `if err != nil {
		return nil, NewModelsError(errorPrefix + " fatal error running the query:", err)
	}
	defer rows.Close()

	var sliceOf{{.GoFriendlyName}} []{{.GoFriendlyName}}

	// BEGIN: if any nullable fields, create temporary nullable variables to receive null values
	{{range $i, $e := .Columns}}{{if .Nullable}}var nullable{{$e.GoName}} {{$e.GoNullableType}} 
	{{end}}{{end}}
	// END: if any nullable fields, create temporary nullable variables to receive null values

	for rows.Next() {

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
		
		sliceOf{{.GoFriendlyName}} = append(sliceOf{{.GoFriendlyName}}, current{{.GoFriendlyName}})

	}
	err = rows.Err()
	if err != nil {
		return nil, NewModelsError(errorPrefix + " error during rows.Next() iterations:", err)
	}
`

const COMMON_CODE_SELECT_TEMPLATE_WHERE_ATOMIC = `
	currentDbHandle := GetDb()
	if currentDbHandle == nil {
		return nil, NewModelsErrorLocal(errorPrefix, "the database handle is nil")
	}

	// define the select query
	var queryParts []string
	
	queryParts = append(queryParts, "{{.GenericSelectQuery}} WHERE ")
	queryParts = append(queryParts, condition)
		
	rows, err := currentDbHandle.Query(JoinStringParts(queryParts,""), params...)	
` + COMMON_CODE_SELECT_QUERY_WHERE

const COMMON_CODE_SELECT_TEMPLATE_WHERE_ATOMIC_PAGED = `
	currentDbHandle := GetDb()
	if currentDbHandle == nil {
		return nil, NewModelsErrorLocal(errorPrefix, "the database handle is nil")
	}

	// define the select query
	var queryParts []string
	
	queryParts = append(queryParts, "{{.GenericSelectQuery}} WHERE ")
	queryParts = append(queryParts, condition)
	
	// apply the pagination filter
	var pageLimit string = Itoa(pageSize)
	var pageOffset string = "0"
	var enablePageOffset bool = false
	
	if pageNumber > 1 {
		pageOffset = Itoa(pageSize * (pageNumber - 1))
		enablePageOffset = true
	}
	
	queryParts = append(queryParts, " LIMIT ")
	queryParts = append(queryParts, pageLimit)
	
	if enablePageOffset {
		queryParts = append(queryParts, " OFFSET ")
		queryParts = append(queryParts, pageOffset)
	}	
		
	rows, err := currentDbHandle.Query(JoinStringParts(queryParts,""), params...)

` + COMMON_CODE_SELECT_QUERY_WHERE

const COMMON_CODE_SELECT_TEMPLATE_WHERE_PAGED_CONDITION_HEADER = `
	if condition == "" {
		return nil, NewModelsErrorLocal(errorPrefix, "No condition specified. Please use SelectAll method to select all rows from {{.DbName}}")
	}
	
	if pageSize < 1 {
		return nil, NewModelsErrorLocal(errorPrefix, "The pageSize parameter must be greater than or equal to 1")
	}
	if pageNumber < 1 {
		return nil, NewModelsErrorLocal(errorPrefix, "The pageNumber parameter must be greater than or equal to 1")
	}
`

const COMMON_CODE_SELECT_TEMPLATE_WHERE_CACHED_HEADER_NON_PAGED = `
	var whereClauseHash string = ""
	var hashErr error = nil	
	
	if cacheOption > FLAG_CACHE_DISABLE {
		
		if cacheOption == FLAG_CACHE_USE || cacheOption == FLAG_CACHE_RELOAD {
			
			whereClauseHash, hashErr = GetHashFromConditionAndParams(condition, params...)
			if hashErr != nil {
				return nil, NewModelsError(errorPrefix + "GetHashFromConditionAndParams() error:",hashErr)
			}			
		}
		
		// check the caching options - in case it's FLAG_CACHE_USE and there is cache available no need to go further
		if cacheOption == FLAG_CACHE_USE {
						
			// try to get the rows from cache, if enabled and valid
			if current{{.GoFriendlyName}}RowsFromCache, cacheValid := utilRef.Cache.GetWhere(whereClauseHash) ; cacheValid == true {
				
				return current{{.GoFriendlyName}}RowsFromCache, nil
			} 
		}
	}
`

const COMMON_CODE_SELECT_TEMPLATE_WHERE_CACHED_HEADER_PAGED = `
	var whereClauseHash string = ""
	var hashErr error = nil	

	if cacheOption > FLAG_CACHE_DISABLE {
		
		if cacheOption == FLAG_CACHE_USE || cacheOption == FLAG_CACHE_RELOAD {
			
			whereClauseHash, hashErr = GetHashFromConditionAndParams(condition, params...)
			if hashErr != nil {
				return nil, NewModelsError(errorPrefix + "GetHashFromConditionAndParams() error:",hashErr)
			}
			
			// because this is a pagination-based method, we need to append the pageSize and pageNum to the cache key
			var whereClauseHashPaginated []string = []string {whereClauseHash}
			whereClauseHashPaginated = append(whereClauseHashPaginated, "-pageSize:")
			whereClauseHashPaginated = append(whereClauseHashPaginated, Itoa(pageSize))
			whereClauseHashPaginated = append(whereClauseHashPaginated, "-pageNumber:")
			whereClauseHashPaginated = append(whereClauseHashPaginated,  Itoa(pageNumber))

			whereClauseHash = JoinStringParts(whereClauseHashPaginated,"")		
			
		}
		
		// check the caching options - in case it's FLAG_CACHE_USE and there is cache available no need to go further
		if cacheOption == FLAG_CACHE_USE {
						
			// try to get the rows from cache, if enabled and valid
			if current{{.GoFriendlyName}}RowsFromCache, cacheValid := utilRef.Cache.GetWhere(whereClauseHash) ; cacheValid == true {
				
				return current{{.GoFriendlyName}}RowsFromCache, nil
			} 
		}
	}
`

const COMMON_CODE_SELECT_TEMPLATE_WHERE_CACHED_FOOTER = `
	// before returning the result, make sure to insert it into cache if instructed
	if cacheOption > FLAG_CACHE_DISABLE && whereClauseHash != "" {
		
		if cacheOption != FLAG_CACHE_DELETE {			
			utilRef.Cache.SetWhere(whereClauseHash, sliceOf{{.GoFriendlyName}})	
		}
	}
`

const SELECT_TEMPLATE_WHERE = `{{$colCount := len .Columns}}
{{$functionName := "Select"}}{{$sourceStructName := print "source" .GoFriendlyName}}
// Returns the rows from {{.DbName}}, corresponding to the supplied condition 
// and the respective parameters. The condition must not include the WHERE keyword.
// This version is not cached and calls the database directly.
// The rows are converted to a slice of {{.GoFriendlyName}} instances
// If operation fails, it returns nil and the error.
func (utilRef *t{{.GoFriendlyName}}Utils) {{$functionName}}(condition string, params ...interface{}) ([]{{.GoFriendlyName}},  error) {
						
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "

	if condition == "" {
		return nil, NewModelsErrorLocal(errorPrefix, "No condition specified. Please use SelectAll method to select all rows from {{.DbName}}")
	}
	
	` + COMMON_CODE_SELECT_TEMPLATE_WHERE_ATOMIC + `		
	
	return sliceOf{{.GoFriendlyName}}, nil
}

{{$colCount := len .Columns}}
{{$functionName := "SelectCached"}}{{$sourceStructName := print "source" .GoFriendlyName}}
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
	
	` + COMMON_CODE_SELECT_TEMPLATE_WHERE_CACHED_HEADER_NON_PAGED + `
	
	` + COMMON_CODE_SELECT_TEMPLATE_WHERE_ATOMIC + `	
	
	` + COMMON_CODE_SELECT_TEMPLATE_WHERE_CACHED_FOOTER + `
	
	return sliceOf{{.GoFriendlyName}}, nil
}

{{$colCount := len .Columns}}
{{$functionName := "SelectPage"}}{{$sourceStructName := print "source" .GoFriendlyName}}
// Returns the paginated rows from {{.DbName}}, corresponding to the supplied condition
// and the respective numbered parameters. The condition must not include the WHERE keyword.
// The pageSize parameter (must be greater than or equal to 1) indicates how many maximum records are to be returned.
// The pageNumber parameter (must be greater than or equal to 1) indicates the page offset from the beginning of the resultset.
// If pageNumber is 1, there is no offset.
// This version is not cached and calls the database directly.
// The rows are converted to a slice of {{.GoFriendlyName}} instances
// If operation fails, it returns nil and the error.
func (utilRef *t{{.GoFriendlyName}}Utils) {{$functionName}}(pageSize int, pageNumber int, condition string, params ...interface{}) ([]{{.GoFriendlyName}},  error) {
						
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "

	` + COMMON_CODE_SELECT_TEMPLATE_WHERE_PAGED_CONDITION_HEADER + `

	if condition == "" {
		return nil, NewModelsErrorLocal(errorPrefix, "No condition specified. Please use SelectAll method to select all rows from {{.DbName}}")
	}	
	
	` + COMMON_CODE_SELECT_TEMPLATE_WHERE_ATOMIC_PAGED + `
	
	return sliceOf{{.GoFriendlyName}}, nil
}

{{$colCount := len .Columns}}
{{$functionName := "SelectPageCached"}}{{$sourceStructName := print "source" .GoFriendlyName}}
// Returns the rows from {{.DbName}}, corresponding to the supplied condition 
// and the respective numbered parameters. The condition must not include the WHERE keyword.
// The pageSize parameter (must be greater than or equal to 1) indicates how many maximum records are to be returned.
// The pageNumber parameter (must be greater than or equal to 1) indicates the page offset from the beginning of the resultset.
// If pageNumber is 1, there is no offset.
// The cacheOption parameter is one of the FLAG_CACHE_[behaviour] global integer constants.
// FLAG_CACHE_DISABLE has a value of 0, and caching is completely bypassed.
// The rows are converted to a slice of {{.GoFriendlyName}} instances
// If operation fails, it returns nil and the error.
func (utilRef *t{{.GoFriendlyName}}Utils) {{$functionName}}(pageSize int, pageNumber int, cacheOption int, condition string, params ...interface{}) ([]{{.GoFriendlyName}},  error) {
						
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "

	` + COMMON_CODE_SELECT_TEMPLATE_WHERE_PAGED_CONDITION_HEADER + `	
	` + COMMON_CODE_SELECT_TEMPLATE_WHERE_CACHED_HEADER_PAGED + `	
	` + COMMON_CODE_SELECT_TEMPLATE_WHERE_ATOMIC_PAGED + `		
	` + COMMON_CODE_SELECT_TEMPLATE_WHERE_CACHED_FOOTER + `
	
	return sliceOf{{.GoFriendlyName}}, nil
}

`

/* ***************************************** */
/* BEGIN: Transaction based Select Templates */
/* ***************************************** */

const COMMON_CODE_SELECT_TEMPLATE_WHERE_TRANSACTION = `
	if txWrapper == nil { return nil, NewModelsErrorLocal(errorPrefix, "the transaction wrapper is nil") }
	if txWrapper.Tx == nil { return nil, NewModelsErrorLocal(errorPrefix, "the transaction object is nil") }	

	// define the select query
	var queryParts []string
	
	queryParts = append(queryParts, "{{.GenericSelectQuery}} WHERE ")
	queryParts = append(queryParts, condition)	
		
	rows, err := txWrapper.Tx.Query(JoinStringParts(queryParts,""), params...)

` + COMMON_CODE_SELECT_QUERY_WHERE

const COMMON_CODE_SELECT_TEMPLATE_WHERE_TRANSACTION_PAGED = `
	if txWrapper == nil { return nil, NewModelsErrorLocal(errorPrefix, "the transaction wrapper is nil") }
	if txWrapper.Tx == nil { return nil, NewModelsErrorLocal(errorPrefix, "the transaction object is nil") }	

	// define the select query
	var queryParts []string
	
	queryParts = append(queryParts, "{{.GenericSelectQuery}} WHERE ")
	queryParts = append(queryParts, condition)
	
	// apply the pagination filter
	var pageLimit string = Itoa(pageSize)
	var pageOffset string = "0"
	var enablePageOffset bool = false
	
	if pageNumber > 1 {
		pageOffset = Itoa(pageSize * (pageNumber - 1))
		enablePageOffset = true
	}
	
	queryParts = append(queryParts, " LIMIT ")
	queryParts = append(queryParts, pageLimit)
	
	if enablePageOffset {
		queryParts = append(queryParts, " OFFSET ")
		queryParts = append(queryParts, pageOffset)
	}	
		
	rows, err := txWrapper.Tx.Query(JoinStringParts(queryParts,""), params...)

` + COMMON_CODE_SELECT_QUERY_WHERE

const SELECT_TEMPLATE_WHERE_TX = `{{$colCount := len .Columns}}
{{$functionName := print "Select" .GoFriendlyName}}{{$sourceStructName := print "source" .GoFriendlyName}}
// Returns the rows from {{.DbName}}, corresponding to the supplied condition 
// and the respective parameters. The condition must not include the WHERE keyword.
// This version is not cached and calls the database directly.
// The rows are converted to a slice of {{.GoFriendlyName}} instances
// If operation fails, it returns nil and the error.
func (txWrapper *Transaction) {{$functionName}}(condition string, params ...interface{}) ([]{{.GoFriendlyName}},  error) {
						
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "

	if condition == "" {
		return nil, NewModelsErrorLocal(errorPrefix, "No condition specified. Please use SelectAll method to select all rows from {{.DbName}}")
	}

	` + COMMON_CODE_SELECT_TEMPLATE_WHERE_TRANSACTION + `

	return sliceOf{{.GoFriendlyName}}, nil
}

{{$colCount := len .Columns}}
{{$functionName := print "SelectCached" .GoFriendlyName}}{{$sourceStructName := print "source" .GoFriendlyName}}
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
	
	var utilRef *t{{.GoFriendlyName}}Utils

	` + COMMON_CODE_SELECT_TEMPLATE_WHERE_CACHED_HEADER_NON_PAGED + `
	` + COMMON_CODE_SELECT_TEMPLATE_WHERE_TRANSACTION + `
	` + COMMON_CODE_SELECT_TEMPLATE_WHERE_CACHED_FOOTER + `	
	
	return sliceOf{{.GoFriendlyName}}, nil
}

{{$colCount := len .Columns}}
{{$functionName := print "SelectPage" .GoFriendlyName}}{{$sourceStructName := print "source" .GoFriendlyName}}
// Returns the paginated rows from {{.DbName}}, corresponding to the supplied condition 
// and the respective numbered parameters. The condition must not include the WHERE keyword.
// The pageSize parameter (must be greater than or equal to 1) indicates how many maximum records are to be returned.
// The pageNumber parameter (must be greater than or equal to 1) indicates the page offset from the beginning of the resultset.
// If pageNumber is 1, there is no offset.
// This version is not cached and calls the database directly.
// The rows are converted to a slice of {{.GoFriendlyName}} instances
// If operation fails, it returns nil and the error.
func (txWrapper *Transaction) {{$functionName}}(pageSize int, pageNumber int, condition string, params ...interface{}) ([]{{.GoFriendlyName}},  error) {
						
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "

	` + COMMON_CODE_SELECT_TEMPLATE_WHERE_PAGED_CONDITION_HEADER + `

	if condition == "" {
		return nil, NewModelsErrorLocal(errorPrefix, "No condition specified. Please use SelectAll method to select all rows from {{.DbName}}")
	}

	` + COMMON_CODE_SELECT_TEMPLATE_WHERE_TRANSACTION_PAGED + `

	return sliceOf{{.GoFriendlyName}}, nil
}

{{$colCount := len .Columns}}
{{$functionName := print "SelectPageCached" .GoFriendlyName}}{{$sourceStructName := print "source" .GoFriendlyName}}
// Returns the paginated rows from {{.DbName}}, corresponding to the supplied condition 
// and the respective numbered parameters. The condition must not include the WHERE keyword.
// The pageSize parameter (must be greater than or equal to 1) indicates how many maximum records are to be returned.
// The pageNumber parameter (must be greater than or equal to 1) indicates the page offset from the beginning of the resultset.
// If pageNumber is 1, there is no offset.
// The cacheOption parameter is one of the FLAG_CACHE_[behaviour] global integer constants.
// FLAG_CACHE_DISABLE has a value of 0, and caching is completely bypassed.
// The rows are converted to a slice of {{.GoFriendlyName}} instances
// If operation fails, it returns nil and the error.
func (txWrapper *Transaction) {{$functionName}}(pageSize int, pageNumber int, cacheOption int, condition string, params ...interface{}) ([]{{.GoFriendlyName}},  error) {
						
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "

	var utilRef *t{{.GoFriendlyName}}Utils

	` + COMMON_CODE_SELECT_TEMPLATE_WHERE_PAGED_CONDITION_HEADER + `
	` + COMMON_CODE_SELECT_TEMPLATE_WHERE_CACHED_HEADER_PAGED + `
	` + COMMON_CODE_SELECT_TEMPLATE_WHERE_TRANSACTION_PAGED + `
	` + COMMON_CODE_SELECT_TEMPLATE_WHERE_CACHED_FOOTER + `

	return sliceOf{{.GoFriendlyName}}, nil
}

`

/* **************************************************** */
/* BEGIN: Atomic (non-transaction) Select All Templates */
/* **************************************************** */

const COMMON_CODE_SELECT_ALL_QUERY = `if err != nil {
		return nil, NewModelsError(errorPrefix + " fatal error running the query:", err)
	}
	defer rows.Close()
	
	var sliceOf{{.GoFriendlyName}} []{{.GoFriendlyName}}
	
	// BEGIN: if any nullable fields, create temporary nullable variables to receive null values
	{{range $i, $e := .Columns}}{{if .Nullable}}var nullable{{$e.GoName}} {{$e.GoNullableType}} 
	{{end}}{{end}}
	// END: if any nullable fields, create temporary nullable variables to receive null values

	for rows.Next() {

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
		
		sliceOf{{.GoFriendlyName}} = append(sliceOf{{.GoFriendlyName}}, current{{.GoFriendlyName}})

	}
	err = rows.Err()
	if err != nil {
		return nil, NewModelsError(errorPrefix + " error during rows.Next() iterations:", err)
	}	
`

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
	
	rows, err := currentDbHandle.Query("{{.GenericSelectQuery}}")

	` + COMMON_CODE_SELECT_ALL_QUERY + `
		
	return sliceOf{{.GoFriendlyName}}, nil
}

{{$colCount := len .Columns}}
{{$functionName := "SelectAllOrderBy"}}{{$sourceStructName := print "source" .GoFriendlyName}}
// Returns all the rows from {{.DbName}} ordered by the field names specified in the orderBy parameter.
// The orderBy parameter must not contain the 'ORDER BY' keywords, and it can be empty, 
// in which case the results are unpredictable.
// The rows are converted to a slice of {{.GoFriendlyName}} instances
// If operation fails, it returns nil and the error.
func (utilRef *t{{.GoFriendlyName}}Utils) {{$functionName}}(orderBy string) ([]{{.GoFriendlyName}},  error) {
						
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "

	currentDbHandle := GetDb()
	if currentDbHandle == nil {
		return nil, NewModelsErrorLocal(errorPrefix, "the database handle is nil")
	}

	// try to get the rows from cache, if enabled and valid
	if all{{.GoFriendlyName}}RowsFromCache, cacheValid := utilRef.Cache.GetAllRows() ; cacheValid == true {		
		return all{{.GoFriendlyName}}RowsFromCache, nil
	}

	// define the select query
	var queryParts []string
	
	queryParts = append(queryParts, "{{.GenericSelectQuery}} ")
	
	if orderBy != "" {
		queryParts = append(queryParts, " ORDER BY ")
		queryParts = append(queryParts, orderBy)
	}
		
	rows, err := currentDbHandle.Query(JoinStringParts(queryParts,""))

	` + COMMON_CODE_SELECT_ALL_QUERY + `
	
	return sliceOf{{.GoFriendlyName}}, nil
}

{{$colCount := len .Columns}}
{{$functionName := "SelectAllPage"}}{{$sourceStructName := print "source" .GoFriendlyName}}
// Returns a page of rows from {{.DbName}} equal to pageSize, 
// with the appropriate offset determined by pageNumber. 
// The orderBy parameter must not contain the 'ORDER BY' keywords, and it can be empty, 
// in which case the results are unpredictable.
// The rows are converted to a slice of {{.GoFriendlyName}} instances
// If operation succeeds, it returns the page-restricted rows, and nil as error.
// If operation fails, it returns nil and the error.
func (utilRef *t{{.GoFriendlyName}}Utils) {{$functionName}}(pageSize int, pageNumber int, orderBy string) ([]{{.GoFriendlyName}},  error) {
						
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "

	if pageSize < 1 {
		return nil, NewModelsErrorLocal(errorPrefix, "The pageSize parameter must be greater than or equal to 1")
	}
	if pageNumber < 1 {
		return nil, NewModelsErrorLocal(errorPrefix, "The pageNumber parameter must be greater than or equal to 1")
	}

	currentDbHandle := GetDb()
	if currentDbHandle == nil {
		return nil, NewModelsErrorLocal(errorPrefix, "the database handle is nil")
	}

	// try to get the rows from cache, if enabled and valid
	if all{{.GoFriendlyName}}RowsFromCache, cacheValid := utilRef.Cache.GetAllRows() ; cacheValid == true {		
				
		if pageNumber == 1 {			
			return all{{.GoFriendlyName}}RowsFromCache[:pageSize], nil
		}
		
		return all{{.GoFriendlyName}}RowsFromCache[((pageNumber-1)*pageSize):((pageNumber-1)*pageSize)+pageSize], nil
		
	}
	
	// define the select query
	var queryParts []string
	
	queryParts = append(queryParts, "{{.GenericSelectQuery}} ")
	
	if orderBy != "" {
		queryParts = append(queryParts, " ORDER BY ")
		queryParts = append(queryParts, orderBy)
	}
	
	// apply the pagination filter
	var pageLimit string = Itoa(pageSize)
	var pageOffset string = "0"
	var enablePageOffset bool = false
	
	if pageNumber > 1 {
		pageOffset = Itoa(pageSize * (pageNumber - 1))
		enablePageOffset = true
	}
	
	queryParts = append(queryParts, " LIMIT ")
	queryParts = append(queryParts, pageLimit)
	
	if enablePageOffset {
		queryParts = append(queryParts, " OFFSET ")
		queryParts = append(queryParts, pageOffset)
	}
		
	rows, err := currentDbHandle.Query(JoinStringParts(queryParts,""))	

	` + COMMON_CODE_SELECT_ALL_QUERY + `
	
	return sliceOf{{.GoFriendlyName}}, nil
}
`

/* ********************************************* */
/* BEGIN: Transaction based Select All Templates */
/* ********************************************* */

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
	
	rows, err := txWrapper.Tx.Query("{{.GenericSelectQuery}}")

	` + COMMON_CODE_SELECT_ALL_QUERY + `
	
	return sliceOf{{.GoFriendlyName}}, nil
}
`
