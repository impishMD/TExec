package projects

import (
	"github.com/impishMD/TExec/api/runners"
	"github.com/impishMD/TExec/services/server"
)

func NewProjectRunnerController(service server.RunnerService) *runners.Controller {
	return runners.NewController(service, true)
}
