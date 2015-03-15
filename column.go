package main

import (
	"bytes"
	"fmt"
	"github.com/silviucm/pgx"
	"log"
	"text/template"
)

/* Column Section */

type Column struct {
	Options        *ToolOptions
	ConnectionPool *pgx.ConnPool
	ParentTable    *Table

	Name         string
	Type         string
	MaxLength    int
	DefaultValue pgx.NullString
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

	// if the table has more than one columns as PK, supply the alternative template
	var templateContent string = PK_GETTER_TEMPLATE_SINGLE_FIELD
	if parentTable.PKColumns != nil && len(parentTable.PKColumns) > 1 {
		templateContent = PK_GETTER_TEMPLATE_MULTI_FIELD
	}

	tmpl, err := template.New("pkGetterTemplate").Funcs(fns).Parse(templateContent)
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
