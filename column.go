package main

import (
	"bytes"
	"fmt"
	"log"
	"text/template"

	"github.com/silviucm/pgx"
)

/* Column Section */

type Column struct {
	Options        *ToolOptions
	ConnectionPool *pgx.ConnPool
	ParentTable    *Table

	DbName          string
	DbComments      string
	OrdinalPosition int
	Type            string
	MaxLength       int
	DefaultValue    pgx.NullString
	Nullable        bool
	IsSequence      bool

	IsPK          bool
	IsCompositePK bool

	IsFK bool

	GoName                 string
	GoType                 string
	GoNullableType         string // e.g. "pgx.NullString"
	NullableTypeCreateFunc string // e.g. "pgx.CreateNullString"
	IsGuid                 bool

	ColumnComment string
}

func (col *Column) GeneratePKGetter(parentTable *Table) []byte {

	col.ParentTable = parentTable
	return col.getColumnTemplate("pkGetterTemplate", PK_GETTER_TEMPLATE_ATOMIC)
}

func (col *Column) GeneratePKGetterTx(parentTable *Table) []byte {

	col.ParentTable = parentTable
	return col.getColumnTemplate("pkGetterTemplate", PK_GETTER_TEMPLATE_TX)
}

func (col *Column) getColumnTemplate(templateName, templateContent string) []byte {

	tmpl, err := template.New(templateName).Funcs(fns).Parse(templateContent)
	if err != nil {
		log.Fatal("getColumnTemplate() fatal error running template.New for template ", templateName, ":", err)
	}

	var generatedTemplate bytes.Buffer
	err = tmpl.Execute(&generatedTemplate, col)
	if err != nil {
		log.Fatal("GeneratePKGetter() fatal error running template.Execute for template ", templateName, ":", err)
	}

	fmt.Println("PK Getter structure for column " + col.GoName + " generated.")
	return generatedTemplate.Bytes()
}
