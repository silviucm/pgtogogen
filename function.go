package main

import (
	"github.com/silviucm/pgx"
	"log"
)

/* Function Section */

type Function struct {
	Options        *ToolOptions
	ConnectionPool *pgx.ConnPool

	DbName         string
	GoFriendlyName string
	DbComments     string

	Parameters []FunctionParameter

	ReturnType          string
	IsReturnUserDefined bool
	IsReturnASet        bool
	IsReturnARecord     bool
	IsReturnTable       bool
	IsReturnView        bool
}

type FunctionParameter struct {
	DbName         string
	GoFriendlyName string
	DbComments     string

	// can be "Input", "Output", "Variant"
	ParamType string

	ParamDataType string

	IsOptional   bool
	DefaultValue string
}

const (
	FUNC_PARAM_TYPE_INPUT   = "Input"
	FUNC_PARAM_TYPE_OUTPUT  = "Output"
	FUNC_PARAM_TYPE_INOUT   = "InOut"
	FUNC_PARAM_TYPE_VARIANT = "Variant"
)

func CollectFunction(t *ToolOptions, functionName string) (Function, error) {

	// for more info, check this url
	// http://www.alberton.info/postgresql_meta_info.html

	// the general function details query
	var functionDetailsQuery string = `SELECT r.routine_name, r.data_type, r.type_udt_name FROM information_schema.routines r
WHERE r.routine_schema=$1 AND routine_catalog=$2 AND r.routine_type = 'FUNCTION'
ORDER BY r.routine_name;

SELECT routines.* FROM information_schema.routines
WHERE routines.specific_schema='public' AND routine_type = 'FUNCTION'
ORDER BY routines.routine_name;


SELECT routines.routine_name, parameters.*
FROM information_schema.routines
    JOIN information_schema.parameters ON routines.specific_name=parameters.specific_name
WHERE routines.specific_schema='public'
ORDER BY routines.routine_name, parameters.ordinal_position;
	`

	rows, err := t.ConnectionPool.Query(functionDetailsQuery, t.DbSchema, t.DbName)

	if err != nil {
		log.Fatal("CollectColumns() fatal error running the query:", err)
	}
	defer rows.Close()

	// todo
	return Function{}, nil

}
