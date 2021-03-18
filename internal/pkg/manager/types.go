package manager

type DeployArgs struct {
	WAFName      string
	Replicas     int
	Namespace    string
	ProxyPass    string
	MainConfName string
	WAFConfName  string
}
