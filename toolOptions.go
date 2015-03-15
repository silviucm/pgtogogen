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

type ToolOptions struct {
	DbHost string
	DbPort uint16
	DbName string
	DbUser string
	DbPass string

	OutputFolder string

	PackageName string

	GeneratePKGetters   bool
	GenerateGuidGetters bool

	ConnectionPool *pgx.ConnPool

	Tables []Table
}

func (t *ToolOptions) InitDatabase() (*pgx.ConnPool, error) {

	var successOrFailure string = "OK"

	var config pgx.ConnPoolConfig

	config.Host = t.DbHost
	config.User = t.DbUser
	config.Password = t.DbPass
	config.Database = t.DbName
	config.Port = t.DbPort

	fmt.Println("--------------------------------------------------------------------------------------------")

	connPool, err := pgx.NewConnPool(config)
	if err != nil {
		successOrFailure = "FAILED"
		log.Println("Connecting to database ", t.DbName, " as user ", t.DbUser, " ", successOrFailure, ": \n ", err)
	} else {
		log.Println("Connecting to database ", t.DbName, " as user ", t.DbUser, ": ", successOrFailure)
	}

	fmt.Println("--------------------------------------------------------------------------------------------")

	t.ConnectionPool = connPool

	return t.ConnectionPool, err

}

func (t *ToolOptions) Collect() {

	fmt.Println("--------------------------------------------------------------------------------------------")
	log.Println("Beginning collection of info from the database...")
	fmt.Println("--------------------------------------------------------------------------------------------")

	// collect all the user tables from the database
	fmt.Print("Collecting tables...")
	if err := t.CollectTables(); err != nil {
		log.Fatal("Generate(): CollectTables fatal error: ", err)
	}

	// iterate through each table and generate the struct
	if t.Tables != nil {
		fmt.Println("Done: Found " + strconv.Itoa(len(t.Tables)) + " tables.")
	} else {
		fmt.Println("Done: No tables found.")
	}
}

func (t *ToolOptions) Generate() {

	fmt.Println("--------------------------------------------------------------------------------------------")
	log.Println("Beginning generation of structures")
	fmt.Println("--------------------------------------------------------------------------------------------")

	// iterate through each table and generate anything related
	if t.Tables != nil {

		for i := range t.Tables {

			fmt.Println("--------------------------------------------------------------------------------------------")
			log.Println("Beginning generation for table: ", t.Tables[i].TableName)
			fmt.Println("--------------------------------------------------------------------------------------------")

			// generate the table structure
			t.Tables[i].GenerateTableStruct()

			// generate the insert-related functions
			t.Tables[i].GenerateInsertFunctions()

			// generate the delete-related functions
			t.Tables[i].GenerateDeleteFunctions()

			// generate the queries by PK
			if t.GeneratePKGetters == true {
				fmt.Println("Generating Primary Key Accessor Methods...")
				for colIndex := range t.Tables[i].Columns {
					if t.Tables[i].Columns[colIndex].IsPK {
						pkGetter := t.Tables[i].Columns[colIndex].GeneratePKGetter(&t.Tables[i])
						if _, writeErr := t.Tables[i].GeneratedTemplate.Write(pkGetter); writeErr != nil {
							log.Fatal("Generate fatal error writing bytes from the GeneratePKGetter call: ", writeErr)
						}
					}

				}
			}
		}
	} else {
		fmt.Println("Done: No tables found.")
	}

}

func (t *ToolOptions) WriteFiles() {

	fmt.Println("--------------------------------------------------------------------------------------------")
	log.Println("Writing to files. Destination folder: ", t.OutputFolder)
	fmt.Println("--------------------------------------------------------------------------------------------")

	// iterate through each table and generate anything related
	if t.Tables != nil {

		for i := range t.Tables {

			// generate the table structure
			t.Tables[i].WriteToFile()

			// generate one-time only custom files
			// if they are already present, they will be skipped
			t.Tables[i].WriteToCustomFile()

		}
	} else {
		fmt.Println("Done: No tables found.")
	}

}

// Generates the base file of the package that contains initialization functions,
// convenience functions to get the database handle, query preparing, etc
func (t *ToolOptions) GenerateBaseFile() {

	tmpl, err := template.New("tableTemplate").Parse(BASE_TEMPLATE)
	if err != nil {
		log.Fatal("GenerateBaseFile() fatal error running template.New:", err)
	}

	var generatedTemplate bytes.Buffer
	err = tmpl.Execute(&generatedTemplate, t)
	if err != nil {
		log.Fatal("GenerateBaseFile() fatal error running template.Execute:", err)
	}

	var filePath string = t.OutputFolder + "/modelsBase.go"

	if FileExists(filePath) {
		fmt.Println("Skipping generating base file. Filepath: " + filePath + " already exists.")
	} else {
		err = ioutil.WriteFile(filePath, generatedTemplate.Bytes(), 0644)
		if err != nil {
			log.Fatal("GenerateBaseFile() - WriteToFile() fatal error writing to file:", err)
		}

		fmt.Println("Finished generating the base file. Filepath: " + filePath)
	}

}

func (t *ToolOptions) CollectTables() error {

	var currentTableName string

	rows, err := t.ConnectionPool.Query("SELECT table_name FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE';")

	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&currentTableName)
		if err != nil {
			log.Fatal("CollectTables fatal error inside rows.Next() iteration: ", err)
		}

		// instantiate a table struct
		currentTable := &Table{
			TableName:          currentTableName,
			GoFriendlyName:     GetGoFriendlyNameForTable(currentTableName),
			ConnectionPool:     t.ConnectionPool,
			Options:            t,
			GeneratedTemplate:  bytes.Buffer{},
			GenericSelectQuery: "",
			GenericInsertQuery: "",

			ColumnsString:   "",
			PKColumnsString: "",
			FKColumnsString: "",
		}

		currentTable.GoTypesToImport = make(map[string]string)

		// collect the columns for the table
		// colect all the column info
		if err := currentTable.CollectColumns(); err != nil {
			log.Fatal("CollectTables(): CollectColumns method for table ", currentTable.TableName, " FATAL error: ", err)
		}

		// collect the primary keys for the table
		if err := currentTable.CollectPrimaryKeys(); err != nil {
			log.Fatal("CollectTables(): CollectPrimaryKeys method for table ", currentTable.TableName, " FATAL error: ", err)
		}

		// generate the typical select sql queries
		currentTable.CreateGenericQueries()

		// add the table to the slice
		t.Tables = append(t.Tables, *currentTable)

	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	return nil

}

/* Util methods */

func GetGoFriendlyNameForTable(tableName string) string {

	// find if the table name has underscore
	if strings.Contains(tableName, "_") == false {
		return strings.Title(tableName)
	}

	subNames := strings.Split(tableName, "_")

	if subNames == nil {
		log.Fatal("GetGoFriendlyNameForTable() fatal error for table name: ", tableName, ". Please ensure a valid table name is provided.")
	}

	for i := range subNames {
		subNames[i] = strings.Title(subNames[i])
	}

	return strings.Join(subNames, "")
}
