package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/silviucm/pgtogogen/internal/pgx"
)

type ToolOptions struct {
	DbHost   string
	DbPort   uint16
	DbName   string
	DbUser   string
	DbPass   string
	DbSchema string

	DbMajorVersion int
	DbMinorVersion int

	OutputFolder            string
	CreateFolderIfNotExists bool

	PackageName string

	PgxImport    string // (the full import path e.g. "github.com/jackc/pgx")
	PgTypeImport string // (the full import path e.g. "github.com/jackc/pgx/pgtype")

	GenerateFunctions   bool
	GeneratePKGetters   bool
	GenerateUQGetters   bool
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

	t.ConnectionPool = connPool

	// get the database version
	majorVersion, minorVersion := t.GetPostgresVersion()
	t.DbMajorVersion = majorVersion
	t.DbMinorVersion = minorVersion
	log.Println("Database version: ", t.DbMajorVersion, ".", t.DbMinorVersion)
	fmt.Println("--------------------------------------------------------------------------------------------")

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

	// collect all the user functions from the database
	if t.DbMajorVersion <= 9 && t.DbMinorVersion < 4 {
		fmt.Print("SKIPPING Collecting functions because Postgres versions before 9.4 do not suport parameter_default inside the informaion schema parameters view.\nFor more details see:\nhttps://www.postgresql.org/docs/9.5/static/infoschema-parameters.html\n")
	} else {
		if t.GenerateFunctions {
			fmt.Print("Collecting functions...")
			if err := t.CollectFunctions(); err != nil {
				log.Fatal("Collect(): CollectFunctions fatal error: ", err)
			}
		} else {
			fmt.Println("Skipping collecting functions...")
		}

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

			// generate the bulk-copy-related functions
			t.Tables[i].GenerateBulkCopyFunctions()

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

			// if the unique constraints getters generate flag is true, then
			// generate those as well
			if t.GenerateUQGetters == true {
				fmt.Println("Generating Unique Constraints Accessor Methods...")

				if t.Tables[i].UniqueConstraints != nil && len(t.Tables[i].UniqueConstraints) > 0 {

					for cIdx, _ := range t.Tables[i].UniqueConstraints {

						// non-transactional getter
						uqGetter := t.Tables[i].UniqueConstraints[cIdx].GenerateUniqueConstraintGetter(&t.Tables[i])
						if _, writeErr := t.Tables[i].GeneratedTemplate.Write(uqGetter); writeErr != nil {
							log.Fatal("Generate fatal error writing bytes from the GenerateUniqueConstraintGetter call: ", writeErr)
						}

						// transactional getter
						uqGetterTx := t.Tables[i].UniqueConstraints[cIdx].GenerateUniqueConstraintGetterTx(&t.Tables[i])
						if _, writeErrTx := t.Tables[i].GeneratedTemplate.Write(uqGetterTx); writeErrTx != nil {
							log.Fatal("Generate fatal error writing bytes from the GenerateUniqueConstraintGetterTx call: ", writeErrTx)
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

	// iterate through each function and generate it
	if t.Functions != nil {

		if len(t.Functions) > 0 {

			for i := range t.Functions {

				fmt.Println("--------------------------------------------------------------------------------------------")
				log.Println("Beginning generation for function: ", t.Functions[i].DbName)
				fmt.Println("--------------------------------------------------------------------------------------------")

				t.Functions[i].Generate()
			}
		}

	}

}

// MkDir creates a folder in the path indicated by t.OutputFolder, if the
// folder does not exist.
func (t *ToolOptions) MkDir() (err error) {

	if _, err = os.Stat(t.OutputFolder); os.IsNotExist(err) {
		return os.MkdirAll(t.OutputFolder, 0755)
	}

	if err != nil {
		return err
	}
	return nil
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

	// iterate through each function and write it to the functions file
	if t.Functions != nil {

		if len(t.Functions) > 0 {

			functionFileBuffer := bytes.Buffer{}

			generateFunctionFilePrefix(t, &functionFileBuffer)

			for i := range t.Functions {

				// generate the table structure
				t.Functions[i].WriteToBuffer(&functionFileBuffer)
			}

			t.WriteFunctionFiles(&functionFileBuffer)
		}

	} else {
		fmt.Println("Done: No functions found.")
	}
}

func (t *ToolOptions) WriteBaseFiles() {

	t.writeBaseTemplateFile("main base file", BASE_TEMPLATE, t.PackageName+"_pgtogogen_base.go", true)
	t.writeBaseTemplateFile("db settings base file", BASE_TEMPLATE_SETTINGS, t.PackageName+"_pgtogogen_db.go", false)
	t.writeBaseTemplateFile("collections base file", BASE_TEMPLATE_COLLECTIONS, t.PackageName+"_pgtogogen_coll.go", false)
	t.writeBaseTemplateFile("collections base file", BASE_TEMPLATE_FORMS, t.PackageName+"_pgtogogen_forms.go", false)
	t.writeBaseTemplateFile("collections base file", BASE_TRANSACTIONS, t.PackageName+"_pgtogogen_tx.go", false)
	t.writeBaseTemplateFile("collections base file", BASE_DB_TYPES, t.PackageName+"_pgtogogen_types.go", false)
	t.writeBaseTemplateFile("collections base file", BASE_BULK_COPY, t.PackageName+"_pgtogogen_copy.go", false)

}

func (t *ToolOptions) WriteFunctionFiles(functionFileBuffer *bytes.Buffer) {

	var filePath string = t.OutputFolder + "/" + t.PackageName + "DbFunctions.go"
	var overwritable bool = true

	if overwritable {

		err := ioutil.WriteFile(filePath, functionFileBuffer.Bytes(), 0644)
		if err != nil {
			log.Fatal("WriteFunctionFiles() - WriteToFile() fatal error writing to file:", err)
		}

		fmt.Println("Finished generating the functions file. Filepath: " + filePath)

	} else {

		if FileExists(filePath) {
			fmt.Println("Skipping generating functions file. Filepath: " + filePath + " already exists.")
		} else {
			err := ioutil.WriteFile(filePath, functionFileBuffer.Bytes(), 0644)
			if err != nil {
				log.Fatal("WriteFunctionFiles() - fatal error writing to file:", err)
			}

			fmt.Println("Finished generating the function file. Filepath: " + filePath)
		}
	}

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
		if err = currentTable.CollectComments(); err != nil {
			log.Fatal("CollectTables(): CollectComments method for table ", currentTable.DbName, " FATAL error: ", err)
		}

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
	var currentFunctionSpecificName string

	var duplicateFuncNameMap map[string]int = make(map[string]int)

	// The routine_name column is the "friendly" name (not guaranteed to be unique).
	// The specific_name is the unique name.
	// e.g. a hello_world function with multiple signatures, would have the
	// "hello_world" value in the routing_name column for all records, but unique,
	// number-prefixed names (such as "hello_world_18534") in the specific_name field.
	var functionNamesQuery string = `SELECT r.routine_name, r.specific_name FROM information_schema.routines r
			WHERE r.routine_schema=$1 AND routine_catalog=$2 AND r.routine_type = 'FUNCTION'
			ORDER BY r.routine_name;`

	// log.Printf("Collect functions main query:\n%s\nwith schema: %s and catalog: %s", functionNamesQuery, t.DbSchema, t.DbName)

	rows, err := t.ConnectionPool.Query(functionNamesQuery, t.DbSchema, t.DbName)

	if err != nil {
		log.Fatal("CollectFunctions fatal error when executing t.ConnectionPool.Query: ", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&currentFunctionName, &currentFunctionSpecificName)
		if err != nil {
			log.Fatal("CollectFunctions fatal error inside rows.Next() iteration: ", err)
		}

		count := duplicateFuncNameMap[currentFunctionName]
		count = count + 1

		// instantiate a function struct and also collect all the necessary information
		currentFunction, err := CollectFunction(t, currentFunctionName, currentFunctionSpecificName, count)
		if err != nil {
			log.Printf("CollectFunctions(\"%s\") error: %s\n", currentFunctionName, err.Error())
			continue
		}

		// add the function to the slice if not nil
		if currentFunction != nil {
			t.Functions = append(t.Functions, *currentFunction)
			// Update the duplicate count
			duplicateFuncNameMap[currentFunctionName] = count
		}

	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	return nil

}

/* Util methods */

// Retrieves the current PostgreSQL version
// For "SELECT version();"  the returned string should look something like: "PostgreSQL 9.3.6, compiled by Visual C++ build 1600, 64-bit"
// For "SHOW server_version;", the result should be something like: "9.3.6"
func (t *ToolOptions) GetPostgresVersion() (majorVersion int, minorVersion int) {

	minorVersion = -1
	majorVersion = -1
	var err error
	var minorVersionInt64, majorVersionInt64 int64

	var pgVersion string

	var selectVersionQuery string = "SHOW server_version;"

	versionRow := t.ConnectionPool.QueryRow(selectVersionQuery)

	// For "SELECT version();"  the returned string should look something like: "PostgreSQL 9.3.6, compiled by Visual C++ build 1600, 64-bit"
	// For "SHOW server_version", the result should be something like: "9.3.6"
	if versionRow != nil {
		scanErr := versionRow.Scan(&pgVersion)
		if scanErr != nil {
			log.Fatal("GetPostgresVersion() fatal error at versionRow.Scan: ", scanErr)
			return
		}
		// try to split based on dots
		versions := strings.Split(pgVersion, ".")
		if (len(versions)) == 0 {
			return
		}
		if len(versions) > 0 {
			majorVersionInt64, err = strconv.ParseInt(versions[0], 10, 64)
			if err != nil {
				log.Fatal("GetPostgresVersion() error parsing major version: ", err)
			}
		}
		if len(versions) > 1 {
			minorVersionInt64, err = strconv.ParseInt(versions[1], 10, 64)
			if err != nil {
				log.Fatal("GetPostgresVersion() error parsing minor version: ", err)
			}
		}

		minorVersion = int(minorVersionInt64)
		majorVersion = int(majorVersionInt64)
		return

	} else {
		log.Fatal("GetPostgresVersion fatal error: returned row supposed to contain the version number is nil")
	}

	return
}

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
