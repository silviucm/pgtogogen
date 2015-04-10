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
	DbHost   string
	DbPort   uint16
	DbName   string
	DbUser   string
	DbPass   string
	DbSchema string

	OutputFolder string

	PackageName string

	GeneratePKGetters   bool
	GenerateGuidGetters bool

	ConnectionPool *pgx.ConnPool

	Tables []Table
	Views  []View

	Functions []Function

	// internal counter for materialized views
	noMaterializedViews int
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
		log.Fatal("Collect(): CollectTables fatal error: ", err)
	}

	// display the table collection summary
	if t.Tables != nil {
		fmt.Println("Done: Found " + strconv.Itoa(len(t.Tables)) + " tables.")
	} else {
		fmt.Println("Done: No tables found.")
	}

	// collect all the user views from the database
	fmt.Println(" ")
	fmt.Print("Collecting views...")
	if err := t.CollectViews(); err != nil {
		log.Fatal("Collect(): CollectViews fatal error: ", err)
	}

	// display the view collection summary
	if t.Views != nil {
		fmt.Println("Done: Found " + strconv.Itoa(len(t.Views)) + " views.")
	} else {
		fmt.Println("Done: No views found.")
	}

	// collect all the materialized views from the database
	fmt.Println(" ")
	fmt.Print("Collecting materialized views...")
	if err := t.CollectMaterializedViews(); err != nil {
		log.Fatal("Collect(): CollectMaterializedViews fatal error: ", err)
	}

	// display the view collection summary
	if t.noMaterializedViews > 0 {
		fmt.Println("Done: Found " + strconv.Itoa(t.noMaterializedViews) + " materialized views.")
	} else {
		fmt.Println("Done: No materialized views found.")
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
			log.Println("Beginning generation for table: ", t.Tables[i].DbName)
			fmt.Println("--------------------------------------------------------------------------------------------")

			// generate the table structure
			t.Tables[i].GenerateTableStruct()

			// generate the select statements
			t.Tables[i].GenerateSelectFunctions()

			// generate the insert-related functions
			t.Tables[i].GenerateInsertFunctions()

			// generate the update-related functions
			t.Tables[i].GenerateUpdateFunctions()

			// generate the delete-related functions
			t.Tables[i].GenerateDeleteFunctions()

			// generate the queries by PK
			if t.GeneratePKGetters == true {
				fmt.Println("Generating Primary Key Accessor Methods...")

				if t.Tables[i].PKColumns != nil {

					if len(t.Tables[i].PKColumns) > 0 {

						// the getter should only return one row,
						// no need to iterate here, just pass the first PK column,
						// the GeneratePKGetter template will render according to
						// the number of PK fields
						pkGetter := t.Tables[i].PKColumns[0].GeneratePKGetter(&t.Tables[i])
						if _, writeErr := t.Tables[i].GeneratedTemplate.Write(pkGetter); writeErr != nil {
							log.Fatal("Generate fatal error writing bytes from the GeneratePKGetter call: ", writeErr)
						}

						pkGetterTx := t.Tables[i].PKColumns[0].GeneratePKGetterTx(&t.Tables[i])
						if _, writeErrTx := t.Tables[i].GeneratedTemplate.Write(pkGetterTx); writeErrTx != nil {
							log.Fatal("Generate fatal error writing bytes from the GeneratePKGetterTx call: ", writeErrTx)
						}

					}

				}
			}
		}
	} else {
		fmt.Println("Done: No tables found.")
	}

	// iterate through each view and generate anything related
	if t.Views != nil {

		for i := range t.Views {

			fmt.Println("--------------------------------------------------------------------------------------------")
			log.Println("Beginning generation for view: ", t.Views[i].DbName)
			fmt.Println("--------------------------------------------------------------------------------------------")

			// generate the table structure
			t.Views[i].GenerateViewStruct()

			// generate the select statements
			t.Views[i].GenerateSelectFunctions()

		}
	} else {
		fmt.Println("Done: No views found.")
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

	// iterate through each view and generate anything related
	if t.Views != nil {

		for i := range t.Views {

			// generate the table structure
			t.Views[i].WriteToFile()

			// generate one-time only custom files
			// if they are already present, they will be skipped
			t.Views[i].WriteToCustomFile()

		}
	} else {
		fmt.Println("Done: No views found.")
	}
}

func (t *ToolOptions) WriteBaseFiles() {

	t.writeBaseTemplateFile("main base file", BASE_TEMPLATE, "modelsBase.go", true)
	t.writeBaseTemplateFile("db settings base file", BASE_TEMPLATE_SETTINGS, "modelsDbSettings.go", false)
	t.writeBaseTemplateFile("collections base file", BASE_TEMPLATE_COLLECTIONS, "modelsCollections.go", false)
	t.writeBaseTemplateFile("collections base file", BASE_TEMPLATE_FORMS, "modelsForms.go", false)

}

