package main

// Usage: pgtogogen -h=yourhostnameoripaddress -n=yourdatabasename -u=yourusername -pass=yourpassword

import (
	"flag"
	"fmt"
	"strconv"
)

const ARGS_ERROR_HEADER string = "\n-------------------------\nARGUMENTS ERROR:\n-------------------------\n"

var dbHost, dbPort, dbName, dbUser, dbPass, outputFolder, packageName *string
var generatePKGetters, generateGuidGetters *bool

var dbPortUInt16 uint16 = 5432

func main() {

	// collect command-line flags

	// db settings flag
	dbHost = flag.String("h", "localhost", "database host, defaults to localhost if empty")
	dbPort = flag.String("port", "5432", "database port, defaults to 5432 if left empty")
	dbName = flag.String("n", "", "database name")
	dbUser = flag.String("u", "", "database user name")
	dbPass = flag.String("pass", "", "database password")

	// location settings
	outputFolder = flag.String("o", "./models", "the output folder to generate the db structures, defaults to models")

	// package settings
	packageName = flag.String("pkg", "models", "the package name for the generated files")

	// output settings
	generatePKGetters = flag.Bool("pk", true, "generate pk select methods, defaults to true")
	generateGuidGetters = flag.Bool("guid", true, "generate guid columns select methods, defaults to true")

	flag.Parse()

	// validate and exit if not true
	if validateFlags() == false {
		return
	}

	// assign the options to a ToolOptions struct
	options := &ToolOptions{
		DbHost: *dbHost,
		DbPort: dbPortUInt16,
		DbName: *dbName,
		DbUser: *dbUser,
		DbPass: *dbPass,

		OutputFolder: *outputFolder,
		PackageName:  *packageName,

		GeneratePKGetters:   *generatePKGetters,
		GenerateGuidGetters: *generateGuidGetters}

	// initialize the database and acquire the database handle
	db, err := options.InitDatabase()
	if err != nil {
		if db != nil {
			db.Close()
		}
		// exit here
		fmt.Println("Exiting here.")
		return

	}

	// make sure the db gets closed eventually
	defer func() {

		if db != nil {
			db.Close()
		}
	}()

	// start collecting db info
	options.Collect()

	// start generating
	options.Generate()

	// start writing to files
	options.WriteFiles()

	// write the base file
	options.WriteBaseFiles()

}

func validateFlags() bool {
	// BEGIN: Perform flags validation
	var flagParsingErrors string = ""

	if *dbName == "" {
		flagParsingErrors = flagParsingErrors + "Missing database name flag -n\n"
	}

	if *dbUser == "" {
		flagParsingErrors = flagParsingErrors + "Missing database user flag -u\n"
	}

	if *dbPass == "" {
		flagParsingErrors = flagParsingErrors + "Missing database password flag -pass\n"
	}

	// make sure the port is uint
	var portUInt16 int
	portUInt16, err := strconv.Atoi(*dbPort)
	if err != nil {
		flagParsingErrors = flagParsingErrors + "Invalid database port (please specify a valid number)\n"
	} else {
		dbPortUInt16 = uint16(portUInt16)
	}

	if flagParsingErrors != "" {
		flagParsingErrors = ARGS_ERROR_HEADER + flagParsingErrors
		fmt.Println(flagParsingErrors)
		return false
	}
	return true
}
