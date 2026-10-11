<template>
  <div class="telegram-alert-settings">
    <v-progress-linear v-if="loading" indeterminate color="primary" />
    <v-alert v-if="error" type="error" text class="mt-4">{{ error }}</v-alert>
    <v-btn v-if="!settings && !loading" text @click="load">{{ $t('alertsRetry') }}</v-btn>
    <v-form v-if="settings" ref="form" @submit.prevent="save" :disabled="busy">
      <div v-if="projectId" class="px-6 pt-5">
          <v-switch
            v-model="enabled"
            :label="$t('alertsEnableTelegram')"
            :disabled="busy"
            data-testid="alerts-enable-telegram"
            class="mt-0 mb-4"
            hide-details
            inset
          />
        <v-tabs v-model="channelTab" class="telegram-alert-tabs"
          :show-arrows="!$vuetify.breakpoint.xsOnly" data-testid="channel-settings-tabs">
          <v-tab href="#general">{{ $t('alertChannelGeneral') }}</v-tab>
          <v-tab href="#templates">{{ $t('alertMessageTemplates') }}</v-tab>
        </v-tabs>
      </div>
      <div v-show="channelTab === 'general'" class="telegram-alert-fields px-6 pt-5 pb-2">
        <template v-if="projectId">
          <p class="text-body-2 text--secondary">{{ $t('alertsProjectHint') }}</p>
          <v-text-field
            v-model="chatId"
            :label="$t('alertsChatId')"
            :hint="$t('alertsChatHint')"
            persistent-hint
            :disabled="!enabled || busy"
            :rules="enabled ? [(v) => !!v.trim() || $t('alertsChatRequired')] : []"
            outlined
            dense
            class="mb-5"
            data-testid="alerts-chat-id"
          />
          <v-checkbox
            v-model="useGlobalToken"
            :label="$t('alertsUseGlobalToken')"
            :disabled="!enabled || busy"
            hide-details
            class="mt-0 mb-3"
            data-testid="alerts-use-global-token"
          />
          <v-alert
            dense text
            :type="settings.global_token_configured ? 'info' : 'warning'"
            class="text-body-2"
          >
            {{ $t(settings.global_token_configured
              ? 'alertsGlobalTokenAvailable' : 'alertsGlobalTokenMissing') }}
          </v-alert>
        </template>
        <p v-else class="text-body-2 text--secondary">{{ $t('alertsGlobalHint') }}</p>
        <v-text-field
          v-model="token"
          type="password"
          autocomplete="new-password"
          :label="$t(projectId ? 'alertsOverrideToken' : 'alertsBotToken')"
          :placeholder="settings.has_token && !clearToken ? $t('alertsTokenSaved') : ''"
          :hint="$t('alertsTokenHint')"
          persistent-hint
          :disabled="busy || (!!projectId && (!enabled || useGlobalToken))"
          :rules="projectId && enabled && !useGlobalToken
            ? [(v) => !!v.trim() || settings.has_token || $t('alertsTokenRequired')] : []"
          outlined
          dense
          data-testid="alerts-bot-token"
        />
        <v-chip
          v-if="settings.has_token && !clearToken"
          small outlined color="success" class="mt-3"
        >
          <v-icon left small>mdi-check-circle-outline</v-icon>
          {{ $t(projectId ? 'alertsOverrideSaved' : 'alertsGlobalSaved') }}
        </v-chip>
        <v-btn
          v-if="!projectId && settings.has_token && !clearToken"
          small text color="error" class="mt-3 ml-2"
          :disabled="busy"
          @click="clearToken = true; token = ''"
          data-testid="alerts-remove-token"
        >{{ $t('alertsRemoveToken') }}</v-btn>
        <p v-if="clearToken" class="text-body-2 warning--text mt-3 mb-0">
          {{ $t('alertsTokenWillBeRemoved') }}
        </p>
        <v-text-field v-if="!projectId" v-model="testChatId" :label="$t('alertsTestChat')"
          :disabled="busy" outlined dense class="mt-5" data-testid="alerts-test-chat" />
      </div>
      <div v-if="projectId && templates" v-show="channelTab === 'templates'" class="px-6 pt-5 pb-2">
        <ProjectAlertTemplates v-model="templates" :project-id="projectId"
          :status.sync="templateStatus" :defaults="settings.template_defaults || {}"
          :variables="settings.template_variables || []" :disabled="!enabled || busy" />
      </div>
      <div class="px-6" v-if="testResult">
        <v-alert :type="testResult.ok ? 'success' : 'error'" text class="mb-0"
          data-testid="alerts-test-result">{{ testResult.text }}</v-alert>
      </div>
      <v-divider class="mt-4" />
      <v-card-actions class="px-6 py-4 telegram-alert-actions">
        <v-btn outlined color="primary" type="button" @click="testNotification"
          :loading="testing" :disabled="busy || (!!projectId && !enabled) || templatesInvalid"
          class="telegram-alert-test"
          data-testid="alerts-test">
          <v-icon left small>mdi-send-check-outline</v-icon>{{ $t('alertsTest') }}
        </v-btn>
        <div class="telegram-alert-save-actions">
          <v-btn v-if="projectId" text color="primary" :disabled="busy || !dirty" @click="reset"
            data-testid="alerts-cancel">
            {{ $t('cancel') }}
          </v-btn>
          <v-btn
            color="primary"
            :loading="saving"
            :disabled="!dirty || busy || (enabled && templatesInvalid)"
            type="submit"
            data-testid="alerts-save"
          >{{ $t('save') }}</v-btn>
        </div>
      </v-card-actions>
    </v-form>
  </div>
