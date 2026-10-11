// API dates are UTC calendar days. Keep each day unique and ordered so Chart.js
// does not overwrite a day's stack with a second, empty point for that date.
export default function taskStats(rows, startDate, endDate) {
  const byDate = new Map(rows.map((row) => [row.date, row.count_by_status]));
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
