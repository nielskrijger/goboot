package template_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nielskrijger/goboot/template"
	"github.com/stretchr/testify/assert"
)

func TestNewEmailTemplate_Success(t *testing.T) {
	l := template.NewTemplateLoader("../testdata/templates")
	tmpl, err := l.LoadTemplate("one")
	assert.Nil(t, err)

	var b bytes.Buffer
	err = tmpl.ExecuteTemplate(&b, "layout1", nil)
	assert.Nil(t, err)
	assert.Equal(t, `
layout: layout1
content: 
    name: template-one
    partial1: partial1
    partial2: partial2`, strings.TrimRight(b.String(), "\n"))
}

func TestNewEmailTemplate_LoadLayout2(t *testing.T) {
	l := template.NewTemplateLoader("../testdata/templates")
	tmpl, err := l.LoadTemplate("one")
	assert.Nil(t, err)

	var b bytes.Buffer
	err = tmpl.ExecuteTemplate(&b, "layout2", nil)
	assert.Nil(t, err)
	assert.Equal(t, `
layout: layout2
content: 
    name: template-one
    partial1: partial1
    partial2: partial2`, strings.TrimRight(b.String(), "\n"))
}

func TestNewEmailTemplate_InvalidTemplatesDir(t *testing.T) {
	l := template.NewTemplateLoader("./testdata/unknown")
	_, err := l.LoadTemplate("one")
	assert.Error(t, err, "open ./testdata/unknown: no such file or directory")
}

func TestNewEmailTemplate_InvalidPartialsDir(t *testing.T) {
	l := template.NewTemplateLoader("../testdata/templates")
	l.PartialsDir = "/unknown/partials"
	_, err := l.LoadTemplate("one")
	assert.Error(t, err, "open ../testdata/templates/unknown: no such file or directory")
}

func TestNewEmailTemplate_InvalidLayoutsDir(t *testing.T) {
	l := template.NewTemplateLoader("../testdata/templates")
	l.LayoutsDir = "/unknown/layouts"
	_, err := l.LoadTemplate("one")
	assert.Error(t, err, "open ../testdata/templates/unknown: no such file or directory")
}

func TestLoadAllTemplates_Success(t *testing.T) {
	l := template.NewTemplateLoader("../testdata/templates")
	tmpl, err := l.LoadAllTemplates()

	assert.Nil(t, err)
	assert.Len(t, tmpl, 2)
	assert.NotNil(t, tmpl["one"])
	assert.NotNil(t, tmpl["two"])
}

func TestLoadAllTemplates_InvalidTemplatesDir(t *testing.T) {
	l := template.NewTemplateLoader("../testdata/unknown")
	_, err := l.LoadAllTemplates()
	assert.Error(t, err, "open ./testdata/unknown: no such file or directory")
}

func TestLoadAllTemplates_InvalidPartialsDir(t *testing.T) {
	l := template.NewTemplateLoader("../testdata/templates")
	l.PartialsDir = "/unknown/partials"
	_, err := l.LoadAllTemplates()
	assert.Error(t, err, "open ../testdata/templates/unknown: no such file or directory")
}
