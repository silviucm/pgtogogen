package main

import "text/template"

/* Template helper functions */
var fns = template.FuncMap{
	"plus1": func(x int) int {
		return x + 1
	},
	"plus": func(x int, y int) int {
		return x + y
	},
}

/* Tables */

const TABLE_TEMPLATE = `package {{.Options.PackageName}}

/* *********************************************************** */
/* This file was automatically generated by pgtogogen.         */
/* Do not modify this file unless you know what you are doing. */
/* *********************************************************** */

import (
	"bytes"
	"net/http"
	"sync"
	{{range $key, $value := .GoTypesToImport}}"{{$value}}"
	{{end}}	
)

const {{.GoFriendlyName}}_DB_TABLE_NAME string = "{{.DbName}}"

{{if ne .DbComments ""}}/* {{.DbComments}} */{{end}}
type {{.GoFriendlyName}} struct {
	{{range .Columns}}{{if ne .DbComments ""}}/* {{.DbComments}} */
	{{.GoName}} {{.GoType}} // IsPK: {{.IsPK}} , IsCompositePK: {{.IsCompositePK}}, IsFK: {{.IsFK}}
	{{else}}{{.GoName}} {{.GoType}} // IsPK: {{.IsPK}} , IsCompositePK: {{.IsCompositePK}}, IsFK: {{.IsFK}}{{end}}
	{{end}}	
	
	// Set this to true if you want Inserts to ignore the PK fields	
	PgToGo_IgnorePKValuesWhenInsertingAndUseSequence bool 

	// Set this to true if you want New or Create operations to automatically
	// set all time.Time (datetime) fields to time.Now()
	PgToGo_SetDateTimeFieldsToNowForNewRecords bool 

	// Set this to true if you want New or Create operations to automatically
	// set all Guid fields to a new guid
	PgToGo_SetGuidFieldsToNewGuidsNewRecords bool
	
}

{{ $tableGoName := .GoFriendlyName}}
/* Sorting helper containers */
{{range $i, $e := .Columns}}
// By{{$e.GoName}} implements sort.Interface for []{{$tableGoName}} based on
// the {{$e.GoName}} field. Usage: sort.Sort(Sort{{$tableGoName}}By{{$e.GoName}}(anyGiven{{$tableGoName}}Slice))
type Sort{{$tableGoName}}By{{$e.GoName}} []{{$tableGoName}}

func (a Sort{{$tableGoName}}By{{$e.GoName}}) Len() int           { return len(a) }
func (a Sort{{$tableGoName}}By{{$e.GoName}}) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Sort{{$tableGoName}}By{{$e.GoName}}) Less(i, j int) bool { return LessComparatorFor_{{$e.GoType}}(a[i].{{$e.GoName}},a[j].{{$e.GoName}}) }
{{end}}

// Utility-oriented, internal type to allow a singleton structure that would hold static-like methods
// and global, single-instance settings
type t{{.GoFriendlyName}}Utils struct {
	
	utilMutex sync.RWMutex
	
	// instance of a CacheFor{{.GoFriendlyName}} structure
	Cache CacheFor{{.GoFriendlyName}}
}

{{$colCount := len .Columns}}{{$functionName := "New"}}
// Creates a new pointer to a blank KiriUser structure.
// Some of the fields, such as the time.Time ones, might be already set to time.Now()
// based on the Tables.PgToGo_SetDateTimeFieldsToNowForNewRecords setting
func (utilRef *t{{.GoFriendlyName}}Utils) {{$functionName}}() *{{.GoFriendlyName}} {
	
	
	{{$structInstanceName := print "new" .GoFriendlyName}}{{$structInstanceName}} := &{{.GoFriendlyName}}{}		
	
	{{$structInstanceName}}.CloneGlobalSettings()
	
	return {{$structInstanceName}}
}

{{$colCount := len .Columns}}{{$functionName := "CreateFromHttpRequest"}}
// Creates a new pointer to a KiriUser from an Http Request.
// The parameters are expected to match the struct field names
func (utilRef *t{{.GoFriendlyName}}Utils) {{$functionName}}(req *http.Request) (*{{.GoFriendlyName}}, error) {
	
	var errorPrefix = "{{.GoFriendlyName}}Utils.{{$functionName}}() ERROR: "
	
	if req == nil {
		return nil, NewModelsErrorLocal(errorPrefix, "The *http.Request parameter provided was nil.")
	}	
	
	var err error = nil
	{{$structInstanceName := print "new" .GoFriendlyName}}{{$structInstanceName}} := &{{.GoFriendlyName}}{}
	
	{{$structInstanceName}}.CloneGlobalSettings()
	
	{{range $i, $e := .Columns}}{{if eq $e.GoType "time.Time"}}{{$structInstanceName}}.{{$e.GoName}}, err = To_Time_FromString(req.FormValue("{{$e.GoName}}"))	
	{{else if eq $e.GoType "string"}}{{$structInstanceName}}.{{$e.GoName}} = req.FormValue("{{$e.GoName}}")
	{{else}}{{$structInstanceName}}.{{$e.GoName}}, err = To_{{$e.GoType}}_FromString(req.FormValue("{{$e.GoName}}"))
	{{end}}if err != nil { return nil, NewModelsError(errorPrefix, err) }
	{{end}}

	return {{$structInstanceName}}, nil
}

{{$colCount := len .Columns}}{{$functionName := "CreateFromHttpRequestIgnoreErrors"}}
// Creates a new pointer to a KiriUser from an Http Request.
// The parameters are expected to match the struct field names
// Unlike CreateFromHttpRequest, this method completely ignores parsing errors, 
// so you will have to call Validate() on the structure if that structure has such a method.
func (utilRef *t{{.GoFriendlyName}}Utils) {{$functionName}}(req *http.Request) *{{.GoFriendlyName}} {
	
	{{$structInstanceName := print "new" .GoFriendlyName}}{{$structInstanceName}} := &{{.GoFriendlyName}}{}
	
	{{$structInstanceName}}.CloneGlobalSettings()
	
	{{range $i, $e := .Columns}}{{if eq $e.GoType "time.Time"}}{{$structInstanceName}}.{{$e.GoName}}, _ = To_Time_FromString(req.FormValue("{{$e.GoName}}"))	
	{{else if eq $e.GoType "string"}}{{$structInstanceName}}.{{$e.GoName}} = req.FormValue("{{$e.GoName}}")
	{{else}}{{$structInstanceName}}.{{$e.GoName}}, _ = To_{{$e.GoType}}_FromString(req.FormValue("{{$e.GoName}}"))
	{{end}}
	{{end}}

	return {{$structInstanceName}}
}

{{$colCount := len .Columns}}{{$functionName := "CloneGlobalSettings"}}{{$structInstanceName := print "instance" .GoFriendlyName}}
// Assigns the global settings for operations to the control fields of this instance
// An example would be:
// {{$structInstanceName}}.PgToGo_IgnorePKValuesWhenInsertingAndUseSequence = Tables.PgToGo_IgnorePKValuesWhenInsertingAndUseSequence
func (instance *{{.GoFriendlyName}}) {{$functionName}}() {
			
	
	instance.PgToGo_IgnorePKValuesWhenInsertingAndUseSequence = Tables.PgToGo_IgnorePKValuesWhenInsertingAndUseSequence
	instance.PgToGo_SetDateTimeFieldsToNowForNewRecords = Tables.PgToGo_SetDateTimeFieldsToNowForNewRecords
	instance.PgToGo_SetGuidFieldsToNewGuidsNewRecords = Tables.PgToGo_SetGuidFieldsToNewGuidsNewRecords
	
	
}

// Returns the database field name, regardless whether the Go name or the db name was provided.
// If no field was found, return empty string.
func (utilRef *t{{.GoFriendlyName}}Utils) ToDbFieldName(fieldDbOrGoName string) string {
	
	{{range $i, $e := .Columns}}if fieldDbOrGoName == "{{$e.GoName}}" || fieldDbOrGoName == "{{$e.DbName}}" { return "{{$e.DbName}}" }			
	{{end}}

	return ""
}

`

