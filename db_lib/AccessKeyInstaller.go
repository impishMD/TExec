package db_lib

import (
	"github.com/impishMD/TExec/db"
	"github.com/impishMD/TExec/pkg/ssh"
	"github.com/impishMD/TExec/pkg/task_logger"
)

type AccessKeyInstaller interface {
	Install(key db.AccessKey, usage db.AccessKeyRole, logger task_logger.Logger) (installation ssh.AccessKeyInstallation, err error)
}
