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
}

func (tbl *Table) GenerateTableStruct() {

	fmt.Println("--------------------------------------------------------------------------------------------")
	log.Println("Beginning generation for table: ", tbl.TableName)
	fmt.Println("--------------------------------------------------------------------------------------------")

	tmpl, err := template.New("tableTemplate").Parse(TABLE_TEMPLATE)
	if err != nil {
		log.Fatal("GenerateTableStruct() fatal error running template.New:", err)
	}

	//

	var generatedTemplate bytes.Buffer
	err = tmpl.Execute(&generatedTemplate, tbl)
	if err != nil {
		log.Fatal("GenerateTableStruct() fatal error running template.Execute:", err)
	}

	var filePath string = tbl.Options.OutputFolder + "/" + cmsutils.String.CamelCase(tbl.GoFriendlyName) + ".go"

	err = ioutil.WriteFile(filePath, generatedTemplate.Bytes(), 0644)
	if err != nil {
		log.Fatal("GenerateTableStruct() fatal error writing to file:", err)
	}

	fmt.Println("Finished generating structures for table. Filepath: " + filePath)

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
