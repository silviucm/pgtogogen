package main

import (
	"bytes"
	"fmt"
	"github.com/silviucm/pgx"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"text/template"
)

/* Table Section */

type Table struct {
	Options        *ToolOptions
	ConnectionPool *pgx.ConnPool

	Columns       []Column
	ColumnsString string

	PKColumns       []Column
	PKColumnsString string

	FKColumns       []Column
	FKColumnsString string

	TableName      string
	GoFriendlyName string

	GoTypesToImport map[string]string

	GeneratedTemplate bytes.Buffer

	// holds a typical SELECT FROM with all the db columns without any WHERE condition
	GenericSelectQuery string

	// holds a typical INSERT query, postgres style
	GenericInsertQuery string
}

func (tbl *Table) CollectColumns() error {

	var currentColumnName, isNullable, dataType string
	var columnDefault pgx.NullString
	var charMaxLength pgx.NullInt32

	rows, err := tbl.ConnectionPool.Query("SELECT column_name, column_default, is_nullable, data_type, character_maximum_length FROM information_schema.columns "+
		" WHERE table_schema = 'public' AND table_name = $1 ORDER BY ordinal_position;", tbl.TableName)

	if err != nil {
		log.Fatal("CollectColumns() fatal error running the query:", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&currentColumnName, &columnDefault, &isNullable, &dataType, &charMaxLength)
		if err != nil {
			log.Fatal("CollectColumns() fatal error inside rows.Next() iteration: ", err)
		}

		resolvedGoType, goTypeToImport := GetGoTypeForColumn(dataType)

		if goTypeToImport != "" {
			if tbl.GoTypesToImport == nil {
				tbl.GoTypesToImport = make(map[string]string)
			}

			tbl.GoTypesToImport[goTypeToImport] = goTypeToImport
		}

		// instantiate a column struct
		currentColumn := &Column{
			Name:         currentColumnName,
			Type:         dataType,
			DefaultValue: columnDefault,
			Nullable:     DecodeNullable(isNullable),
			MaxLength:    DecodeMaxLength(charMaxLength),
			IsSequence:   DecodeIsColumnSequence(columnDefault),

			IsCompositePK: false, IsPK: false, IsFK: false,

			GoName: GetGoFriendlyNameForColumn(currentColumnName),
			GoType: resolvedGoType,

			ConnectionPool: tbl.ConnectionPool,
			Options:        tbl.Options,
		}

		tbl.Columns = append(tbl.Columns, *currentColumn)

	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	return nil

}

func (tbl *Table) CollectPrimaryKeys() error {

	var currentConstraintName, currentColumnName string
	var ordinalPosition int32

	var pkQuery = `SELECT kcu.constraint_name,
         kcu.column_name,
         kcu.ordinal_position
			FROM    INFORMATION_SCHEMA.TABLES t
			         LEFT JOIN INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
			                 ON tc.table_catalog = t.table_catalog
			                 AND tc.table_schema = t.table_schema
			                 AND tc.table_name = t.table_name
			                 AND tc.constraint_type = 'PRIMARY KEY'
			         LEFT JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE kcu
			                 ON kcu.table_catalog = tc.table_catalog
			                 AND kcu.table_schema = tc.table_schema
			                 AND kcu.table_name = tc.table_name
			                 AND kcu.constraint_name = tc.constraint_name
			WHERE   t.table_schema NOT IN ('pg_catalog', 'information_schema') AND t.table_catalog = $1 AND t.table_name = $2
			ORDER BY t.table_catalog,
			         t.table_schema,
			         t.table_name,
			         kcu.constraint_name,
			         kcu.ordinal_position;`

	rows, err := tbl.ConnectionPool.Query(pkQuery, tbl.Options.DbName, tbl.TableName)

	if err != nil {
		log.Fatal("CollectPrimaryKeys() fatal error running the query:", err)
	}
	defer rows.Close()

	var numberOfPKs int = 0

	pkColumnsString := ""
	for rows.Next() {
		err := rows.Scan(&currentConstraintName, &currentColumnName, &ordinalPosition)
		if err != nil {
			log.Fatal("CollectPrimaryKeys() fatal error inside rows.Next() iteration: ", err)
		}

		if tbl.Columns == nil {
			log.Fatal("CollectPrimaryKeys() FATAL: nil Columns slice in this Table struct instance. Make sure you call CollectColumns() before this method.")
		}

		for i := range tbl.Columns {
			if tbl.Columns[i].Name == currentColumnName {
				tbl.Columns[i].IsPK = true
				tbl.Columns[i].IsCompositePK = false
				numberOfPKs = numberOfPKs + 1

				// add this column to the tables's PK columns slice
				tbl.PKColumns = append(tbl.PKColumns, tbl.Columns[i])

				// and to the pk columns string
				pkColumnsString = pkColumnsString + currentColumnName + ", "
			}
		}

	}

	// just in case ignoring sequence columns happened to produce a situation where there is a
	// comma followed by space at the end of the string, let's strip it
	if strings.HasSuffix(pkColumnsString, ", ") {
		pkColumnsString = strings.TrimSuffix(pkColumnsString, ", ")
	}
	tbl.PKColumnsString = pkColumnsString

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	// if we have more than one PK we need to iterate again and set the
	// composite PK flag wherever IsPK is true
	if numberOfPKs > 1 {
		for i := range tbl.Columns {
			if tbl.Columns[i].IsPK == true {
				tbl.Columns[i].IsCompositePK = true
			}
		}
	}

	// in case we generate the PK Getter function, we need to make sure the "database/sql" type is
	// imported inside the generated template
	// the generated function will make use of comparisons, such as
	// case err == sql.ErrNoRows
	if tbl.Options.GeneratePKGetters {
		tbl.AddGoTypeToImport("database/sql")
	}

	return nil

}

func (tbl *Table) CollectForeignKeys() error {

	var currentConstraintName, currentColumnName, foreignTableName, foreignColumnName string

	var fkQuery string = `SELECT
		    tc.constraint_name, kcu.column_name, 
		    ccu.table_name AS foreign_table_name,
		    ccu.column_name AS foreign_column_name 
		FROM 
		    information_schema.table_constraints AS tc 
		    JOIN information_schema.key_column_usage AS kcu
		      ON tc.constraint_name = kcu.constraint_name
		    JOIN information_schema.constraint_column_usage AS ccu
		      ON ccu.constraint_name = tc.constraint_name
		WHERE constraint_type = 'FOREIGN KEY' AND tc.table_catalog=$1 AND tc.table_name=$2 ;`

	rows, err := tbl.ConnectionPool.Query(fkQuery, tbl.Options.DbName, tbl.TableName)

	if err != nil {
		log.Fatal("CollectForeignKeys() fatal error running the query:", err)
	}
	defer rows.Close()

	var numberOfFKs int = 0

	fkColumnsString := ""
	for rows.Next() {
		err := rows.Scan(&currentConstraintName, &currentColumnName, &foreignTableName, &foreignColumnName)
		if err != nil {
			log.Fatal("CollectForeignKeys() fatal error inside rows.Next() iteration: ", err)
		}

		if tbl.Columns == nil {
			log.Fatal("CollectForeignKeys() FATAL: nil Columns slice in this Table struct instance. Make sure you call CollectColumns() before this method.")
		}

		for i := range tbl.Columns {
			if tbl.Columns[i].Name == currentColumnName {
				tbl.Columns[i].IsFK = true
				numberOfFKs = numberOfFKs + 1

				// add this column to the tables's FK columns slice
				tbl.FKColumns = append(tbl.FKColumns, tbl.Columns[i])

				// and to the fk columns string
				fkColumnsString = fkColumnsString + currentColumnName + ", "
			}
		}

	}

	// just in case ignoring sequence columns happened to produce a situation where there is a
	// comma followed by space at the end of the string, let's strip it
	if strings.HasSuffix(fkColumnsString, ", ") {
		fkColumnsString = strings.TrimSuffix(fkColumnsString, ", ")
	}
	tbl.FKColumnsString = fkColumnsString

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	// in case we generate the FK Getter function, we need to make sure the "database/sql" type is
	// imported inside the generated template
	// the generated function will make use of comparisons, such as
	// case err == sql.ErrNoRows
	if tbl.Options.GenerateGuidGetters {
		tbl.AddGoTypeToImport("database/sql")
	}

	return nil
}

func (tbl *Table) AddGoTypeToImport(goTypeToImport string) {

	if tbl.GoTypesToImport == nil {
		tbl.GoTypesToImport = make(map[string]string)
	}

	tbl.GoTypesToImport[goTypeToImport] = goTypeToImport
}

func (tbl *Table) CreateGenericQueries() {

	// BEGIN Create the generic SELECT query
	if tbl.Columns != nil {
		genericSelectQueryBuffer := bytes.Buffer{}

		// The SELECT prefix
		_, writeErr := genericSelectQueryBuffer.WriteString("SELECT ")
		if writeErr != nil {
			log.Fatal("CollectTables(): FATAL error writing to buffer when generating GenericSelectQuery for table ", tbl.TableName, ": ", writeErr)
		}

		// the column names, comma-separated
		var ignoreSerialColumns bool = true
		_, writeErr = genericSelectQueryBuffer.WriteString(tbl.getSqlFriendlyColumnList(ignoreSerialColumns))
		if writeErr != nil {
			log.Fatal("CollectTables(): FATAL error writing to buffer when generating the column names for table (select) ", tbl.TableName, ": ", writeErr)
		}

		// The FROM section
		_, writeErr = genericSelectQueryBuffer.WriteString(" FROM " + tbl.TableName + " ")
		if writeErr != nil {
			log.Fatal("CollectTables(): FATAL error writing to buffer when generating GenericSelectQuery for table ", tbl.TableName, ": ", writeErr)
		}
		tbl.GenericSelectQuery = genericSelectQueryBuffer.String()
	}
	// END Create the generic SELECT query

	// BEGIN Create the generic INSERT query
	if tbl.Columns != nil {
		genericInsertQueryBuffer := bytes.Buffer{}

		// The INSERT prefix
		_, writeErr := genericInsertQueryBuffer.WriteString("INSERT INTO " + tbl.TableName + "(")
		if writeErr != nil {
			log.Fatal("CollectTables(): FATAL error writing to buffer when generating GenericInsertQuery for table ", tbl.TableName, ": ", writeErr)
		}

		// the column names, comma-separated
		var ignoreSerialColumns bool = true
		_, writeErr = genericInsertQueryBuffer.WriteString(tbl.getSqlFriendlyColumnList(ignoreSerialColumns))
		if writeErr != nil {
			log.Fatal("CollectTables(): FATAL error writing to buffer when generating the column names for table (insert) ", tbl.TableName, ": ", writeErr)
		}

		// The VALUES section
		_, writeErr = genericInsertQueryBuffer.WriteString(") VALUES(" + tbl.getSqlFriendlyParameters(ignoreSerialColumns) + ") ")
		if writeErr != nil {
			log.Fatal("CollectTables(): FATAL error writing to buffer when generating GenericInsertQuery for table ", tbl.TableName, ": ", writeErr)
		}
		tbl.GenericInsertQuery = genericInsertQueryBuffer.String()
	}
	// END Create the generic INSERT query

}

// returns a string of comma separated database column names, as they are used in SELECT
// or INSERT sql statements (e.g. "username, first_name, last_name")
// if ignoreSequenceColumns is true, it checks which columns are auto-generated via
// sequences and does not include those.
func (tbl *Table) getSqlFriendlyColumnList(ignoreSequenceColumns bool) string {

	genericQueryFriendlyColumnsBuffer := bytes.Buffer{}

	var totalNumberOfColumns int = len(tbl.Columns) - 1
	var colNameToWriteToBuffer string = ""

	for colRange := range tbl.Columns {

		if ignoreSequenceColumns == true && tbl.Columns[colRange].IsSequence == true {
			continue
		}

		if totalNumberOfColumns == colRange {
			colNameToWriteToBuffer = tbl.Columns[colRange].Name
		} else {
			colNameToWriteToBuffer = tbl.Columns[colRange].Name + ", "
		}

		_, writeErr := genericQueryFriendlyColumnsBuffer.WriteString(colNameToWriteToBuffer)
		if writeErr != nil {
			log.Fatal("Table.getSqlFriendlyColumnList(): FATAL error writing to buffer when generating column names for table ", tbl.TableName, ": ", writeErr)
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
func (tbl *Table) getSqlFriendlyParameters(ignoreSequenceColumns bool) string {

	genericQueryFriendlyParamsBuffer := bytes.Buffer{}

	var totalNumberOfColumns int = len(tbl.Columns) - 1
	var paramToWriteToBuffer string = ""

	var realParamCount int = 1

	for colRange := range tbl.Columns {

		if ignoreSequenceColumns == true && tbl.Columns[colRange].IsSequence == true {
			continue
		}

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
			log.Fatal("Table.getSqlFriendlyParameters(): FATAL error writing to buffer when generating params for table ", tbl.TableName, ": ", writeErr)
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

func (tbl *Table) GenerateTableStruct() {

	tmpl, err := template.New("tableTemplate").Parse(TABLE_TEMPLATE)
	if err != nil {
		log.Fatal("GenerateTableStruct() fatal error running template.New:", err)
	}

	var generatedTemplate bytes.Buffer
	err = tmpl.Execute(&generatedTemplate, tbl)
	if err != nil {
		log.Fatal("GenerateTableStruct() fatal error running template.Execute:", err)
	}

	if _, err = tbl.GeneratedTemplate.Write(generatedTemplate.Bytes()); err != nil {
		log.Fatal("GenerateTableStruct() fatal error writing the generated template bytes to the table buffer:", err)
	}

	fmt.Println("Table structure generated.")

}

func (tbl *Table) GenerateInsertFunctions() {

	tmpl, err := template.New("tableInsertFunctionTemplate").Funcs(fns).Parse(TABLE_STATIC_INSERT_TEMPLATE)
	if err != nil {
		log.Fatal("GenerateInsertFunctions() fatal error running template.New:", err)
	}

	var generatedTemplate bytes.Buffer
	err = tmpl.Execute(&generatedTemplate, tbl)
	if err != nil {
		log.Fatal("GenerateInsertFunctions() fatal error running template.Execute:", err)
	}

	if _, err = tbl.GeneratedTemplate.Write(generatedTemplate.Bytes()); err != nil {
		log.Fatal("GenerateInsertFunctions() fatal error writing the generated template bytes to the table buffer:", err)
	}

	fmt.Println("Table insert functions generated.")

}

func (tbl *Table) WriteToFile() {

	var filePath string = tbl.Options.OutputFolder + "/" + CamelCase(tbl.GoFriendlyName) + ".go"

	err := ioutil.WriteFile(filePath, tbl.GeneratedTemplate.Bytes(), 0644)
	if err != nil {
		log.Fatal("WriteToFile() fatal error writing to file:", err)
	}

	fmt.Println("Finished generating structures for table " + tbl.TableName + ". Filepath: " + filePath)
}

// This method only creates the custom file if it is not already in the folder.
// Custom files are ideal to place your own methods in addition to the auto-generated ones.
// If the generated files is named user.go, the custom file would be named: user-custom.go
func (tbl *Table) WriteToCustomFile() {

	tmpl, err := template.New("tableTemplateCustom").Parse(TABLE_TEMPLATE_CUSTOM)
	if err != nil {
		log.Fatal("WriteToCustomFile() fatal error running template.New for TABLE_TEMPLATE_CUSTOM:", err)
	}

	var generatedCustomFileTemplate bytes.Buffer
	err = tmpl.Execute(&generatedCustomFileTemplate, tbl)
	if err != nil {
		log.Fatal("WriteToCustomFile() fatal error running template.Execute:", err)
	}

	var customFilePath string = tbl.Options.OutputFolder + "/" + CamelCase(tbl.GoFriendlyName) + "-custom.go"

	if FileExists(customFilePath) {
		fmt.Println("Skipping generating custom file for table " + tbl.TableName + ". Filepath: " + customFilePath + " already exists.")
	} else {
		err := ioutil.WriteFile(customFilePath, generatedCustomFileTemplate.Bytes(), 0644)
		if err != nil {
			log.Fatal("WriteToCustomFile() fatal error writing to file:", err)
		}

		fmt.Println("Finished generating custom file for table " + tbl.TableName + ". Filepath: " + customFilePath)
	}

}