// Generates the base file of the package that contains initialization functions,
// convenience functions to get the database handle, query preparing, etc

func (t *ToolOptions) writeBaseTemplateFile(templateName, templateContent string, baseFilename string, overwritable bool) {

	tmpl, err := template.New(templateName).Funcs(fns).Parse(templateContent)
	if err != nil {
		log.Fatal("writeBaseTemplateFileFile() fatal error running template.New:", err)
	}

	var generatedTemplate bytes.Buffer
	err = tmpl.Execute(&generatedTemplate, t)
	if err != nil {
		log.Fatal("writeBaseTemplateFileFile() fatal error running template.Execute:", err)
	}

	var filePath string = t.OutputFolder + "/" + baseFilename

	if overwritable {

		err = ioutil.WriteFile(filePath, generatedTemplate.Bytes(), 0644)
		if err != nil {
			log.Fatal("GenerateBaseFile() - WriteToFile() fatal error writing to file:", err)
		}

		fmt.Println("Finished generating the " + templateName + " base file. Filepath: " + filePath)

	} else {

		if FileExists(filePath) {
			fmt.Println("Skipping generating base file. Filepath: " + filePath + " already exists.")
		} else {
			err = ioutil.WriteFile(filePath, generatedTemplate.Bytes(), 0644)
			if err != nil {
				log.Fatal("GenerateBaseFile() - WriteToFile() fatal error writing to file:", err)
			}

			fmt.Println("Finished generating the " + templateName + " base file. Filepath: " + filePath)
		}
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
			DbName:             currentTableName,
			GoFriendlyName:     GetGoFriendlyNameForTable(currentTableName),
			ConnectionPool:     t.ConnectionPool,
			Options:            t,
			GeneratedTemplate:  bytes.Buffer{},
			GenericSelectQuery: "",
			GenericInsertQuery: "",

			ColumnsString:   "",
			PKColumnsString: "",
			FKColumnsString: "",
			IsTable:         true,
		}

		currentTable.GoTypesToImport = make(map[string]string)

		// collect the columns for the table
		// colect all the column info
		if err := currentTable.CollectColumns(); err != nil {
			log.Fatal("CollectTables(): CollectColumns method for table ", currentTable.DbName, " FATAL error: ", err)
		}

		// collect the primary keys for the table
		if err := currentTable.CollectPrimaryKeys(); err != nil {
			log.Fatal("CollectTables(): CollectPrimaryKeys method for table ", currentTable.DbName, " FATAL error: ", err)
		}

		// collect the unique constraints for the table
		if err := currentTable.CollectUniqueConstraints(); err != nil {
			log.Fatal("CollectTables(): CollectUniqueConstraints method for table ", currentTable.DbName, " FATAL error: ", err)
		}

		// generate the typical select sql queries
		currentTable.CreateGenericQueries()

		// collect the comments for the table and the columns
		/* BUGS if constraint comments are inserted - to find how to tie comments to fields specifically
		if err = currentTable.CollectComments(); err != nil {
			log.Fatal("CollectTables(): CollectComments method for table ", currentTable.DbName, " FATAL error: ", err)
		}
		*/

		// add the table to the slice
		t.Tables = append(t.Tables, *currentTable)

	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	return nil

}

func (t *ToolOptions) CollectViews() error {

	var currentViewName string

	rows, err := t.ConnectionPool.Query("SELECT table_name FROM information_schema.views WHERE table_schema=$1 AND table_catalog=$2", t.DbSchema, t.DbName)

	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&currentViewName)
		if err != nil {
			log.Fatal("CollectViews fatal error inside rows.Next() iteration: ", err)
		}

		// instantiate a table struct
		currentView := &View{
			DbName:             currentViewName,
			GoFriendlyName:     GetGoFriendlyNameForTable(currentViewName),
			ConnectionPool:     t.ConnectionPool,
			Options:            t,
			GeneratedTemplate:  bytes.Buffer{},
			GenericSelectQuery: "",
			GenericInsertQuery: "",

			ColumnsString:  "",
			IsMaterialized: false,
			IsTable:        false,
		}

		currentView.GoTypesToImport = make(map[string]string)

		// collect the columns for the view
		// colect all the column info
		if err := currentView.CollectColumns(); err != nil {
			log.Fatal("CollectViews(): CollectColumns method for table ", currentView.DbName, " FATAL error: ", err)
		}

		// generate the typical select sql queries
		currentView.CreateGenericQueries()

		// add the view to the slice
		t.Views = append(t.Views, *currentView)

	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	return nil

}

