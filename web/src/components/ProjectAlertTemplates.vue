<template>
  <div class="ProjectAlertTemplates">
    <v-progress-linear v-if="loading" indeterminate />
    <div v-if="error" class="pa-4">
      <v-alert type="error" text dense>{{ error }}</v-alert>
      <v-btn v-if="!settings" text @click="load">{{ $t('refresh') }}</v-btn>
    </div>
    <template v-if="settings">
      <div class="pa-4 pa-sm-6">
        <p class="text-body-2 text--secondary">{{ $t('alertMessageTemplatesHint') }}</p>
        <v-alert v-if="!configured" type="info" text dense>
          {{ $t('alertMessageTemplatesUnsaved') }}
        </v-alert>
        <v-row>
          <v-col cols="12" sm="6" lg="4">
            <v-select v-model="settings.locale" :items="locales"
              :label="$t('alertMessageLanguage')" outlined dense hide-details :disabled="busy" />
          </v-col>
          <v-col cols="12" sm="6" lg="4">
            <v-combobox v-model="settings.timezone" :items="timezones"
              :label="$t('workflowNotificationTimezone')" outlined dense hide-details="auto"
              :disabled="busy" />
          </v-col>
        </v-row>
        <v-tabs v-model="status" class="mt-4" show-arrows data-testid="message-status-tabs">
          <v-tab href="#success">{{ $t('alertMessageSuccess') }}</v-tab>
          <v-tab href="#error">{{ $t('alertMessageError') }}</v-tab>
        </v-tabs>
        <v-row class="mt-2">
          <v-col cols="12" md="6">
            <v-textarea ref="message" v-model="settings[status]" outlined auto-grow rows="9"
              :label="$t('workflowNotificationMessage')" counter="3000" :disabled="busy"
              :error-messages="problem ? [$t(problem)] : []" data-testid="message-template" />
            <v-select v-model="variable" :items="variables" outlined dense
              :label="$t('workflowNotificationInsertVariable')" :disabled="busy" hide-details
              @change="insertVariable" data-testid="message-variable" />
            <v-btn text small color="primary" class="mt-3" :disabled="busy"
              @click="resetTemplate" data-testid="message-reset">
              <v-icon left small>mdi-restore</v-icon>{{ $t('alertMessageReset') }}
            </v-btn>
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
        <v-alert v-if="otherProblem" type="warning" text dense>
          {{ $t('alertMessageOtherInvalid') }}
        </v-alert>
        <v-alert v-if="result" :type="result.type" text dense>{{ result.text }}</v-alert>
      </div>
      <v-divider />
      <div class="ProjectAlertTemplates__actions pa-4 pa-sm-6">
        <v-btn outlined color="primary" :disabled="busy || invalid || previewLoading"
          :loading="testing" @click="testNotification" data-testid="message-test">
          <v-icon left>mdi-send-check-outline</v-icon>{{ $t('alertsTest') }}
        </v-btn>
        <div class="ProjectAlertTemplates__save">
          <v-btn text :disabled="busy || !dirty" @click="cancel">{{ $t('cancel') }}</v-btn>
          <v-btn color="primary" :disabled="busy || invalid || !dirty" :loading="saving"
            @click="save" data-testid="message-save">{{ $t('save') }}</v-btn>
        </div>
      </div>
    </template>
  </div>
</template>

<script>
import axios from 'axios';
import { getErrorMessage } from '@/lib/error';
import { notificationProblem } from '@/lib/workflowNotifications';

