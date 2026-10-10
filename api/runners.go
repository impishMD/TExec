package api

import (
	"github.com/impishMD/TExec/api/runners"
	"github.com/impishMD/TExec/services/server"
)

// GlobalRunnerController exposes administrator management of all runners.
type GlobalRunnerController = runners.Controller

func NewGlobalRunnerController(service server.RunnerService) *GlobalRunnerController {
	return runners.NewController(service, false)
}
