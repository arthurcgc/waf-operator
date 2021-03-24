package manager

type DeployArgs struct {
	Name      string
	Replicas  int
	Namespace string
	ProxyPass string
}

type DeleteArgs struct {
	Name      string
	Namespace string
}
