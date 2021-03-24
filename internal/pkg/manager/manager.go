package manager

import "context"

type Manager interface {
	Deploy(context.Context, DeployArgs) error
	Delete(context.Context, DeleteArgs) error
}
