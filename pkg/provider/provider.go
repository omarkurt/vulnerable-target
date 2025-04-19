package provider

type Provider interface {
	Name() string
	Start()
	Stop()
}
