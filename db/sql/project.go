package sql

import (
	"encoding/json"
	"github.com/Masterminds/squirrel"
	"github.com/impishMD/TExec/db"
	"github.com/impishMD/TExec/pkg/taskmessage"
	"github.com/impishMD/TExec/pkg/tz"
)

func (d *SqlDb) CreateProject(project db.Project) (newProject db.Project, err error) {
	if err = project.ValidateIcon(); err != nil {
		return
	}
	if _, err = taskmessage.Parse(project.NotificationTemplates); err != nil {
		return
	}
	project.Created = tz.Now()

	insertId, err := d.insert(
		"id",
		"insert into project(name, created, type, alert, alert_chat, max_parallel_tasks, icon, notification_templates) values (?, ?, ?, ?, ?, ?, ?, ?)",
		project.Name, project.Created, project.Type, project.Alert, project.AlertChat, project.MaxParallelTasks, project.Icon, project.NotificationTemplates)

	if err != nil {
		return
	}

	newProject = project
	newProject.ID = insertId
	return
}

func (d *SqlDb) GetAllProjects() (projects []db.Project, err error) {
	query, args, err := squirrel.Select("p.*").
		From("project as p").
		OrderBy("p.name").
		Limit(200).
		ToSql()

	if err != nil {
		return
	}

	_, err = d.selectAll(&projects, query, args...)

	return
}

func (d *SqlDb) GetProjects(userID int) (projects []db.Project, err error) {
	query, args, err := squirrel.Select("p.*").
		From("project as p").
		Join("project__user as pu on pu.project_id=p.id").
		Where("pu.user_id=?", userID).
		OrderBy("p.name").
		Limit(200).
		ToSql()

	if err != nil {
		return
	}

	_, err = d.selectAll(&projects, query, args...)

	return
}

func (d *SqlDb) GetProject(projectID int) (project db.Project, err error) {
	query, args, err := squirrel.Select("p.*").
		From("project as p").
		Where("p.id=?", projectID).
		ToSql()

	if err != nil {
		return
	}

	err = d.selectOne(&project, query, args...)

	return
}

func (d *SqlDb) DeleteProject(projectID int) error {

	//tpls, err := d.GetTemplates(projectID, db.TemplateFilter{}, db.RetrieveQueryParams{})
	//
	//if err != nil {
	//	return err
	//}
	// TODO: sort projects

	tx, err := d.Sql().Begin()

	if err != nil {
		return err
	}

	statements := []string{
		"delete from project__alert_channel where project_id=?",
		"delete from project__workflow_run where project_id=?",
		"delete from project__workflow_template where project_id=?",
		"update project__template set build_template_id = null where project_id=?",
		"delete from project__template where project_id=?",
		"delete from project__user where project_id=?",
		"delete from project__repository where project_id=?",
		"delete from project__inventory where project_id=?",
		// Before the keys: a mapping references access_key without an ON DELETE
		// action, so the key deletion below is rejected while one still exists.
		// The restriction stays, so deleting a key on its own still reports the
		// mappings using it.
		"delete from project__host_config where project_id=?",
		"delete from project__environment_expression where environment_id in (select id from project__environment where project_id=?)",
		"delete from project__environment_key where environment_id in (select id from project__environment where project_id=?)",
		"delete from access_key where project_id=?",
	}

	for _, statement := range statements {
		_, err = tx.Exec(d.PrepareQuery(statement), projectID)

		if err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	// The project row decides: a project with no children is still a real delete.
	err = requireDeletedRow(tx.Exec(d.PrepareQuery("delete from project where id=?"), projectID))
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *SqlDb) UpdateProject(project db.Project) error {
	if err := project.ValidateIcon(); err != nil {
		return err
	}
	_, err := d.exec(
		"update project set name=?, alert=?, alert_chat=?, max_parallel_tasks=?, icon=? where id=?",
		project.Name,
		project.Alert,
		project.AlertChat,
		project.MaxParallelTasks,
		project.Icon,
		project.ID)
	return err
}

// Separate from UpdateProject so saving general settings never overwrites templates.
func (d *SqlDb) SetProjectNotificationTemplates(projectID int, settings taskmessage.Settings) error {
	if err := settings.Validate(); err != nil {
		return err
	}
	content, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	_, err = d.exec("update project set notification_templates=? where id=?", string(content), projectID)
	return err
}
