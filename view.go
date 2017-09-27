package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"text/template"

	"github.com/silviucm/pgx"
)

/* View Section */

type View struct {
	Options        *ToolOptions
	ConnectionPool *pgx.ConnPool

	Columns       []Column
	ColumnsString string

	DbName         string
	GoFriendlyName string

	GoTypesToImport map[string]string

	GeneratedTemplate bytes.Buffer

	// holds a typical SELECT FROM with all the db columns without any WHERE condition
	GenericSelectQuery string

	// holds a typical INSERT query, postgres style
	GenericInsertQuery     string
	GenericInsertQueryNoPK string

	// holds the parameters comma-separated
	ParamString     string
	ParamStringNoPK string

	// true if view is a materialized view, false otherwise
	IsMaterialized bool

	// this value is true for tables, false for views
	IsTable bool
}

func (v *View) CollectColumns() error {

	var currentColumnName, isNullable, dataType string
	var columnDefault pgx.NullString
	var charMaxLength pgx.NullInt32

	rows, err := v.ConnectionPool.Query("SELECT column_name, column_default, is_nullable, data_type, character_maximum_length FROM information_schema.columns "+
		" WHERE table_schema = 'public' AND table_name = $1 ORDER BY ordinal_position;", v.DbName)

	if err != nil {
		log.Fatal("View.CollectColumns() fatal error running the query:", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&currentColumnName, &columnDefault, &isNullable, &dataType, &charMaxLength)
		if err != nil {
			log.Fatal("View.CollectColumns() fatal error inside rows.Next() iteration: ", err)
		}

		nullable := DecodeNullable(isNullable)
		resolvedGoType, nullableType, nullableTypeCreateFunc, goTypeToImport := GetGoTypeForColumn(dataType, nullable)

		if goTypeToImport != "" {
			if v.GoTypesToImport == nil {
				v.GoTypesToImport = make(map[string]string)
			}

			v.GoTypesToImport[goTypeToImport] = goTypeToImport
		}

		// instantiate a column struct
		currentColumn := &Column{
			DbName:       currentColumnName,
			Type:         dataType,
			DefaultValue: columnDefault,
			Nullable:     nullable,
			MaxLength:    DecodeMaxLength(charMaxLength),
			IsSequence:   DecodeIsColumnSequence(columnDefault),

			IsCompositePK: false, IsPK: false, IsFK: false,

			GoName:                 GetGoFriendlyNameForColumn(currentColumnName),
			GoType:                 resolvedGoType,
			GoNullableType:         nullableType,
			NullableTypeCreateFunc: nullableTypeCreateFunc,

			ConnectionPool: v.ConnectionPool,
			Options:        v.Options,
		}

		v.Columns = append(v.Columns, *currentColumn)

	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	if v.Columns != nil {
		// get all columns and all params string friendly
		v.ColumnsString = v.getSqlFriendlyColumnList()
		v.ParamString = v.getSqlFriendlyParameters()
	}

	return nil

}

func (v *View) CollectMaterializedViewColumns() error {

	var currentColumnName, isNullable, dataType string
	var columnDefault pgx.NullString
	var charMaxLength pgx.NullInt32

	var materializedViewsColumnsQuery string = `SELECT attname AS column_name, 
	null as column_default,
	CAST(NOT(attnotnull) AS varchar(10)) as is_nullable,
	atttypid::regtype AS data_type, 
	atttypmod AS character_maximum_length
FROM   pg_attribute
WHERE  attrelid = '` + v.Options.DbSchema + `.` + v.DbName + `'::regclass
AND    attnum > 0
AND    NOT attisdropped; 
`

	rows, err := v.ConnectionPool.Query(materializedViewsColumnsQuery)

	if err != nil {
		log.Fatal("View.CollectColumns() fatal error running the query:", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&currentColumnName, &columnDefault, &isNullable, &dataType, &charMaxLength)
		if err != nil {
			log.Fatal("View.CollectColumns() fatal error inside rows.Next() iteration: ", err)
		}

		nullable := DecodeNullable(isNullable)
		resolvedGoType, nullableType, nullableTypeCreateFunc, goTypeToImport := GetGoTypeForColumn(dataType, nullable)

		if goTypeToImport != "" {
			if v.GoTypesToImport == nil {
				v.GoTypesToImport = make(map[string]string)
			}

			v.GoTypesToImport[goTypeToImport] = goTypeToImport
		}

		// instantiate a column struct
		currentColumn := &Column{
			DbName:       currentColumnName,
			Type:         dataType,
			DefaultValue: columnDefault,
			Nullable:     nullable,
			MaxLength:    DecodeMaxLength(charMaxLength),
			IsSequence:   false,

			IsCompositePK: false, IsPK: false, IsFK: false,

			GoName:                 GetGoFriendlyNameForColumn(currentColumnName),
			GoType:                 resolvedGoType,
			GoNullableType:         nullableType,
			NullableTypeCreateFunc: nullableTypeCreateFunc,

			ConnectionPool: v.ConnectionPool,
			Options:        v.Options,
		}

		v.Columns = append(v.Columns, *currentColumn)

	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	if v.Columns != nil {
		// get all columns and all params string friendly
		v.ColumnsString = v.getSqlFriendlyColumnList()
		v.ParamString = v.getSqlFriendlyParameters()
	}

	return nil

}

func (v *View) AddGoTypeToImport(goTypeToImport string) {

	if v.GoTypesToImport == nil {
		v.GoTypesToImport = make(map[string]string)
	}

	v.GoTypesToImport[goTypeToImport] = goTypeToImport
}

func (v *View) CreateGenericQueries() {

	// BEGIN Create the generic SELECT query
	if v.Columns != nil {
		genericSelectQueryBuffer := bytes.Buffer{}

		// The SELECT prefix
		_, writeErr := genericSelectQueryBuffer.WriteString("SELECT ")
		if writeErr != nil {
			log.Fatal("(v *View) CreateGenericQueries(): FATAL error writing to buffer when generating GenericSelectQuery for view ", v.DbName, ": ", writeErr)
		}

		// the column names, comma-separated
		_, writeErr = genericSelectQueryBuffer.WriteString(v.getSqlFriendlyColumnList())
		if writeErr != nil {
			log.Fatal("(v *View) CreateGenericQueries(): FATAL error writing to buffer when generating the column names for view (select) ", v.DbName, ": ", writeErr)
		}

		// The FROM section
		_, writeErr = genericSelectQueryBuffer.WriteString(" FROM " + v.DbName + " ")
		if writeErr != nil {
			log.Fatal("(v *View) CreateGenericQueries(): FATAL error writing to buffer when generating GenericSelectQuery for view ", v.DbName, ": ", writeErr)
		}
		v.GenericSelectQuery = genericSelectQueryBuffer.String()
	}
	// END Create the generic SELECT query

}

// returns a string of comma separated database column names, as they are used in SELECT
// or INSERT sql statements (e.g. "username, first_name, last_name")
// if ignoreSequenceColumns is true, it checks which columns are auto-generated via
// sequences and does not include those.
func (v *View) getSqlFriendlyColumnList() string {

	genericQueryFriendlyColumnsBuffer := bytes.Buffer{}

	var totalNumberOfColumns int = len(v.Columns) - 1
	var colNameToWriteToBuffer string = ""

	for colRange := range v.Columns {

		if totalNumberOfColumns == colRange {
			colNameToWriteToBuffer = v.Columns[colRange].DbName
		} else {
			colNameToWriteToBuffer = v.Columns[colRange].DbName + ", "
		}

		_, writeErr := genericQueryFriendlyColumnsBuffer.WriteString(colNameToWriteToBuffer)
		if writeErr != nil {
			log.Fatal("View.getSqlFriendlyColumnList(): FATAL error writing to buffer when generating column names for table ", v.DbName, ": ", writeErr)
		}
	}

	finalString := genericQueryFriendlyColumnsBuffer.String()

	// just in case ignoring sequence columns happened to produce a situation where there is a
	// comma followed by space at the end of the string, let's strip it
	if strings.HasSuffix(finalString, ", ") {
		finalString = strings.TrimSuffix(finalString, ", ")
	}

	return finalString
}

// Returns a string of comma separated parameters, incremented by 1, Postgres style,
// but taking into account if some columns are have default sequence autogeneration,
// hence should not be inserted
func (v *View) getSqlFriendlyParameters() string {

	genericQueryFriendlyParamsBuffer := bytes.Buffer{}

	var totalNumberOfColumns int = len(v.Columns) - 1
	var paramToWriteToBuffer string = ""

	var realParamCount int = 1

	for colRange := range v.Columns {

		// we cannot rely on the colRange iterator because we may skip columns
		// which are sequence based, so we would have a situation such as
		// "$1, $3, $4, etc" with $2 missing due to the continue statement above
		var currentParamCount string = "$" + strconv.Itoa(realParamCount)
		realParamCount = realParamCount + 1

		if totalNumberOfColumns == colRange {
			paramToWriteToBuffer = currentParamCount
		} else {
			paramToWriteToBuffer = currentParamCount + ", "
		}

		_, writeErr := genericQueryFriendlyParamsBuffer.WriteString(paramToWriteToBuffer)
		if writeErr != nil {
			log.Fatal("View.getSqlFriendlyParameters(): FATAL error writing to buffer when generating params for table ", v.DbName, ": ", writeErr)
		}
	}

	finalString := genericQueryFriendlyParamsBuffer.String()

	// just in case ignoring sequence columns happened to produce a situation where there is a
	// comma followed by space at the end of the string, let's strip it
	if strings.HasSuffix(finalString, ", ") {
		finalString = strings.TrimSuffix(finalString, ", ")
	}

	return finalString
}

func (v *View) GenerateViewStruct() {

	v.generateAndAppendTemplate("GenerateTableStruct()", VIEW_TEMPLATE, "View structure generated.")
}

func (v *View) GenerateSelectFunctions() {

	v.generateAndAppendTemplate("viewSelectWhereTemplate", SELECT_TEMPLATE_WHERE, "")
	v.generateAndAppendTemplate("viewSelectAllTemplate", SELECT_TEMPLATE_ALL, "")

	v.generateAndAppendTemplate("viewSelectWhereTemplateTx", SELECT_TEMPLATE_WHERE_TX, "")
	v.generateAndAppendTemplate("viewSelectAllTemplateTx", SELECT_TEMPLATE_ALL_TX, "")

	// generate the caching functionality
	v.generateAndAppendTemplate("viewCachingTemplate", TABLE_TEMPLATE_CACHE, "")

	// generate the extra functionality (count, first, last, single)
	v.generateAndAppendTemplate("viewCountTemplate", SELECT_TEMPLATE_COUNT, "")
	v.generateAndAppendTemplate("viewSingleTemplate", SELECT_TEMPLATE_SINGLE_ATOMIC, "")
	v.generateAndAppendTemplate("viewSingleTemplate", SELECT_TEMPLATE_SINGLE_TX, "")

	fmt.Println("View select functions generated.")

}

func (v *View) WriteToFile() {

	var filePath string = v.Options.OutputFolder + "/" + CamelCase(v.GoFriendlyName) + ".go"

	err := ioutil.WriteFile(filePath, v.GeneratedTemplate.Bytes(), 0644)
	if err != nil {
		log.Fatal("WriteToFile() fatal error writing to file:", err)
	}

	fmt.Println("Finished generating structures for view " + v.DbName + ". Filepath: " + filePath)
}

// This method only creates the custom file if it is not already in the folder.
// Custom files are ideal to place your own methods in addition to the auto-generated ones.
// If the generated files is named user.go, the custom file would be named: user-custom.go
func (v *View) WriteToCustomFile() {

	tmpl, err := template.New("viewTemplateCustom").Parse(VIEW_TEMPLATE_CUSTOM)
	if err != nil {
		log.Fatal("WriteToCustomFile() fatal error running template.New for TABLE_TEMPLATE_CUSTOM:", err)
	}

	var generatedCustomFileTemplate bytes.Buffer
	err = tmpl.Execute(&generatedCustomFileTemplate, v)
	if err != nil {
		log.Fatal("WriteToCustomFile() fatal error running template.Execute:", err)
	}

	var customFilePath string = v.Options.OutputFolder + "/" + CamelCase(v.GoFriendlyName) + "-custom.go"

	if FileExists(customFilePath) {
		fmt.Println("Skipping generating custom file for view " + v.DbName + ". Filepath: " + customFilePath + " already exists.")
	} else {
		err := ioutil.WriteFile(customFilePath, generatedCustomFileTemplate.Bytes(), 0644)
		if err != nil {
			log.Fatal("WriteToCustomFile() fatal error writing to file:", err)
		}

		fmt.Println("Finished generating custom file for view " + v.DbName + ". Filepath: " + customFilePath)
	}

}

func (v *View) generateAndAppendTemplate(templateName string, templateContent string, taskCompletionMessage string) {

	tmpl, err := template.New(templateName).Funcs(fns).Parse(templateContent)
	if err != nil {
		log.Fatal(templateName+" fatal error running template.New:", err)
	}

	var generatedTemplate bytes.Buffer
	err = tmpl.Execute(&generatedTemplate, v)
	if err != nil {
		log.Fatal(templateName+" fatal error running template.Execute:", err)
	}

	if _, err = v.GeneratedTemplate.Write(generatedTemplate.Bytes()); err != nil {
		log.Fatal(templateName+" fatal error writing the generated template bytes to the view buffer:", err)
	}

	if taskCompletionMessage != "" {
		fmt.Println(taskCompletionMessage)
	}

}