export default {
  props: { projectId: { type: Number, required: true } },
  data() {
    return {
      settings: null,
      original: null,
      defaults: {},
      variables: [],
      configured: false,
      status: 'success',
      variable: null,
      loading: false,
      saving: false,
      testing: false,
      error: '',
      result: null,
      preview: '',
      previewError: '',
      previewLoading: false,
      timer: null,
      sequence: 0,
      generation: 0,
      locales: [{ value: 'ru', text: 'Русский' }, { value: 'en', text: 'English' }],
    };
  },
  computed: {
    url() { return `/api/project/${this.projectId}/alerts/templates`; },
    busy() { return this.loading || this.saving || this.testing; },
    dirty() {
      return !this.configured || JSON.stringify(this.settings) !== JSON.stringify(this.original);
    },
    problem() { return this.problemFor(this.status); },
    otherProblem() { return this.problemFor(this.status === 'success' ? 'error' : 'success'); },
    invalid() { return !!(this.problem || this.otherProblem); },
    timezones() { return [...new Set(['UTC', 'Europe/Moscow', this.settings?.timezone].filter(Boolean))]; },
  },
  watch: {
    projectId: { immediate: true, handler: 'load' },
    settings: { deep: true, handler: 'schedulePreview' },
    status: 'schedulePreview',
  },
  beforeDestroy() { clearTimeout(this.timer); this.sequence += 1; this.generation += 1; },
  methods: {
    problemFor(status) {
      const config = { ...this.settings, message: this.settings?.[status] };
      return notificationProblem(config, this.variables);
    },
    async load() {
      this.generation += 1;
      const generation = this.generation;
      this.loading = true;
      this.settings = null;
      this.error = '';
      this.result = null;
      try {
        const locale = this.$i18n?.locale?.startsWith('ru') ? 'ru' : 'en';
        const { data } = await axios.get(this.url, { params: { locale } });
        if (generation !== this.generation) return;
        this.defaults = data.defaults;
        this.variables = data.variables;
        this.configured = data.configured;
        this.settings = { ...data.settings };
        if (!data.configured) {
          this.settings.timezone = Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC';
        }
        this.original = { ...this.settings };
      } catch (error) {
        if (generation === this.generation) this.error = getErrorMessage(error);
      } finally { if (generation === this.generation) this.loading = false; }
    },
    cancel() { this.settings = { ...this.original }; this.error = ''; },
    resetTemplate() {
      this.$set(this.settings, this.status, this.defaults[this.settings.locale][this.status]);
    },
    insertVariable(name) {
      if (!name) return;
      const field = this.$refs.message.$el.querySelector('textarea');
      const start = field.selectionStart;
      const end = field.selectionEnd;
      const text = this.settings[this.status];
      const token = `{{ ${name} }}`;
      this.$set(this.settings, this.status, text.slice(0, start) + token + text.slice(end));
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
      this.result = null;
      this.previewLoading = false;
      if (!this.settings || this.invalid) return;
      this.previewLoading = true;
      this.timer = setTimeout(() => this.loadPreview(sequence), 300);
    },
    async loadPreview(sequence) {
      try {
        const { data } = await axios.post(`${this.url}/preview`, { settings: this.settings, status: this.status });
        if (sequence === this.sequence) this.preview = data.text;
      } catch (error) {
        if (sequence === this.sequence) this.previewError = getErrorMessage(error);
      } finally { if (sequence === this.sequence) this.previewLoading = false; }
    },
    async save() {
      if (this.busy || this.invalid || !this.dirty) return;
      const generation = this.generation;
      this.saving = true;
      this.error = '';
      this.result = null;
      try {
        const { data } = await axios.put(this.url, { settings: this.settings });
        if (generation !== this.generation) return;
        this.original = { ...data };
        this.configured = true;
        this.result = { type: 'success', text: this.$t('alertsSaved') };
      } catch (error) {
        if (generation === this.generation) this.error = getErrorMessage(error);
      } finally { this.saving = false; }
    },
    async testNotification() {
      if (this.busy || this.invalid) return;
      const sequence = this.sequence;
      this.testing = true;
      this.result = null;
      try {
        await axios.post(`${this.url}/test`, { settings: this.settings, status: this.status });
        if (sequence === this.sequence) this.result = { type: 'success', text: this.$t('alertsTestSent') };
      } catch (error) {
        if (sequence === this.sequence) this.result = { type: 'error', text: getErrorMessage(error) };
      } finally { this.testing = false; }
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
  &__actions { display: flex; flex-wrap: wrap; gap: 16px; }
  &__actions > .v-btn {
    max-width: 100%; height: auto !important; min-height: 40px;
    padding-top: 8px; padding-bottom: 8px;
  }
  &__actions .v-btn__content { white-space: normal; }
  &__save { display: flex; gap: 12px; margin-left: auto; }
}
</style>
