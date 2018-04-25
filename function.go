package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"text/template"

	"github.com/silviucm/pgtogogen/internal/pgx"
	"github.com/silviucm/pgtogogen/internal/pgx/pgtype"
)

/* Function Section */

type Function struct {
	Options        *ToolOptions
	ConnectionPool *pgx.ConnPool

	DbName         string
	GoFriendlyName string
	DbComments     string

	Parameters []FunctionParameter

	ReturnType         string
	ReturnGoType       string
	ReturnNullableType string // e.g. "pgx.NullString"

	Columns []Column // column definitions if the return type is a table

	IsReturnVoid        bool
	IsReturnUserDefined bool
	IsReturnASet        bool
	IsReturnARecord     bool
	IsReturnTable       bool
	IsReturnView        bool

	GeneratedTemplate bytes.Buffer
	GoTypesToImport   map[string]string
}

type FunctionParameter struct {
	DbName         string
	GoFriendlyName string
	DbComments     string

	// can be "Input", "Output", "InOut", "Variant"
	Mode string

	Type           string
	GoType         string
	GoNullableType string // e.g. "pgx.NullString"

	IsOptional   bool
	DefaultValue string
}

const (
	FUNC_PARAM_TYPE_INPUT   = "Input"
	FUNC_PARAM_TYPE_OUTPUT  = "Output"
	FUNC_PARAM_TYPE_INOUT   = "InOut"
	FUNC_PARAM_TYPE_VARIANT = "Variant"
)

var FunctionFileGoTypesToImport map[string]string = make(map[string]string)

func CollectFunction(t *ToolOptions, functionName string) (*Function, error) {

	// for more info, check this url
	// http://www.alberton.info/postgresql_meta_info.html

	// the general function details query
	var functionDetailsQuery string = `
		SELECT r.routine_name, r.data_type, r.type_udt_name,  
		proc_details.proretset  
		FROM information_schema.routines r,
		(
		SELECT pg_proc.*
		FROM pg_catalog.pg_proc
		JOIN pg_catalog.pg_namespace ON (pg_proc.pronamespace = pg_namespace.oid)
		WHERE pg_proc.prorettype <> 'pg_catalog.cstring'::pg_catalog.regtype
		AND (pg_proc.proargtypes[0] IS NULL
		OR pg_proc.proargtypes[0] <> 'pg_catalog.cstring'::pg_catalog.regtype)
		AND NOT pg_proc.proisagg
		AND pg_proc.proname = $1 
		AND pg_namespace.nspname = $2 
		AND pg_catalog.pg_function_is_visible(pg_proc.oid) 
		LIMIT 1
		) as proc_details
		WHERE r.routine_schema=$3 AND routine_catalog=$4 AND r.routine_name=$5 
		AND r.routine_type = 'FUNCTION' 
		AND proc_details.proname = r.routine_name
		ORDER BY r.routine_name;`

	var routineName, routineDataType, routineUdtName string
	var isSetOf bool = false

	err := t.ConnectionPool.QueryRow(functionDetailsQuery, functionName, t.DbSchema, t.DbSchema, t.DbName, functionName).Scan(&routineName, &routineDataType, &routineUdtName, &isSetOf)

	switch {
	case err == sql.ErrNoRows:
		log.Println("CollectFunction(): function ", functionName, " no rows returned from information_schema.routines. Skipping.")
		return nil, nil
	case err != nil:
		return nil, err
	default:
	}

	// create a function holder struct
	newFunction := &Function{
		ConnectionPool:    t.ConnectionPool,
		Options:           t,
		DbName:            routineName,
		GoFriendlyName:    GetGoFriendlyNameForFunction(routineName),
		IsReturnASet:      isSetOf,
		IsReturnARecord:   (isSetOf == false),
		GeneratedTemplate: bytes.Buffer{},
	}

	// determine if the return type is user defined or standard postgres type
	if routineDataType == "USER-DEFINED" {

		found := false
		// iterate through the list of tables and views and see if they match the UDT type provided
		for _, currentTable := range t.Tables {
			if currentTable.DbName == routineUdtName {
				found = true
				newFunction.ReturnType = currentTable.DbName
				newFunction.ReturnGoType = currentTable.GoFriendlyName
				newFunction.Columns = currentTable.Columns
			}
		}
		for _, currentView := range t.Views {
			if currentView.DbName == routineUdtName {
				found = true
				newFunction.ReturnType = currentView.DbName
				newFunction.ReturnGoType = currentView.GoFriendlyName
				newFunction.Columns = currentView.Columns
			}
		}
		if found == false {
			log.Println("CollectFunction(): function ", functionName, " has USER-DEFINED data type but no table or view with name ", routineUdtName, " found. Skipping.")
			return nil, nil
		}
		newFunction.IsReturnUserDefined = true

	} else {

		// the function returns a regular Postgres type, so make sure it's not void first
		if routineDataType == "void" {
			newFunction.IsReturnVoid = true
		} else {

			newFunction.ReturnType = routineDataType

			// get the corresponding go type
			correspondingGoType, nullableType, goTypeToImport := GetGoTypeForColumn(routineDataType, true)

			if goTypeToImport != "" {
				if newFunction.GoTypesToImport == nil {
					newFunction.GoTypesToImport = make(map[string]string)
				}

				newFunction.GoTypesToImport[goTypeToImport] = goTypeToImport
			}

			newFunction.ReturnGoType = correspondingGoType
			newFunction.ReturnNullableType = nullableType
		}
	}

	// get the parameters
	newFunction.CollectParameters()

	return newFunction, nil

}

