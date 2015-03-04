package main

import (
	"bytes"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	cmsutils "github.com/silviucm/utils"
	"io/ioutil"
	"log"
	"text/template"
)

/* Table Section */

type Table struct {
	Options  *ToolOptions
	DbHandle *sql.DB

	Columns        []Column
	TableName      string
	GoFriendlyName string

	GoTypesToImport map[string]string

	GeneratedTemplate bytes.Buffer
}

func (tbl *Table) CollectColumns() error {

	var currentColumnName, isNullable, dataType string
	var columnDefault sql.NullString
	var charMaxLength sql.NullInt64

	rows, err := tbl.DbHandle.Query("SELECT column_name, column_default, is_nullable, data_type, character_maximum_length FROM information_schema.columns "+
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

			DbHandle: tbl.DbHandle,
			Options:  tbl.Options,
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
	var ordinalPosition int

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

	rows, err := tbl.DbHandle.Query(pkQuery, tbl.Options.DbName, tbl.TableName)

	if err != nil {
		log.Fatal("CollectPrimaryKeys() fatal error running the query:", err)
	}
	defer rows.Close()

	var numberOfPKs int = 0

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
			}
		}

	}

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

	rows, err := tbl.DbHandle.Query(fkQuery, tbl.Options.DbName, tbl.TableName)

	if err != nil {
		log.Fatal("CollectForeignKeys() fatal error running the query:", err)
	}
	defer rows.Close()

	var numberOfFKs int = 0

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
			}
		}

	}

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	return nil
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

func (tbl *Table) WriteToFile() {

	var filePath string = tbl.Options.OutputFolder + "/" + cmsutils.String.CamelCase(tbl.GoFriendlyName) + ".go"

	err := ioutil.WriteFile(filePath, tbl.GeneratedTemplate.Bytes(), 0644)
	if err != nil {
		log.Fatal("WriteToFile() fatal error writing to file:", err)
	}

	fmt.Println("Finished generating structures for table " + tbl.TableName + ". Filepath: " + filePath)
}