const TABLE_TEMPLATE_CUSTOM = `package {{.Options.PackageName}}

/* *********************************************************** **/
/* This file is generated by pgtogogen FIRST-TIME ONLY.         */
/* It will not subsequently overwrite it if it already exists.  */
/* Use this file to create your custom extension functionality. */
/* ************************************************************ */

/*
import (
	{{range $key, $value := .GoTypesToImport}}"{{$value}}"
	{{end}}	
)
*/

// Implements the Validator interface. 
func (t *{{.GoFriendlyName}}) Validate() (bool, []error) {

	// Returns true for now. 
	// Todo: modify as needed
	return true, nil

}	

`

const TABLE_TEMPLATE_CACHE = `

/* ************************************************************ */
/* BEGIN: Caching Functionality for {{.GoFriendlyName}}         */
/* ************************************************************ */

type CacheFor{{.GoFriendlyName}} struct {
		
	enabled bool // flag to determine if caching is enabled for {{.GoFriendlyName}}
	
	sliceCache      map[string][]{{.GoFriendlyName}}
	sliceCacheMutex sync.RWMutex

	whereCache      map[string][]{{.GoFriendlyName}}
	whereCacheMutex sync.RWMutex

	singleRowCache      map[string]{{.GoFriendlyName}}
	singleRowCacheMutex sync.RWMutex

	all      []{{.GoFriendlyName}}
	allMutex sync.RWMutex

	CacheProvider ICacheProvider
}

func (c *CacheFor{{.GoFriendlyName}}) Init() {
	
	if c.sliceCache == nil { c.sliceCache = make(map[string][]{{.GoFriendlyName}}) }
	if c.whereCache == nil { c.whereCache = make(map[string][]{{.GoFriendlyName}}) }
	if c.singleRowCache == nil { c.singleRowCache = make(map[string]{{.GoFriendlyName}}) }
	
} 

func (c *CacheFor{{.GoFriendlyName}}) Dealloc() {
	
	if c.sliceCache != nil { c.sliceCache = nil }
	if c.whereCache != nil { c.whereCache = nil }
	if c.singleRowCache != nil { c.singleRowCache = nil }

	if c.all != nil { c.all = nil }
	
} 

func (c *CacheFor{{.GoFriendlyName}}) IsEnabled() bool {	
	return c.enabled	
}

func (c *CacheFor{{.GoFriendlyName}}) Enable() {
	
	c.enabled = true	
	c.Init()	
}

func (c *CacheFor{{.GoFriendlyName}}) Disable() {
		
	c.enabled = false
	c.Dealloc()
	
}

func (c *CacheFor{{.GoFriendlyName}}) GetAll() ([]{{.GoFriendlyName}}, bool) {

	if c.enabled == false { return nil, false } 

	// if cache provider is nil use memory cache via the built-in
	// map and mutex combo
	if c.CacheProvider == nil {

		c.allMutex.RLock()
		allRecords := c.all	
		c.allMutex.RUnlock()				
		
		return allRecords, (allRecords != nil)
	}

	// todo: implement CacheProvider functionality
	return nil, false

}

// Sets or refreshes the cache for all {{.GoFriendlyName}} records in the database
func (c *CacheFor{{.GoFriendlyName}}) SetAll(all []{{.GoFriendlyName}}) {

	if c.enabled == false { return 	} 	

	// if cache provider is nil use memory cache via the built-in
	// map and mutex combo
	if c.CacheProvider == nil {

		if all != nil {
			
			c.allMutex.Lock()
			
			// empty the slice and release its memory to GC
			if c.all != nil { c.all = nil }
			
			c.all = append(c.all, all...)
			c.allMutex.Unlock()
		}

	}

	// todo: implement CacheProvider functionality

}

// Together with SetWhere, GetWhere enables caching of the Where methods, where the condition
// represents the cache store key.
func (c *CacheFor{{.GoFriendlyName}}) GetWhere(key string) ([]{{.GoFriendlyName}}, bool) {

	if c.enabled == false { return nil, false } 

	// if cache provider is nil use memory cache via the built-in
	// map and mutex combo
	if c.CacheProvider == nil {

		c.whereCacheMutex.RLock()
		w{{.GoFriendlyName}}, keyExists := c.whereCache[key]
		c.whereCacheMutex.RUnlock()

		return w{{.GoFriendlyName}}, keyExists
	}

	// todo: implement CacheProvider functionality
	return nil, false

}

// Together with GetWhere, SetWhere enables caching of the Where methods, where the condition
// represents the cache store key.
func (c *CacheFor{{.GoFriendlyName}}) SetWhere(key string, slice{{.GoFriendlyName}} []{{.GoFriendlyName}}) {

	if c.enabled == false { return 	} 	

	// if cache provider is nil use memory cache via the built-in
	// map and mutex combo
	if c.CacheProvider == nil {

		if slice{{.GoFriendlyName}} != nil {

			whereSliceCopy := make([]{{.GoFriendlyName}}, len(slice{{.GoFriendlyName}}))
			copy(whereSliceCopy, slice{{.GoFriendlyName}})

			c.whereCacheMutex.Lock()
			c.whereCache[key] = whereSliceCopy
			c.whereCacheMutex.Unlock()
		}

	}

	// todo: implement CacheProvider functionality

}

func (c *CacheFor{{.GoFriendlyName}}) GetSlice(key string) ([]{{.GoFriendlyName}}, bool) {

	if c.enabled == false { return nil, false } 

	// if cache provider is nil use memory cache via the built-in
	// map and mutex combo
	if c.CacheProvider == nil {

		c.sliceCacheMutex.RLock()
		s{{.GoFriendlyName}}, keyExists := c.sliceCache[key]
		c.sliceCacheMutex.RUnlock()

		return s{{.GoFriendlyName}}, keyExists
	}

	// todo: implement CacheProvider functionality
	return nil, false

}

func (c *CacheFor{{.GoFriendlyName}}) SetSlice(key string, slice{{.GoFriendlyName}} []{{.GoFriendlyName}}) {

	if c.enabled == false { return 	} 	

	// if cache provider is nil use memory cache via the built-in
	// map and mutex combo
	if c.CacheProvider == nil {

		if slice{{.GoFriendlyName}} != nil {

			sliceCopy := make([]{{.GoFriendlyName}}, len(slice{{.GoFriendlyName}}))
			copy(sliceCopy, slice{{.GoFriendlyName}})

			c.sliceCacheMutex.Lock()
			c.sliceCache[key] = sliceCopy
			c.sliceCacheMutex.Unlock()
		}

	}

	// todo: implement CacheProvider functionality

}

func (c *CacheFor{{.GoFriendlyName}}) Get(key string) (*{{.GoFriendlyName}}, bool) {

	if c.enabled == false { return nil, false } 

	// if cache provider is nil use memory cache via the built-in
	// map and mutex combo
	if c.CacheProvider == nil {

		c.singleRowCacheMutex.RLock()
		singleCachedObject, exists := c.singleRowCache[key]
		c.singleRowCacheMutex.RUnlock()

		if exists {
			return &singleCachedObject, true
		}

		return nil, false
	}

	// todo: implement CacheProvider functionality
	return nil, false
}

func (c *CacheFor{{.GoFriendlyName}}) Set(key string, struct{{.GoFriendlyName}} {{.GoFriendlyName}}) {

	if c.enabled == false { return 	} 	

	// if cache provider is nil use memory cache via the built-in
	// map and mutex combo
	if c.CacheProvider == nil {

		c.singleRowCacheMutex.Lock()
		c.singleRowCache[key] = struct{{.GoFriendlyName}}
		c.singleRowCacheMutex.Unlock()

	}

	// todo: implement CacheProvider functionality

}

/* ************************************************************ */
/* END: Caching Functionality for {{.GoFriendlyName}}           */
/* ************************************************************ */

`

