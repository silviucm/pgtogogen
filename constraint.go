package main

import (
	"github.com/silviucm/pgx"
)

/* Constraint Section */

type Constraint struct {
	Options        *ToolOptions
	ConnectionPool *pgx.ConnPool
	ParentTable    *Table

	DbName     string
	DbComments string
	Type       string

	Columns []Column

	IsPK          bool
	IsCompositePK bool
	IsFK          bool
	IsUnique      bool
}
