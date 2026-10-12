<template>
  <BarChartGenerator
    :chart-id="chartId"
    dataset-id-key="status"
    :chart-options="chartOptions"
    :chart-data="chartData"
  />
</template>

<script>

import {
  BarElement,
  CategoryScale,
  Chart as ChartJS,
  Legend,
  LinearScale,
  LineElement,
  PointElement,
  TimeScale,
  Title,
  Tooltip,
} from 'chart.js';

import './chartjs-adapter-day';

ChartJS.register(
  Title,
  Tooltip,
  Legend,
  BarElement,
  LineElement,
  LinearScale,
  CategoryScale,
  PointElement,
  TimeScale,
);

export default {

  name: 'LineChart',

  props: {

    sourceData: Array,

    chartId: String,
  },

  computed: {
    chartColors() {
      const current = this.$vuetify.theme.themes[this.$vuetify.theme.dark ? 'dark' : 'light'];
      return {
        text: current.muted || '#62777e',
        grid: current.border || '#dfe8e7',
        stopped: current.muted || '#62777e',
        success: current.success,
        error: current.error,
      };
    },

    chartData() {
      return {

        labels: (this.sourceData || []).map((row) => row.date),

        datasets: [
          {
            status: 'success',
            label: this.$t('status_success'),
            borderColor: this.chartColors.success,
            backgroundColor: this.chartColors.success,
            data: (this.sourceData || []).map((row) => row.count_by_status.success),
          },
          {
            status: 'error',
            label: this.$t('status_failed'),
            borderColor: this.chartColors.error,
            backgroundColor: this.chartColors.error,
            data: (this.sourceData || []).map((row) => row.count_by_status.error),
          },
          {
            status: 'stopped',
            label: this.$t('status_stopped'),
            borderColor: this.chartColors.stopped,
            backgroundColor: this.chartColors.stopped,
            data: (this.sourceData || []).map((row) => row.count_by_status.stopped),
          },
        ],
      };
    },
    chartOptions() {
      return {
        scales: {
          x: {
            stacked: true,
            type: 'time',
            time: {
              unit: 'day',
              tooltipFormat: 'D MMM YYYY',
            },
            ticks: { color: this.chartColors.text },
            grid: { color: this.chartColors.grid },
          },
          y: {
            stacked: true,
            beginAtZero: true,
            grid: { color: this.chartColors.grid },
            ticks: {
              color: this.chartColors.text,
              maxTicksLimit: 20,
              stepSize: 1,
            },
          },
        },
        plugins: {
          legend: { labels: { color: this.chartColors.text } },
        },
        interaction: { mode: 'index', intersect: false },
        responsive: true,
        maintainAspectRatio: false,
        animation: {
          duration: 0,
        },
      };
    },
  },
};

</script>
