alter table project__workflow_node add notification text null;
create table project__workflow_notification (
    id integer primary key autoincrement,
    project_id int not null,
    workflow_run_id int not null,
    workflow_node_id int not null,
    status varchar(16) not null,
    attempts int not null default 0,
    message text not null,
    last_error text not null,
    created datetime not null,
    next_attempt_at datetime not null,
    resolved datetime null,
    foreign key (project_id) references project(id) on delete cascade,
    foreign key (workflow_run_id) references project__workflow_run(id) on delete cascade,
    foreign key (workflow_node_id) references project__workflow_node(id) on delete cascade,
    unique (workflow_run_id, workflow_node_id)
);
create index project__workflow_notification_due on project__workflow_notification(status, next_attempt_at);
