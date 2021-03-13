package internal

var wafImage = "owasp/modsecurity"

type DeployOpts struct {
	Name      string
	Replicas  int
	Namespace string
}
