<template>
  <div class="WorkflowNotificationForm">
    <v-select
      :value="value.channel" :items="[{ value: 'telegram', text: 'Telegram' }]"
      :label="$t('workflowNotificationChannel')" :disabled="disabled" outlined dense
      hide-details="auto" class="mb-4" @change="update('channel', $event)"
    />
    <div class="text-caption text--secondary mb-4">
      {{ $t('workflowNotificationChannelHint') }}
    </div>
    <v-select
      :value="value.source_node_id || null" :items="sourceOptions"
      :label="$t('workflowNotificationSource')" :disabled="disabled" outlined dense
      hide-details="auto" class="mb-4" @change="update('source_node_id', $event)"
    />
    <v-textarea
      ref="message" :value="value.message" :label="$t('workflowNotificationMessage')"
      :disabled="disabled" outlined dense auto-grow rows="5" counter="3000"
      :error-messages="problem ? [$t(problem)] : []"
      class="mb-3" @input="update('message', $event)"
    />
    <v-select
      v-model="variable" :items="variables" :label="$t('workflowNotificationInsertVariable')"
      :disabled="disabled" outlined dense hide-details class="mb-4"
      @change="insertVariable"
    />
    <v-combobox
      :value="value.timezone" :items="timezones" :label="$t('workflowNotificationTimezone')"
      :disabled="disabled" outlined dense hide-details="auto" class="mb-4"
      @input="update('timezone', $event || 'UTC')"
    />
    <v-select
      v-if="!approval" :value="value.on_error || 'continue'" :items="failureOptions"
      :label="$t('workflowNotificationOnError')" :disabled="disabled" outlined dense
      hide-details="auto" class="mb-4" @change="update('on_error', $event)"
    />
    <div class="text-caption text--secondary mb-4">
      {{ $t('workflowNotificationErrorHint') }}
    </div>
    <v-card outlined flat class="WorkflowNotificationForm__preview pa-3 mb-4">
      <div class="text-subtitle-2 mb-2">{{ $t('workflowNotificationPreview') }}</div>
      <div class="text-caption text--secondary mb-2">
        {{ $t('workflowNotificationSampleData') }}
      </div>
      <v-progress-linear v-if="loading" indeterminate class="mb-2" />
      <v-alert v-if="error" dense text type="error">{{ error }}</v-alert>
      <div v-else class="WorkflowNotificationForm__text">{{ preview }}</div>
    </v-card>
    <v-btn
      block small outlined color="primary" :disabled="disabled || !!problem || loading || !!error"
      :loading="testing" @click="testNotification"
    >
      <v-icon left small>mdi-send-check-outline</v-icon>
      {{ $t('alertsTest') }}
    </v-btn>
    <v-alert v-if="testResult" dense text :type="testResult.type" class="mt-3">
      {{ testResult.text }}
    </v-alert>
  </div>
</template>

<script>
import axios from 'axios';
import { getErrorMessage } from '@/lib/error';
import { notificationVariables, notificationProblem } from '@/lib/workflowNotifications';

export default {
  props: {
    value: { type: Object, required: true },
    projectId: { type: Number, required: true },
    sources: { type: Array, default: () => [] },
    disabled: Boolean,
    approval: Boolean,
  },
  data() {
    return {
      variables: notificationVariables,
      variable: null,
      preview: '',
      error: '',
      loading: false,
      testing: false,
      testResult: null,
      timer: null,
      sequence: 0,
    };
  },
  computed: {
    problem() { return notificationProblem(this.value); },
    sourceOptions() {
      return [{ value: null, text: this.$t('workflowNotificationAllPredecessors') }, ...this.sources];
    },
    timezones() { return [...new Set(['UTC', 'Europe/Moscow', this.value.timezone].filter(Boolean))]; },
    failureOptions() {
      return [
        { value: 'continue', text: this.$t('workflowNotificationContinue') },
        { value: 'fail', text: this.$t('workflowNotificationFail') },
      ];
    },
  },
  watch: {
    value: { deep: true, immediate: true, handler: 'schedulePreview' },
  },
  beforeDestroy() { clearTimeout(this.timer); this.sequence += 1; },
  methods: {
    update(key, value) { this.$emit('input', { ...this.value, [key]: value }); },
    insertVariable(name) {
      if (!name) return;
      const field = this.$refs.message.$el.querySelector('textarea');
      const start = field.selectionStart;
      const end = field.selectionEnd;
      const token = `{{ ${name} }}`;
      this.update('message', this.value.message.slice(0, start) + token + this.value.message.slice(end));
      this.$nextTick(() => {
        this.variable = null;
        field.focus();
        field.setSelectionRange(start + token.length, start + token.length);
      });
    },
    schedulePreview() {
      clearTimeout(this.timer);
      this.sequence += 1;
      this.preview = '';
      this.error = '';
      this.testResult = null;
      if (this.problem || this.disabled) { this.loading = false; return; }
      this.loading = true;
      const sequence = this.sequence;
      this.timer = setTimeout(() => this.loadPreview(sequence), 300);
    },
    async loadPreview(sequence) {
      try {
        const { data } = await axios.post(`/api/project/${this.projectId}/workflows/notifications/preview`, { notification: this.value });
        if (sequence === this.sequence) this.preview = data.text;
      } catch (error) {
        if (sequence === this.sequence) this.error = getErrorMessage(error) || this.$t('workflowNotificationPreviewFailed');
      } finally {
        if (sequence === this.sequence) this.loading = false;
      }
    },
    async testNotification() {
      if (this.disabled || this.problem || this.testing) return;
      this.testing = true;
      const sequence = this.sequence;
      try {
        await axios.post(`/api/project/${this.projectId}/workflows/notifications/test`, { notification: this.value });
        if (sequence === this.sequence) this.testResult = { type: 'success', text: this.$t('alertsTestSent') };
      } catch (error) {
        if (sequence === this.sequence) this.testResult = { type: 'error', text: getErrorMessage(error) || this.$t('workflowNotificationTestFailed') };
      } finally { this.testing = false; }
    },
  },
};
</script>

<style lang="scss">
.WorkflowNotificationForm {
  min-width: 0;
  .v-input { min-width: 0; }
  .v-select__selection { max-width: 100%; }
  &__preview { background: transparent !important; }
  &__text { white-space: pre-wrap; overflow-wrap: anywhere; font-size: 13px; line-height: 1.5; }
}
</style>
