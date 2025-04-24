package provider

import "github.com/happyhackingspace/vulnerable-target/pkg/templates"

type Provider interface {
	Name() string
	Start(template *templates.Template) error
	Stop(template *templates.Template) error
}
