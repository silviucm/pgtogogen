package main

import (
	"bytes"
	"fmt"
	"log"
	"text/template"

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

// Generates a getter template for the unique constraint
func (c *Constraint) GenerateUniqueConstraintGetter(parentTable *Table) []byte {

	if c.IsUnique == false {
		return []byte{}
	}

	c.ParentTable = parentTable
	return c.getConstraintTemplate("uqGetterTemplate", UQ_GETTER_TEMPLATE_ATOMIC)
}

func (c *Constraint) GenerateUniqueConstraintGetterTx(parentTable *Table) []byte {

	if c.IsUnique == false {
		return []byte{}
	}

	c.ParentTable = parentTable
	return c.getConstraintTemplate("uqGetterTemplateTx", UQ_GETTER_TEMPLATE_TX)
}

func (c *Constraint) getConstraintTemplate(templateName, templateContent string) []byte {

	tmpl, err := template.New(templateName).Funcs(fns).Parse(templateContent)
	if err != nil {
		log.Fatal("getConstraintTemplate() fatal error running template.New for template ", templateName, ":", err)
	}

	var generatedTemplate bytes.Buffer
	err = tmpl.Execute(&generatedTemplate, c)
	if err != nil {
		log.Fatal("getConstraintTemplate() fatal error running template.Execute for template ", templateName, ":", err)
	}

	fmt.Println("UQ Getter structure for unique constraint " + c.DbName + " generated.")
	return generatedTemplate.Bytes()
}