/* Columns */

const PK_GETTER_TEMPLATE_SINGLE_FIELD = `{{$colCount := len .ParentTable.Columns}}{{$functionName := "GetBy"}}{{$sourceParam := print "input" .GoName}}
// Queries the database for a single row based on the specified {{.GoName}} value.
// Returns a pointer to a {{.ParentTable.GoFriendlyName}} structure if a record was found,
// otherwise it returns nil.
func (utilRef *t{{.ParentTable.GoFriendlyName}}Utils) {{$functionName}}{{.GoName}}({{$sourceParam}} {{.GoType}}) (returnStruct *{{.ParentTable.GoFriendlyName}}, err error) {
	
	returnStruct = nil
	err = nil
	
	var errorPrefix = "{{.ParentTable.GoFriendlyName}}GetBy{{.GoName}}() ERROR: "
	
	currentDbHandle := GetDb()
	if currentDbHandle == nil {
		return nil, NewModelsErrorLocal(errorPrefix, "the database handle is nil")
	}

	// define receiving params for the row iteration
	{{range .ParentTable.Columns}}var param{{.GoName}} {{.GoType}}
	{{end}}

	// define the select query
	var query = "{{.ParentTable.GenericSelectQuery}} WHERE {{.DbName}} = $1";

	// we are aiming for a single row so we will use Query Row	
	err = currentDbHandle.QueryRow(query, {{$sourceParam }}).Scan({{range $i, $e := .ParentTable.Columns}}&param{{$e.GoName}}{{if ne (plus1 $i) $colCount}},{{end}}{{end}})
    switch {
    case err == ErrNoRows:
            // no such row found, return nil and nil
			return nil, nil
    case err != nil:
            return nil, NewModelsError(errorPrefix + "fatal error running the query:", err)
    default:
           	// create the return structure as a pointer of the type
			returnStruct = &{{.ParentTable.GoFriendlyName}}{
				{{range .ParentTable.Columns}}{{.GoName}}: param{{.GoName}},
				{{end}}
			}
			// return the structure
			return returnStruct, nil
    }			
}
`

const PK_GETTER_TEMPLATE_MULTI_FIELD = `{{$colCount := len .ParentTable.Columns}}{{$pkColCount := len .ParentTable.PKColumns}}{{$functionName := "GetBy"}}
// Queries the database for a single row based on the specified {{.GoName}} value.
// Returns a pointer to a {{.ParentTable.GoFriendlyName}} structure if a record was found,
// otherwise it returns nil.
func (utilRef *t{{.ParentTable.GoFriendlyName}}Utils) {{$functionName}}` +
	`{{range $i, $e := .ParentTable.PKColumns}}{{$e.GoName}}{{if ne (plus1 $i) $pkColCount}}And{{end}}{{end}}(` +
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
	{{range .ParentTable.Columns}}var param{{.GoName}} {{.GoType}}
	{{end}}

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
				{{range .ParentTable.Columns}}{{.GoName}}: param{{.GoName}},
				{{end}}
			}
			// return the structure
			return returnStruct, nil
    }			
}
`
