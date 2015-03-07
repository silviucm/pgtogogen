package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"text/template"
)

/* Column Section */

type Column struct {
	Options     *ToolOptions
	DbHandle    *sql.DB
	ParentTable *Table

	Name         string
	Type         string
	MaxLength    int
	DefaultValue sql.NullString
	Nullable     bool
	IsSequence   bool

	IsPK          bool
	IsCompositePK bool

	IsFK bool

	GoName string
	GoType string

	ColumnComment string
}

func (col *Column) GeneratePKGetter(parentTable *Table) []byte {

	col.ParentTable = parentTable

	var fns = template.FuncMap{
		"plus1": func(x int) int {
			return x + 1
		},
	}

	tmpl, err := template.New("pkGetterTemplate").Funcs(fns).Parse(PK_GETTER_TEMPLATE)
	if err != nil {
		log.Fatal("GeneratePKGetter() fatal error running template.New:", err)
	}

	var generatedTemplate bytes.Buffer
	err = tmpl.Execute(&generatedTemplate, col)
	if err != nil {
		log.Fatal("GeneratePKGetter() fatal error running template.Execute:", err)
	}

	fmt.Println("PK Getter structure for column " + col.GoName + " generated.")
	return generatedTemplate.Bytes()

}
