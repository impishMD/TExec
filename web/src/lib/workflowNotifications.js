export const notificationVariables = [
  'project.name', 'workflow.name', 'run.id', 'run.started_by', 'run.url',
  'date', 'time', 'timezone', 'source.name', 'source.status', 'source.error', 'approval.deadline',
];

export function defaultNotification(locale = 'en', approval = false) {
  const ru = locale.startsWith('ru');
  const headings = ru
    ? ['Оповещение рабочего процесса', 'Требуется подтверждение']
    : ['Workflow notification', 'Approval required'];
  const heading = headings[approval ? 1 : 0];
  return {
    channel: 'telegram',
    message: `${heading}\n{{ project.name }} · {{ workflow.name }} #{{ run.id }}\n{{ date }} {{ time }} ({{ timezone }})\n{{ source.name }}: {{ source.status }}${approval ? '\n{{ approval.deadline }}' : ''}\n{{ run.url }}`,
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC',
    locale,
    source_node_id: null,
    on_error: 'continue',
  };
}

export function notificationProblem(config, variables = notificationVariables) {
  if (!config || !config.message?.trim() || [...config.message].length > 3000) return 'workflowNotificationMessageInvalid';
  const matches = [...config.message.matchAll(/\{\{\s*([a-z_]+(?:\.[a-z_]+)?)\s*\}\}/g)];
  if (matches.some((m) => !variables.includes(m[1]))) return 'workflowNotificationVariableInvalid';
  const rest = config.message.replace(/\{\{\s*([a-z_]+(?:\.[a-z_]+)?)\s*\}\}/g, '');
  if (rest.includes('{{') || rest.includes('}}')) return 'workflowNotificationVariableInvalid';
  try {
    Intl.DateTimeFormat('en', { timeZone: config.timezone || 'UTC' });
  } catch {
    return 'workflowNotificationTimezoneInvalid';
  }
  return null;
}

export function notificationSources(nodes, edges, nodeId) {
  const seen = new Set();
  function visit(id) {
    edges.filter((edge) => edge.destination_node_id === id).forEach((edge) => {
      if (seen.has(edge.source_node_id)) return;
      seen.add(edge.source_node_id);
      visit(edge.source_node_id);
    });
  }
  visit(nodeId);
  return nodes.filter((node) => seen.has(node.id) && node.kind !== 'note');
}

export function notificationRunStatus(delivery, config) {
  if (delivery.status === 'sent') return 'success';
  if (delivery.status === 'failed' && config?.on_error !== 'fail') return 'success';
  if (delivery.status === 'failed' || delivery.status === 'cancelled') return 'error';
  return 'running';
}