func (t *ToolOptions) CollectMaterializedViews() error {

	// materialized views cannot (as of March 2015) be extracted easily from information schema
	// the query is very complicated, as below
	var materializedViewsQuery = `SELECT table_name FROM
(SELECT 
    NULL AS TABLE_CAT, 
    n.nspname AS TABLE_SCHEM, 
    c.relname AS TABLE_NAME,  
    CASE n.nspname ~ '^pg_' OR n.nspname = 'information_schema'  
        WHEN true THEN
            CASE  
                WHEN n.nspname = 'pg_catalog' OR n.nspname = 'information_schema' THEN
                    CASE c.relkind
                        WHEN 'r' THEN 'SYSTEM TABLE'   
                        WHEN 'v' THEN 'SYSTEM VIEW'   
                        WHEN 'i' THEN 'SYSTEM INDEX'   
                        ELSE NULL  
                    END 
                WHEN n.nspname = 'pg_toast' THEN
                    CASE c.relkind   
                        WHEN 'r' THEN 'SYSTEM TOAST TABLE'   
                        WHEN 'i' THEN 'SYSTEM TOAST INDEX'
                        ELSE NULL
                    END 
                ELSE
                    CASE c.relkind
                        WHEN 'r' THEN 'TEMPORARY TABLE'   
                        WHEN 'i' THEN 'TEMPORARY INDEX'   
                        WHEN 'S' THEN 'TEMPORARY SEQUENCE'   
                        WHEN 'v' THEN 'TEMPORARY VIEW'
                        ELSE NULL   
                    END  
            END  
        WHEN false THEN
            CASE c.relkind  
                WHEN 'r' THEN 'TABLE'  
                WHEN 'i' THEN 'INDEX'  
                WHEN 'S' THEN 'SEQUENCE'  
                WHEN 'v' THEN 'VIEW'  
                WHEN 'c' THEN 'TYPE'  
                WHEN 'f' THEN 'FOREIGN TABLE'  
                WHEN 'm' THEN 'MATERIALIZED VIEW'  
                ELSE NULL
            END  
        ELSE NULL 
    END  
        AS TABLE_TYPE,
    d.description AS REMARKS,
    c.relkind

FROM
    pg_catalog.pg_namespace n, pg_catalog.pg_class c
    LEFT JOIN pg_catalog.pg_description d ON (c.oid = d.objoid AND d.objsubid = 0)  
    LEFT JOIN pg_catalog.pg_class dc ON (d.classoid=dc.oid AND dc.relname='pg_class')  
    LEFT JOIN pg_catalog.pg_namespace dn ON (dn.oid=dc.relnamespace AND dn.nspname='pg_catalog') 

WHERE
    c.relnamespace = n.oid
    AND c.relname LIKE '%'
    AND n.nspname <> 'pg_catalog'
    AND n.nspname <> 'information_schema'
    AND c.relkind IN ('r', 'v', 'm', 'f')

ORDER BY
    TABLE_TYPE,
    TABLE_SCHEM,
    TABLE_NAME) t WHERE t.table_type = 'MATERIALIZED VIEW' AND TABLE_SCHEM = $1
	`

	var currentViewName string

	rows, err := t.ConnectionPool.Query(materializedViewsQuery, t.DbSchema)

	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&currentViewName)
		if err != nil {
			log.Fatal("CollectViews fatal error inside rows.Next() iteration: ", err)
		}

		// instantiate a table struct
		currentView := &View{
			DbName:             currentViewName,
			GoFriendlyName:     GetGoFriendlyNameForTable(currentViewName),
			ConnectionPool:     t.ConnectionPool,
			Options:            t,
			GeneratedTemplate:  bytes.Buffer{},
			GenericSelectQuery: "",
			GenericInsertQuery: "",

			ColumnsString:  "",
			IsMaterialized: true,
		}

		currentView.GoTypesToImport = make(map[string]string)

		// collect the columns for the view
		// colect all the column info
		if err := currentView.CollectMaterializedViewColumns(); err != nil {
			log.Fatal("CollectViews(): CollectMaterializedViewColumns method for table ", currentView.DbName, " FATAL error: ", err)
		}

		// generate the typical select sql queries
		currentView.CreateGenericQueries()

		// add the view to the slice
		t.Views = append(t.Views, *currentView)

		t.noMaterializedViews = t.noMaterializedViews + 1

	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	return nil

}

func (t *ToolOptions) CollectFunctions() error {

	var currentFunctionName string

	var functionNamesQuery string = `SELECT r.routine_name FROM information_schema.routines r
			WHERE r.routine_schema=$1 AND routine_catalog=$2 AND r.routine_type = 'FUNCTION'
			ORDER BY r.routine_name;`

	rows, err := t.ConnectionPool.Query(functionNamesQuery, t.DbSchema, t.DbName)

	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&currentFunctionName)
		if err != nil {
			log.Fatal("CollectFunctions fatal error inside rows.Next() iteration: ", err)
		}

		// instantiate a function struct and also collect all the necessary information
		currentFunction, err := CollectFunction(t, currentFunctionName)
		if err != nil {
			log.Fatal("CollectFunctions fatal error inside a CollectFunction() routine: ", err)
		}

		// add the function to the slice if not nil
		if currentFunction != nil {
			t.Functions = append(t.Functions, *currentFunction)
		}

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