</template>

<script>
import axios from 'axios';
import EventBus from '@/event-bus';
import ProjectAlertTemplates from '@/components/ProjectAlertTemplates.vue';
import defaultMessages from '@/lib/taskNotifications';
import { notificationProblem } from '@/lib/workflowNotifications';
import { getErrorMessage } from '@/lib/error';

export default {
  components: { ProjectAlertTemplates },
  props: { projectId: { type: Number, default: 0 } },
  data() {
    return {
      loading: false,
      saving: false,
      testing: false,
      testChatId: '',
      testResult: null,
      settings: null,
      enabled: false,
      chatId: '',
      token: '',
      useGlobalToken: true,
      clearToken: false,
      error: '',
      templates: null,
      originalTemplates: null,
      templateStatus: 'success',
      channelTab: 'general',
      generation: 0,
    };
  },
  computed: {
    busy() { return this.loading || this.saving || this.testing; },
    testInput() {
      return JSON.stringify([this.chatId, this.testChatId, this.token, this.useGlobalToken,
        this.enabled, this.clearToken, this.templates, this.templateStatus]);
    },
    url() {
      return this.projectId
        ? `/api/project/${this.projectId}/alerts/telegram` : '/api/alerts/telegram';
    },
    templatesInvalid() {
      if (!this.projectId || !this.templates) return false;
      return ['success', 'error'].some((status) => notificationProblem({ ...this.templates, message: this.templates[status] }, this.settings.template_variables || []));
    },
    dirty() {
      return this.settings && (this.token.trim() !== '' || this.clearToken
        || (this.projectId && (this.enabled !== this.settings.enabled
          || this.chatId !== this.settings.chat_id
          || this.useGlobalToken === this.settings.has_token
          || (this.enabled && this.templates && !this.settings.templates)
          || JSON.stringify(this.templates) !== JSON.stringify(this.originalTemplates))));
    },
  },
  watch: {
    testInput() { this.testResult = null; },
    projectId() { this.load(); },
  },
  created() { this.load(); },
  methods: {
    reset() {
      this.apply(this.settings);
      this.error = '';
      this.testResult = null;
      this.$refs.form.resetValidation();
    },
    apply(settings) {
      this.settings = settings;
      this.enabled = settings.enabled;
      this.chatId = settings.chat_id;
      this.useGlobalToken = !settings.has_token;
      this.token = '';
      this.clearToken = false;
      const defaults = defaultMessages(settings.template_defaults || {}, this.$i18n?.locale);
      this.templates = defaults ? {
        ...defaults,
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC',
      } : null;
      if (settings.templates) this.templates = { ...settings.templates };
      this.originalTemplates = this.templates ? { ...this.templates } : null;
    },
    async load() {
      this.generation += 1;
      const generation = this.generation;
      this.testResult = null;
      this.channelTab = 'general';
      this.loading = true;
      this.error = '';
      this.settings = null;
      this.token = '';
      try {
        const { data } = await axios.get(this.url);
        if (generation === this.generation) this.apply(data);
      } catch (err) {
        if (generation === this.generation) this.error = getErrorMessage(err);
      } finally {
        if (generation === this.generation) this.loading = false;
      }
    },
    requestData() {
      const data = { enabled: this.enabled, chat_id: this.chatId };
      if (this.projectId && this.useGlobalToken) data.token = '';
      else if (this.token.trim()) data.token = this.token.trim();
      else if (this.clearToken) data.token = '';
      if (this.projectId && this.templates && !this.templatesInvalid) {
        data.templates = this.templates;
      }
      return data;
    },
    async testNotification() {
      if (this.busy || this.templatesInvalid || (this.projectId && !this.enabled)) return;
      if (!this.$refs.form.validate()) { this.channelTab = 'general'; return; }
      const data = this.requestData();
      data.chat_id = (this.projectId ? this.chatId : this.testChatId).trim();
      if (!data.chat_id) {
        this.testResult = { ok: false, text: this.$t('alertsChatRequired') };
        return;
      }
      data.locale = this.$i18n?.locale || 'en';
      if (this.projectId) data.status = this.templateStatus;
      const url = this.url;
      this.testing = true;
      this.testResult = null;
      try {
        await axios.post(`${url}/test`, data);
        if (url === this.url) this.testResult = { ok: true, text: this.$t('alertsTestSent') };
      } catch (err) {
        if (url === this.url) this.testResult = { ok: false, text: getErrorMessage(err) };
      } finally {
        this.testing = false;
      }
    },
    async save() {
      if (this.busy || !this.dirty || (this.enabled && this.templatesInvalid)) return;
      if (!this.$refs.form.validate()) { this.channelTab = 'general'; return; }
      const generation = this.generation;
      this.error = '';
      this.saving = true;
      const data = this.requestData();
      try {
        const response = await axios.put(this.url, data);
        if (generation !== this.generation) return;
        this.apply(response.data);
        EventBus.$emit('i-snackbar', { color: 'success', text: this.$t('alertsSaved') });
        this.$emit('saved');
      } catch (err) {
        if (generation === this.generation) this.error = getErrorMessage(err);
      } finally {
        this.saving = false;
      }
    },
  },
};
</script>

<style lang="scss">
@media (max-width: 600px) {
  .telegram-alert-tabs {
    .v-tabs-bar { height: auto !important; }
    .v-tabs-bar__content { width: 100%; gap: 8px; }
    .v-tab {
      flex: 1 1 0;
      height: auto !important;
      min-height: 56px !important;
      min-width: 0 !important;
      max-width: none !important;
      margin: 0 !important;
      padding: 8px !important;
      white-space: normal;
      text-align: center;
      line-height: 1.4;
    }
  }
}

.telegram-alert-actions {
  flex-wrap: wrap;
  gap: 12px;
  .v-btn { margin: 0 !important; }
  .telegram-alert-test {
    max-width: 100%;
    height: auto !important;
    min-height: 38px;
    padding-top: 8px;
    padding-bottom: 8px;
    .v-btn__content { white-space: normal; }
  }
}
.telegram-alert-save-actions {
  display: flex;
  flex-shrink: 0;
  gap: 12px;
  margin-left: auto;
}
</style>
