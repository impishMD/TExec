alter table project add column notification_templates text null;
update project set notification_templates=(
  select message_templates from project__alert_channel where project_id=project.id and channel='telegram'
);
alter table project__alert_channel drop column message_templates;
