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
      return this.$vuetify.theme.dark
        ? { text: '#a0b5be', grid: '#2b404a', stopped: '#a0b5be' }
        : { text: '#62777e', grid: '#dfe8e7', stopped: '#62777e' };
    },

    chartData() {
      return {

        labels: (this.sourceData || []).map((row) => row.date),

        datasets: [
          {
            status: 'success',
            label: this.$t('status_success'),
            borderColor: '#4caf50',
            backgroundColor: '#4caf50',
            data: (this.sourceData || []).map((row) => row.count_by_status.success),
          },
          {
            status: 'error',
            label: this.$t('status_failed'),
            borderColor: '#ff5252',
            backgroundColor: '#ff5252',
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