func (f *Function) CollectParameters() {

	var currentParameterName, parameterDataType, parameterMode string
	var parameterDefault pgtype.Text
	var parameterOrdinalPosition pgtype.Int4

	var paramsQuery = `
		SELECT p.parameter_name, p.data_type, p.parameter_mode, p.parameter_default, p.ordinal_position
		FROM information_schema.routines r
		    JOIN information_schema.parameters p ON r.specific_name=p.specific_name
		WHERE r.routine_schema=$1 AND r.routine_catalog=$2 AND r.routine_name=$3 
		AND r.routine_type = 'FUNCTION' 
		ORDER BY r.routine_name, p.ordinal_position;
	`

	rows, err := f.ConnectionPool.Query(paramsQuery, f.Options.DbSchema, f.Options.DbName, f.DbName)

	if err != nil {
		log.Fatal("CollectParameters() fatal error running the query:", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&currentParameterName, &parameterDataType, &parameterMode, &parameterDefault, &parameterOrdinalPosition)
		if err != nil {
			log.Fatal("CollectParameters() fatal error inside rows.Next() iteration: ", err)
		}

		resolvedGoType, nullableType, goTypeToImport := GetGoTypeForColumn(parameterDataType, false)

		if goTypeToImport != "" {
			if f.GoTypesToImport == nil {
				f.GoTypesToImport = make(map[string]string)
			}
			f.GoTypesToImport[goTypeToImport] = goTypeToImport
		}

		var parameterDefaultVal string = ""
		if parameterDefault.Status == pgtype.Present && parameterDefault.String != "" {
			parameterDefaultVal = parameterDefault.String
		}

		// instantiate a function parameter struct
		currentParam := &FunctionParameter{
			DbName:       currentParameterName,
			Type:         parameterDataType,
			DefaultValue: parameterDefaultVal,

			GoFriendlyName: GetGoFriendlyNameForFunctionParam(currentParameterName),
			GoType:         resolvedGoType,
			GoNullableType: nullableType,
		}

		if currentParam.GoType != "" {
			f.Parameters = append(f.Parameters, *currentParam)
		}

	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	return

}

func (f *Function) Generate() {

	f.generateAndAppendTemplate("GenerateFunction", FUNCTION_TEMPLATE, "Function generated.")

}

func (f *Function) WriteToBuffer(functionBuffer *bytes.Buffer) {

	_, err := functionBuffer.Write(f.GeneratedTemplate.Bytes())
	if err != nil {
		log.Fatal("WriteToBuffer() for function ", f.DbName, " fatal error running template.Execute:", err)
	}

}

func (f *Function) generateAndAppendTemplate(templateName string, templateContent string, taskCompletionMessage string) {

	tmpl, err := template.New(templateName).Funcs(fns).Parse(templateContent)
	if err != nil {
		log.Fatal(templateName+" fatal error running template.New:", err)
	}

	var generatedTemplate bytes.Buffer
	err = tmpl.Execute(&generatedTemplate, f)
	if err != nil {
		log.Fatal(templateName+" fatal error running template.Execute:", err)
	}

	if _, err = f.GeneratedTemplate.Write(generatedTemplate.Bytes()); err != nil {
		log.Fatal(templateName+" fatal error writing the generated template bytes to the function buffer:", err)
	}

	if taskCompletionMessage != "" {
		fmt.Println(taskCompletionMessage)
	}

}

func generateFunctionFilePrefix(t *ToolOptions, functionBuffer *bytes.Buffer) {

	templateName := "FUNCTION_TEMPLATE_PREFIX"
	templateCode := FUNCTION_TEMPLATE_PREFIX

	tmpl, err := template.New("FunctionPrefixTemplate").Funcs(fns).Parse(templateCode)
	if err != nil {
		log.Fatal(templateName, " template: fatal error running template.New:", err)
	}

	functionData := struct {
		Options         *ToolOptions
		GoTypesToImport map[string]string
	}{t, FunctionFileGoTypesToImport}

	var generatedTemplate bytes.Buffer
	err = tmpl.Execute(&generatedTemplate, functionData)
	if err != nil {
		log.Fatal(templateName+" fatal error running template.Execute:", err)
	}

	if _, err = functionBuffer.Write(generatedTemplate.Bytes()); err != nil {
		log.Fatal(templateName+" fatal error writing the generated template bytes to the function buffer:", err)
	}

}

/* Util methods */

func GetGoFriendlyNameForFunction(routineName string) string {

	// find if the table name has underscore
	if strings.Contains(routineName, "_") == false {
		return strings.Title(routineName)
	}

	subNames := strings.Split(routineName, "_")

	if subNames == nil {
		log.Fatal("GetGoFriendlyNameForFunction() fatal error for function name: ", routineName, ". Please ensure a valid function name is provided.")
	}

	for i := range subNames {
		subNames[i] = strings.Title(subNames[i])
	}

	return strings.Join(subNames, "")
}

func GetGoFriendlyNameForFunctionParam(paramName string) string {

	// find if the table name has underscore
	if strings.Contains(paramName, "_") == false {
		return strings.Title(paramName)
	}

	subNames := strings.Split(paramName, "_")

	if subNames == nil {
		log.Fatal("GetGoFriendlyNameForFunctionParam() fatal error for param name: ", paramName, ". Please ensure a valid param name is provided.")
	}

	for i := range subNames {
		subNames[i] = strings.Title(subNames[i])
	}

	return strings.Join(subNames, "")
}
