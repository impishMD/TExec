// SQL DATE values reach the API as YYYY-MM-DD (SQLite) or RFC3339 timestamps
// (PostgreSQL/MySQL). Keep the calendar day already grouped by the database:
// converting midnight to another timezone could move its counts to another day.
export default function taskStats(rows, startDate, endDate) {
  const byDate = new Map(rows.map((row) => [row.date.slice(0, 10), row.count_by_status]));
  const result = [];
  const date = new Date(`${startDate}T00:00:00Z`);
  const end = new Date(`${endDate}T00:00:00Z`);
  while (date <= end) {
    const key = date.toISOString().slice(0, 10);
    const counts = byDate.get(key) || {};
    result.push({
      date: key,
      count_by_status: {
        success: counts.success || 0,
        error: counts.error || 0,
        stopped: counts.stopped || 0,
      },
    });
    date.setUTCDate(date.getUTCDate() + 1);
  }
  return result;
}
