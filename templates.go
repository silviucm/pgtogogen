package main

const TABLE_TEMPLATE = `package main

import (
	"database/sql"
	_ "github.com/lib/pq"
)

const {{.GoFriendlyName}}_DB_TABLE_NAME string = "{{.TableName}}"

type {{.GoFriendlyName}} struct {
	
}`
