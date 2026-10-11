import { expect } from 'chai';
import taskStats from '@/lib/taskStats';

describe('Task statistics daily series', () => {
  it('keeps descending API results once per day without erasing today’s counts', () => {
    const rows = [
      { date: '2026-10-11', count_by_status: { success: 3, error: 2, stopped: 1 } },
      { date: '2026-10-10', count_by_status: { success: 1 } },
    ];
    const original = JSON.stringify(rows);
    const result = taskStats(rows, '2026-10-04', '2026-10-11');
    expect(result).to.have.length(8);
    expect(new Set(result.map((row) => row.date)).size).to.equal(8);
    expect(result[7]).to.deep.equal(rows[0]);
    expect(result[6].count_by_status).to.deep.equal({ success: 1, error: 0, stopped: 0 });
    expect(JSON.stringify(rows)).to.equal(original);
  });

  it('fills empty days and all status counts for an empty response', () => {
    expect(taskStats([], '2026-10-10', '2026-10-11')).to.deep.equal([
      { date: '2026-10-10', count_by_status: { success: 0, error: 0, stopped: 0 } },
      { date: '2026-10-11', count_by_status: { success: 0, error: 0, stopped: 0 } },
    ]);
  });

  it('fills gaps independently of response order and ignores dates outside the range', () => {
    const result = taskStats([
      { date: '2026-10-09', count_by_status: { error: 99 } },
      { date: '2026-10-10', count_by_status: { error: 2 } },
      { date: '2026-10-12', count_by_status: { stopped: 1 } },
    ], '2026-10-10', '2026-10-12');
    expect(result.map((row) => row.count_by_status)).to.deep.equal([
      { success: 0, error: 2, stopped: 0 },
      { success: 0, error: 0, stopped: 0 },
      { success: 0, error: 0, stopped: 1 },
    ]);
  });

  it('keeps calendar days consecutive across daylight saving and leap days', () => {
    expect(taskStats([], '2026-03-28', '2026-03-30').map((row) => row.date))
      .to.deep.equal(['2026-03-28', '2026-03-29', '2026-03-30']);
    expect(taskStats([], '2024-02-28', '2024-03-01').map((row) => row.date))
      .to.deep.equal(['2024-02-28', '2024-02-29', '2024-03-01']);
  });
});
