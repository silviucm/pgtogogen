package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"text/template"

	pgx "github.com/silviucm/pgtogogen/internal/pgx"
	pgtype "github.com/silviucm/pgtogogen/internal/pgx/pgtype"
)

/* Table Section */

type Table struct {
	Options        *ToolOptions
	ConnectionPool *pgx.ConnPool

	Columns           []Column
	ColumnsString     string
	ColumnsStringNoPK string

	// Go-safe column sequence, prefixed by the underscore character. For example, the column "type" would fail in Go, because "type" is
	// a reserved keyword. Appending an underscore solves this problem.
	ColumnsStringGoSafe     string
	ColumnsStringNoPKGoSafe string

	PKColumns       []Column
	PKColumnsString string

	FKColumns       []Column
	FKColumnsString string

	UniqueConstraints []Constraint

	DbName         string
	GoFriendlyName string
	DbComments     string

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

	// this value is true for tables, false for views
	IsTable bool
}

func (tbl *Table) CollectColumns() error {

	var currentColumnName, isNullable string
	var columnDefault pgtype.Text
	var charMaxLength pgtype.Int4
	var ordinalPosition int

	// For fixed length arrays (e.g. character[]) we cannot infer the data type just
	// from the data_type column. That will contain "ARRAY" and udt_name will contain
	// the specific type (e.g. "_bpchar" for character[])
	var dataType, udtName string

	columnQuery := "SELECT column_name, column_default, is_nullable, data_type, udt_name, character_maximum_length, ordinal_position FROM information_schema.columns " +
		" WHERE table_schema = 'public' AND table_name = $1 ORDER BY ordinal_position;"
	rows, err := tbl.ConnectionPool.Query(columnQuery, tbl.DbName)

	if err != nil {
		log.Fatal("CollectColumns() fatal error running the query:", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&currentColumnName, &columnDefault, &isNullable, &dataType, &udtName, &charMaxLength, &ordinalPosition)
		if err != nil {
			log.Fatal("CollectColumns() fatal error inside rows.Next() iteration: ", err)
		}

		nullable := DecodeNullable(isNullable)
		resolvedGoType, nullableType, goTypeToImport := GetGoTypeForColumn(dataType, nullable, udtName)

		if resolvedGoType == "" {
			log.Fatalf("FATAL: CollectColumns for table %s, column %s could not resolve type %s.\nQuery:\n%s\n", tbl.DbName, currentColumnName, dataType, columnQuery)
		}

		if goTypeToImport != "" {
			if tbl.GoTypesToImport == nil {
				tbl.GoTypesToImport = make(map[string]string)
			}

			tbl.GoTypesToImport[goTypeToImport] = goTypeToImport
		}

		// instantiate a column struct
		currentColumn := &Column{
			DbName:          currentColumnName,
			OrdinalPosition: ordinalPosition,
			Type:            dataType,
			DefaultValue:    columnDefault,
			Nullable:        nullable,
			MaxLength:       DecodeMaxLength(charMaxLength),
			IsSequence:      DecodeIsColumnSequence(columnDefault),

			IsCompositePK: false, IsPK: false, IsFK: false,

			GoName:         GetGoFriendlyNameForColumn(currentColumnName),
			GoType:         resolvedGoType,
			GoNullableType: nullableType,

			ConnectionPool: tbl.ConnectionPool,
			Options:        tbl.Options,
			IsGuid:         (dataType == "uuid"),
		}

		tbl.Columns = append(tbl.Columns, *currentColumn)

	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	if tbl.Columns != nil {
		// get all columns and all params string friendly
		tbl.ColumnsString = tbl.getSqlFriendlyColumnList(false, false)
		tbl.ColumnsStringGoSafe = tbl.getSqlFriendlyColumnList(false, true)
		tbl.ParamString = tbl.getSqlFriendlyParameters(false)
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

	rows, err := tbl.ConnectionPool.Query(pkQuery, tbl.Options.DbName, tbl.DbName)

	if err != nil {
		log.Fatal("CollectPrimaryKeys() fatal error running the query:", err)
	}
	defer rows.Close()

	var numberOfPKs int = 0

	pkColumnsString := ""
	for rows.Next() {
		err := rows.Scan(&currentConstraintName, &currentColumnName, &ordinalPosition)

		if err != nil && strings.Contains(err.Error(), "Cannot decode null into string") == false {
			log.Fatal("CollectPrimaryKeys() fatal error inside rows.Next() iteration: ", err)
		}

		if err != nil && strings.Contains(err.Error(), "Cannot decode null into string") {
			continue
		}

		if tbl.Columns == nil {
			log.Fatal("CollectPrimaryKeys() FATAL: nil Columns slice in this Table struct instance. Make sure you call CollectColumns() before this method.")
		}

		for i := range tbl.Columns {
			if tbl.Columns[i].DbName == currentColumnName {
				tbl.Columns[i].IsPK = true
				tbl.Columns[i].Nullable = false
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
	if err != nil && strings.Contains(err.Error(), "Cannot decode null into string") == false {
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

	// let's generate the PK-dependent strings properly
	tbl.ColumnsString = tbl.getSqlFriendlyColumnList(false, false)
	tbl.ColumnsStringGoSafe = tbl.getSqlFriendlyColumnList(false, true)
	tbl.ColumnsStringNoPK = tbl.getSqlFriendlyColumnList(true, false)
	tbl.ColumnsStringNoPKGoSafe = tbl.getSqlFriendlyColumnList(true, true)

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

	rows, err := tbl.ConnectionPool.Query(fkQuery, tbl.Options.DbName, tbl.DbName)

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
			if tbl.Columns[i].DbName == currentColumnName {
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

	return nil
}

// CollectUniqueConstraints collects all the unique constraints for the table.
// Unique indexes are not included, and are collected by CollectUniqueIndexes.
func (tbl *Table) CollectUniqueConstraints() error {

	var currentConstraintName, currentConstraintSchema, currentTableName, currentColumnName, currentConstraintType string

	var constraintsQuery = `SELECT  
        tc.constraint_name,         
        tc.constraint_schema,
        tc.table_name, 
        kcu.column_name,
        tc.constraint_type
    FROM 
        information_schema.table_constraints AS tc  
        JOIN information_schema.key_column_usage AS kcu ON (tc.constraint_name = kcu.constraint_name and tc.table_name = kcu.table_name)     
    WHERE tc.constraint_schema = $1 AND tc.table_name = $2 AND tc.constraint_type = 'UNIQUE'   
	`

	rows, err := tbl.ConnectionPool.Query(constraintsQuery, tbl.Options.DbSchema, tbl.DbName)

	if err != nil {
		log.Fatal("CollectUniqueConstraints() fatal error running the query:", err)
	}
	defer rows.Close()

	var numberOfUQs int = 0

	constraintsMap := make(map[string]Constraint)
	for rows.Next() {
		err := rows.Scan(&currentConstraintName, &currentConstraintSchema, &currentTableName, &currentColumnName, &currentConstraintType)
		if err != nil {
			log.Fatal("CollectUniqueConstraints() fatal error inside rows.Next() iteration: ", err)
		}

		if tbl.Columns == nil {
			log.Fatal("CollectUniqueConstraints() FATAL: nil Columns slice in this Table struct instance. Make sure you call CollectColumns() before this method.")
		}

		// create the constraint object and put it into the temp map if not already there
		var newOrExistingConstraint Constraint
		newOrExistingConstraint, alreadyThere := constraintsMap[currentConstraintName]

		if alreadyThere == false {
			newOrExistingConstraint = Constraint{}
			newOrExistingConstraint.ConnectionPool = tbl.ConnectionPool
			newOrExistingConstraint.Options = tbl.Options
			newOrExistingConstraint.ParentTable = tbl

			newOrExistingConstraint.DbName = currentConstraintName
			newOrExistingConstraint.IsUnique = true
			newOrExistingConstraint.Type = currentConstraintType

			numberOfUQs = numberOfUQs + 1
		}

		for i := range tbl.Columns {
			if tbl.Columns[i].DbName == currentColumnName {
				// add the column to the Columns slice of the constraint
				newOrExistingConstraint.Columns = append(newOrExistingConstraint.Columns, tbl.Columns[i])
			}
		}

		constraintsMap[currentConstraintName] = newOrExistingConstraint

	}

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	if numberOfUQs > 0 {
		for _, constraint := range constraintsMap {
			tbl.UniqueConstraints = append(tbl.UniqueConstraints, constraint)
		}
	}

	//log.Println("Number of unique constraints collected for ", tbl.DbName, ": ", len(tbl.UniqueConstraints))
	return nil

}

// CollectUniqueIndexes collects all the unique indexes minus the already-collected
// unique constraints.
func (tbl *Table) CollectUniqueIndexes() error {

	var currentConstraintName, currentConstraintSchema, currentTableName, currentColumnName, currentConstraintType string

	var uniqueIndexesQuery = `select i.relname as constraint_name, ist.table_schema as constraint_schema,
    t.relname as table_name, a.attname as column_name, 'UNIQUE' as constraint_type
	from pg_class t, information_schema.tables ist, pg_class i,  pg_index ix, pg_attribute a
	where t.relname = $1 and t.oid = ix.indrelid and ix.indisunique = true and i.oid = ix.indexrelid
    and a.attrelid = t.oid and a.attnum = ANY(ix.indkey) and t.relkind = 'r'
    and t.relname = ist.table_name and ist.table_catalog = 'tradermood' and ist.table_schema = 'public'
    and i.relname NOT IN (SELECT tc.constraint_name FROM information_schema.table_constraints AS tc  
    JOIN information_schema.key_column_usage AS kcu ON (tc.constraint_name = kcu.constraint_name and tc.table_name = kcu.table_name))
	group by t.relname, i.relname, ix.indisunique, a.attname, ist.table_schema
	order by t.relname, i.relname;
	`

	rows, err := tbl.ConnectionPool.Query(uniqueIndexesQuery, tbl.DbName)

	if err != nil {
		log.Fatal("CollectUniqueIndexes() fatal error running the query:", err)
	}
	defer rows.Close()

	var numberOfUQs int = 0

	constraintsMap := make(map[string]Constraint)
	for rows.Next() {
		err := rows.Scan(&currentConstraintName, &currentConstraintSchema, &currentTableName, &currentColumnName, &currentConstraintType)
		if err != nil {
			log.Fatal("CollectUniqueIndexes() fatal error inside rows.Next() iteration: ", err)
		}

		if tbl.Columns == nil {
			log.Fatal("CollectUniqueIndexes() FATAL: nil Columns slice in this Table struct instance. Make sure you call CollectColumns() before this method.")
		}

		// create the constraint object and put it into the temp map if not already there
		var newOrExistingConstraint Constraint
		newOrExistingConstraint, alreadyThere := constraintsMap[currentConstraintName]

		if alreadyThere == false {
			newOrExistingConstraint = Constraint{}
			newOrExistingConstraint.ConnectionPool = tbl.ConnectionPool
			newOrExistingConstraint.Options = tbl.Options
			newOrExistingConstraint.ParentTable = tbl

			newOrExistingConstraint.DbName = currentConstraintName
			newOrExistingConstraint.IsUnique = true
			newOrExistingConstraint.Type = currentConstraintType

			numberOfUQs = numberOfUQs + 1
		}

		for i := range tbl.Columns {
			if tbl.Columns[i].DbName == currentColumnName {
				// add the column to the Columns slice of the constraint
				newOrExistingConstraint.Columns = append(newOrExistingConstraint.Columns, tbl.Columns[i])
			}
		}

		constraintsMap[currentConstraintName] = newOrExistingConstraint

	}

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	if numberOfUQs > 0 {
		for _, constraint := range constraintsMap {
			tbl.UniqueConstraints = append(tbl.UniqueConstraints, constraint)
		}
	}

	//log.Println("Number of unique constraints collected for ", tbl.DbName, ": ", len(tbl.UniqueConstraints))
	return nil

}

func (tbl *Table) CollectComments() error {

	var currentComment string
	var currentObjsubid int32

	var commentsQuery string = `select description as object_comment, objsubid 
		from pg_description join pg_class on pg_description.objoid = pg_class.oid join pg_namespace on pg_class.relnamespace = pg_namespace.oid
		where relname=$1 and nspname=$2 order by objsubid`

	rows, err := tbl.ConnectionPool.Query(commentsQuery, tbl.DbName, tbl.Options.DbSchema)

	if err != nil {
		log.Fatal("CollectComments() fatal error running the query:", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&currentComment, &currentObjsubid)
		if err != nil {
			log.Fatal("CollectComments() fatal error inside rows.Next() iteration: ", err)
		}

		// the table comment is the one with objsubid == 0
		if currentObjsubid == 0 {
			tbl.DbComments = currentComment
		} else {
			if tbl.Columns != nil {
				if len(tbl.Columns) > 0 {

					for i := range tbl.Columns {
						if tbl.Columns[i].OrdinalPosition == int(currentObjsubid) {
							// assign the comment based on the column ordinal position which
							// is the same as the
							tbl.Columns[i].DbComments = currentComment
						}
					}

				}
			}
		}

	}

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
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
			log.Fatal("CollectTables(): FATAL error writing to buffer when generating GenericSelectQuery for table ", tbl.DbName, ": ", writeErr)
		}

		// the column names, comma-separated
		var ignoreSerialColumns bool = false
		var appendUnderscorePrefix bool = false
		_, writeErr = genericSelectQueryBuffer.WriteString(tbl.getSqlFriendlyColumnList(ignoreSerialColumns, appendUnderscorePrefix))
		if writeErr != nil {
			log.Fatal("CollectTables(): FATAL error writing to buffer when generating the column names for table (select) ", tbl.DbName, ": ", writeErr)
		}

		// The FROM section
		_, writeErr = genericSelectQueryBuffer.WriteString(" FROM " + tbl.DbName + " ")
		if writeErr != nil {
			log.Fatal("CollectTables(): FATAL error writing to buffer when generating GenericSelectQuery for table ", tbl.DbName, ": ", writeErr)
		}
		tbl.GenericSelectQuery = genericSelectQueryBuffer.String()
	}
	// END Create the generic SELECT query

	// BEGIN Create the generic INSERT query
	if tbl.Columns != nil {
		genericInsertQueryAllColumnsBuffer := bytes.Buffer{}
		genericInsertQueryNonPKColumnsBuffer := bytes.Buffer{}

		// The INSERT prefix
		_, writeErr := genericInsertQueryNonPKColumnsBuffer.WriteString("INSERT INTO " + tbl.DbName + "(")
		if writeErr != nil {
			log.Fatal("CollectTables(): FATAL error writing to buffer when generating GenericInsertQuery for table ", tbl.DbName, ": ", writeErr)
		}

		_, writeErr = genericInsertQueryAllColumnsBuffer.WriteString("INSERT INTO " + tbl.DbName + "(")
		if writeErr != nil {
			log.Fatal("CollectTables(): FATAL error writing to buffer when generating GenericInsertQuery for table ", tbl.DbName, ": ", writeErr)
		}

		// the column names, comma-separated
		var ignoreSerialColumns bool = true
		var appendUnderscorePrefix bool = false
		_, writeErr = genericInsertQueryNonPKColumnsBuffer.WriteString(tbl.getSqlFriendlyColumnList(ignoreSerialColumns, appendUnderscorePrefix))
		if writeErr != nil {
			log.Fatal("CollectTables(): FATAL error writing to buffer when generating the column names (without pk) for table (insert) ", tbl.DbName, ": ", writeErr)
		}

		ignoreSerialColumns = false
		_, writeErr = genericInsertQueryAllColumnsBuffer.WriteString(tbl.getSqlFriendlyColumnList(ignoreSerialColumns, appendUnderscorePrefix))
		if writeErr != nil {
			log.Fatal("CollectTables(): FATAL error writing to buffer when generating the column names (with pk) for table (insert) ", tbl.DbName, ": ", writeErr)
		}

		// The VALUES section
		ignoreSerialColumns = true
		_, writeErr = genericInsertQueryNonPKColumnsBuffer.WriteString(") VALUES(" + tbl.getSqlFriendlyParameters(ignoreSerialColumns) + ") ")
		if writeErr != nil {
			log.Fatal("CollectTables(): FATAL error writing to buffer when generating GenericInsertQuery (without pk) for table ", tbl.DbName, ": ", writeErr)
		}

		ignoreSerialColumns = false
		_, writeErr = genericInsertQueryAllColumnsBuffer.WriteString(") VALUES(" + tbl.getSqlFriendlyParameters(ignoreSerialColumns) + ") ")
		if writeErr != nil {
			log.Fatal("CollectTables(): FATAL error writing to buffer when generating GenericInsertQuery (with pk) for table ", tbl.DbName, ": ", writeErr)
		}
		tbl.GenericInsertQuery = genericInsertQueryAllColumnsBuffer.String()
		tbl.GenericInsertQueryNoPK = genericInsertQueryNonPKColumnsBuffer.String()

		tbl.ParamString = tbl.getSqlFriendlyParameters(false)
		tbl.ParamStringNoPK = tbl.getSqlFriendlyParameters(true)
	}
	// END Create the generic INSERT query

}

// returns a string of comma separated database column names, as they are used in SELECT
// or INSERT sql statements (e.g. "username, first_name, last_name")
// if ignoreSequenceColumns is true, it checks which columns are auto-generated via
// sequences and does not include those.
func (tbl *Table) getSqlFriendlyColumnList(ignoreSequenceColumns bool, appendUnderscorePrefix bool) string {

	var underscorePrefix string = "_"
	if appendUnderscorePrefix == false {
		underscorePrefix = ""
	}

	genericQueryFriendlyColumnsBuffer := bytes.Buffer{}

	var totalNumberOfColumns int = len(tbl.Columns) - 1
	var colNameToWriteToBuffer string = ""

	for colRange := range tbl.Columns {

		if ignoreSequenceColumns == true && tbl.Columns[colRange].IsSequence == true {
			continue
		}

		if totalNumberOfColumns == colRange {
			colNameToWriteToBuffer = underscorePrefix + tbl.Columns[colRange].DbName
		} else {
			colNameToWriteToBuffer = underscorePrefix + tbl.Columns[colRange].DbName + ", "
		}

		_, writeErr := genericQueryFriendlyColumnsBuffer.WriteString(colNameToWriteToBuffer)
		if writeErr != nil {
			log.Fatal("Table.getSqlFriendlyColumnList(): FATAL error writing to buffer when generating column names for table ", tbl.DbName, ": ", writeErr)
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
			log.Fatal("Table.getSqlFriendlyParameters(): FATAL error writing to buffer when generating params for table ", tbl.DbName, ": ", writeErr)
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

	tbl.generateAndAppendTemplate("GenerateTableStruct()", TABLE_TEMPLATE, "Table structure generated.")
}

func (tbl *Table) GenerateSelectFunctions() {

	tbl.generateAndAppendTemplate("tableSelectWhereTemplate", SELECT_TEMPLATE_WHERE, "")

	tbl.generateAndAppendTemplate("tableSelectAllTemplate", SELECT_TEMPLATE_ALL, "")

	tbl.generateAndAppendTemplate("tableSelectWhereTemplateTx", SELECT_TEMPLATE_WHERE_TX, "")
	tbl.generateAndAppendTemplate("tableSelectAllTemplateTx", SELECT_TEMPLATE_ALL_TX, "")

	// generate the caching functionality
	tbl.generateAndAppendTemplate("tableCachingTemplate", TABLE_TEMPLATE_CACHE, "")

	// generate the extra functionality (count, first, last, single)
	tbl.generateAndAppendTemplate("tableCountTemplate", SELECT_TEMPLATE_COUNT, "")
	tbl.generateAndAppendTemplate("tableSingleTemplate", SELECT_TEMPLATE_SINGLE_ATOMIC, "")
	tbl.generateAndAppendTemplate("tableSingleTemplate", SELECT_TEMPLATE_SINGLE_TX, "")

	fmt.Println("Table select functions generated.")

}

func (tbl *Table) GenerateInsertFunctions() {

	tbl.generateAndAppendTemplate("tableInsertFunctionTemplate", TABLE_STATIC_INSERT_TEMPLATE_ATOMIC, "")
	tbl.generateAndAppendTemplate("tableInsertFunctionTemplateTx", TABLE_STATIC_INSERT_TEMPLATE_TX, "")

	fmt.Println("Table insert functions generated.")

}

func (tbl *Table) GenerateBulkCopyFunctions() {

	tbl.generateAndAppendTemplate("tableBulkCopyTemplate", TABLE_STATIC_BULK_COPY_TEMPLATE, "")

	fmt.Println("Table bulk copy functions generated.")

}

func (tbl *Table) GenerateUpdateFunctions() {

	tbl.generateAndAppendTemplate("tableUpdateFunctionTemplate", TABLE_STATIC_UPDATE_TEMPLATE, "")
	tbl.generateAndAppendTemplate("tableUpdateFunctionTemplateTx", TABLE_STATIC_UPDATE_TEMPLATE_TX, "")

	tbl.generateAndAppendTemplate("tableUpdateWithMaskFunctionTemplate", TABLE_STATIC_UPDATE_WITH_MASK, "")
	tbl.generateAndAppendTemplate("tableUpdateWithMaskFunctionTemplateTx", TABLE_STATIC_UPDATE_WITH_MASK_TX, "")

	tbl.generateAndAppendTemplate("tableInstanceUpdateFunctionTemplate", TABLE_INSTANCE_UPDATE_TEMPLATE, "")
	tbl.generateAndAppendTemplate("tableInstanceUpdateFunctionTemplateTx", TABLE_INSTANCE_UPDATE_TEMPLATE_TX, "")

	fmt.Println("Table update functions generated.")

}

func (tbl *Table) GenerateDeleteFunctions() {

	tbl.generateAndAppendTemplate("tableDeleteFunctionTemplate", TABLE_STATIC_DELETE_TEMPLATE, "")
	tbl.generateAndAppendTemplate("tableDeleteFunctionTemplate", TABLE_STATIC_DELETE_TEMPLATE_TX, "")

	tbl.generateAndAppendTemplate("tableDeleteInstanceFunctionTemplate", TABLE_STATIC_DELETE_INSTANCE_TEMPLATE, "")
	tbl.generateAndAppendTemplate("tableDeleteInstanceFunctionTemplate", TABLE_STATIC_DELETE_INSTANCE_TEMPLATE_TX, "")

	tbl.generateAndAppendTemplate("tableDeleteAllFunctionTemplate", TABLE_STATIC_DELETE_ALL_TEMPLATE, "")
	tbl.generateAndAppendTemplate("tableDeleteAllFunctionTemplate", TABLE_STATIC_DELETE_ALL_TEMPLATE_TX, "")

	fmt.Println("Table delete functions generated.")
}

func (tbl *Table) WriteToFile() {

	var filePath string = tbl.Options.OutputFolder + "/" + CamelCase(tbl.GoFriendlyName) + ".go"

	err := ioutil.WriteFile(filePath, tbl.GeneratedTemplate.Bytes(), 0644)
	if err != nil {
		log.Fatal("WriteToFile() fatal error writing to file:", err)
	}

	fmt.Println("Finished generating structures for table " + tbl.DbName + ". Filepath: " + filePath)
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
		fmt.Println("Skipping generating custom file for table " + tbl.DbName + ". Filepath: " + customFilePath + " already exists.")
	} else {
		err := ioutil.WriteFile(customFilePath, generatedCustomFileTemplate.Bytes(), 0644)
		if err != nil {
			log.Fatal("WriteToCustomFile() fatal error writing to file:", err)
		}

		fmt.Println("Finished generating custom file for table " + tbl.DbName + ". Filepath: " + customFilePath)
	}

}

func (tbl *Table) generateAndAppendTemplate(templateName string, templateContent string, taskCompletionMessage string) {

	tmpl, err := template.New(templateName).Funcs(fns).Parse(templateContent)
	if err != nil {
		log.Fatal(templateName+" fatal error running template.New:", err)
	}

	var generatedTemplate bytes.Buffer
	err = tmpl.Execute(&generatedTemplate, tbl)
	if err != nil {
		log.Fatal(templateName+" fatal error running template.Execute:", err)
	}

	if _, err = tbl.GeneratedTemplate.Write(generatedTemplate.Bytes()); err != nil {
		log.Fatal(templateName+" fatal error writing the generated template bytes to the table buffer:", err)
	}

	if taskCompletionMessage != "" {
		fmt.Println(taskCompletionMessage)
	}

}
