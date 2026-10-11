<template>
  <div class="ProjectAlertTemplates">
    <p class="text-body-2 text--secondary">{{ $t('alertMessageTemplatesHint') }}</p>
    <v-row>
      <v-col cols="12" sm="6" lg="4">
        <v-combobox :value="value.timezone" :items="timezones" @input="patch({ timezone: $event })"
          :label="$t('workflowNotificationTimezone')" outlined dense hide-details="auto"
          :disabled="disabled" data-testid="message-timezone" />
      </v-col>
    </v-row>
    <v-tabs :value="status" @change="$emit('update:status', $event)" class="mt-4" show-arrows
      data-testid="message-status-tabs">
      <v-tab href="#success">{{ $t('alertMessageSuccess') }}</v-tab>
      <v-tab href="#error">{{ $t('alertMessageError') }}</v-tab>
    </v-tabs>
    <v-row class="mt-2">
      <v-col cols="12" md="6">
        <v-textarea ref="message" :value="value[status]" @input="patch({ [status]: $event })"
          outlined auto-grow rows="9" :label="$t('workflowNotificationMessage')" counter="3000"
          :disabled="disabled" :error-messages="!disabled && problem ? [$t(problem)] : []"
          data-testid="message-template" />
        <v-select v-model="variable" :items="variables" outlined dense
          :label="$t('workflowNotificationInsertVariable')" :disabled="disabled" hide-details
          @change="insertVariable" data-testid="message-variable" />
        <v-btn text small color="primary" class="mt-3" :disabled="disabled"
          @click="resetTemplate" data-testid="message-reset">
          <v-icon left small>mdi-restore</v-icon>{{ $t('alertMessageReset') }}
        </v-btn>
        <p class="text-caption text--secondary mt-2">{{ $t('alertMessageResetHint') }}</p>
      </v-col>
      <v-col cols="12" md="6">
        <v-card outlined flat class="ProjectAlertTemplates__preview pa-4">
          <div class="text-subtitle-2 mb-2">{{ $t('workflowNotificationPreview') }}</div>
          <p class="text-caption text--secondary">{{ $t('workflowNotificationSampleData') }}</p>
          <v-progress-linear v-if="previewLoading" indeterminate class="mb-3" />
          <v-alert v-if="previewError" type="error" text dense>{{ previewError }}</v-alert>
          <div v-else class="ProjectAlertTemplates__text" data-testid="message-preview">
            {{ preview }}
          </div>
        </v-card>
        <p class="text-caption text--secondary mt-3">{{ $t('alertMessageVariablesHint') }}</p>
        <p class="text-caption text--secondary">{{ $t('workflowNotificationErrorHint') }}</p>
      </v-col>
    </v-row>
    <v-alert v-if="!disabled && otherProblem" type="warning" text dense>
      {{ $t('alertMessageOtherInvalid') }}
    </v-alert>
  </div>
</template>

<script>
import axios from 'axios';
import { getErrorMessage } from '@/lib/error';
import { notificationProblem } from '@/lib/workflowNotifications';
import defaultMessages from '@/lib/taskNotifications';

export default {
  props: {
    projectId: { type: Number, required: true },
    value: { type: Object, required: true },
    defaults: { type: Object, required: true },
    variables: { type: Array, required: true },
    status: { type: String, default: 'success' },
    disabled: { type: Boolean, default: false },
  },
  data() {
    return {
      variable: null,
      preview: '',
      previewError: '',
      previewLoading: false,
      timer: null,
      sequence: 0,
    };
  },
  computed: {
    problem() { return this.problemFor(this.status); },
    otherProblem() { return this.problemFor(this.status === 'success' ? 'error' : 'success'); },
    timezones() { return [...new Set(['UTC', 'Europe/Moscow', this.value.timezone].filter(Boolean))]; },
  },
  watch: {
    value: { deep: true, immediate: true, handler: 'schedulePreview' },
    status: 'schedulePreview',
    projectId: 'schedulePreview',
    disabled: 'schedulePreview',
    '$i18n.locale': 'schedulePreview',
  },
  beforeDestroy() { clearTimeout(this.timer); this.sequence += 1; },
  methods: {
    patch(change) { if (!this.disabled) this.$emit('input', { ...this.value, ...change }); },
    problemFor(status) {
      return notificationProblem({ ...this.value, message: this.value[status] }, this.variables);
    },
    resetTemplate() {
      const defaults = defaultMessages(this.defaults, this.$i18n?.locale);
      if (!defaults) return;
      this.patch({
        [this.status]: defaults[this.status],
        [`${this.status}_locale`]: defaults[`${this.status}_locale`],
      });
    },
    insertVariable(name) {
      if (!name || this.disabled) return;
      const field = this.$refs.message.$el.querySelector('textarea');
      const start = field.selectionStart;
      const end = field.selectionEnd;
      const token = `{{ ${name} }}`;
      this.patch({
        [this.status]: this.value[this.status].slice(0, start)
        + token + this.value[this.status].slice(end),
      });
      this.$nextTick(() => {
        this.variable = null;
        field.focus();
        field.setSelectionRange(start + token.length, start + token.length);
      });
    },
    schedulePreview() {
      clearTimeout(this.timer);
      this.sequence += 1;
      const sequence = this.sequence;
      this.preview = '';
      this.previewError = '';
      this.previewLoading = false;
      if (this.disabled || this.problem || this.otherProblem) return;
      this.previewLoading = true;
      this.timer = setTimeout(() => this.loadPreview(sequence), 300);
    },
    async loadPreview(sequence) {
      try {
        const { data } = await axios.post(
          `/api/project/${this.projectId}/alerts/telegram/templates/preview`,
          { templates: this.value, status: this.status, locale: this.$i18n?.locale || 'en' },
        );
        if (sequence === this.sequence) this.preview = data.text;
      } catch (error) {
        if (sequence === this.sequence) this.previewError = getErrorMessage(error);
      } finally { if (sequence === this.sequence) this.previewLoading = false; }
    },
  },
};
</script>

<style lang="scss">
.ProjectAlertTemplates {
  min-width: 0;
  .v-input { min-width: 0; }
  &__preview { background: transparent !important; min-height: 220px; }
  &__text { white-space: pre-wrap; overflow-wrap: anywhere; font-size: 14px; line-height: 1.6; }
}
</style>
