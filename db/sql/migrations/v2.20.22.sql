alter table project__alert_channel add column message_templates text null;

-- Preserve templates from the first local preview, scoped to Telegram only.
insert into project__alert_channel (project_id, channel, enabled, settings, secret)
select id, 'telegram', false, '{}', '' from project
where notification_templates is not null
and not exists (select 1 from project__alert_channel c where c.project_id=project.id and c.channel='telegram');

update project__alert_channel set message_templates=(
  select notification_templates from project where id=project__alert_channel.project_id
) where channel='telegram';

alter table project drop column notification_templates;
