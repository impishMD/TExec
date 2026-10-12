<template>
  <div>
    <v-toolbar flat>
      <v-app-bar-nav-icon @click="showDrawer" />
      <v-toolbar-title>{{ $t('generalSettings') }}</v-toolbar-title>
    </v-toolbar>
    <v-divider />
    <div class="server-settings-page mx-auto mt-8">
      <p class="text-body-1 text--secondary mb-6">{{ $t('serverSettingsHint') }}</p>
      <v-progress-linear v-if="loading" indeterminate color="primary" />
      <v-alert v-if="error" type="error" text data-testid="server-settings-error">
        {{ error }}
      </v-alert>
      <v-btn v-if="!settings && !loading" text @click="load" data-testid="server-settings-retry">
        {{ $t('alertsRetry') }}
      </v-btn>
      <v-card v-if="settings" outlined class="pa-6">
        <v-form @submit.prevent="save">
          <h2 class="text-h6 mb-2">{{ $t('appearance') }}</h2>
          <p class="text-body-2 text--secondary">{{ $t('appearanceGlobalHint') }}</p>
          <ThemePicker v-model="appearanceTheme" :disabled="saving" />
          <v-divider class="my-6" />
          <v-combobox
            v-model="scheduleTimezone"
            :items="timezones"
            :label="$t('scheduleTimezone')"
            :placeholder="$t('scheduleTimezoneSystem', { zone: settings.schedule_timezone.system })"
            :disabled="saving || timezoneLocked"
            aria-describedby="schedule-timezone-hint"
            hide-details
            clearable
            outlined
            dense
            class="mb-4"
            data-testid="server-schedule-timezone"
          />
          <p id="schedule-timezone-hint" class="text-body-2 text--secondary">
            {{ $t('scheduleTimezoneSettingsHint') }}
          </p>
          <p class="text-body-2 text--secondary" data-testid="schedule-timezone-effective">
            {{ $t('scheduleTimezoneEffective', { zone: settings.schedule_timezone.value }) }}
          </p>
          <v-alert v-if="timezoneLocked" type="info" text data-testid="schedule-timezone-locked">
            {{ $t(settings.schedule_timezone.source === 'environment'
              ? 'scheduleTimezoneEnvironmentOverride' : 'scheduleTimezoneConfigurationOverride', {
                zone: settings.schedule_timezone.value,
              }) }}
          </v-alert>
          <v-divider class="my-6" />
          <v-switch
            v-model="useRemoteRunner"
            :label="$t('remoteRunnersDefault')"
            :disabled="saving"
            aria-describedby="remote-runners-hint"
            data-testid="server-remote-runners"
            class="mt-0 mb-4"
            hide-details
            inset
          />
          <p id="remote-runners-hint" class="text-body-2 text--secondary">
            {{ $t('remoteRunnersDefaultHint') }}
          </p>
          <v-btn text small to="/runners" class="mb-4">
            <v-icon left small>mdi-cogs</v-icon>{{ $t('runners') }}
          </v-btn>
          <div class="d-flex justify-end">
            <v-btn
              type="submit" color="primary" :disabled="!dirty || saving" :loading="saving"
              data-testid="server-settings-save"
            >{{ $t('save') }}</v-btn>
          </div>
        </v-form>
      </v-card>
    </div>
  </div>
</template>

<script>
import axios from 'axios';
import ThemePicker from '@/components/ThemePicker.vue';
import EventBus from '@/event-bus';
import { getErrorMessage } from '@/lib/error';

export default {
  components: { ThemePicker },
  data: () => ({
    settings: null,
    useRemoteRunner: false,
    scheduleTimezone: '',
    appearanceTheme: 'texec',
    loading: true,
    saving: false,
    error: '',
  }),
  computed: {
    dirty() {
      return this.settings && (this.appearanceTheme !== (this.settings.appearance_theme || 'texec')
        || this.useRemoteRunner !== this.settings.use_remote_runner
        || (!this.timezoneLocked
          && (this.scheduleTimezone || '') !== this.settings.schedule_timezone.saved));
    },
    timezoneLocked() {
      return ['environment', 'configuration'].includes(this.settings?.schedule_timezone.source);
    },
    timezones() {
      return [...new Set([
        this.settings?.schedule_timezone.system,
        this.settings?.schedule_timezone.saved,
        'UTC', 'Europe/Moscow',
        ...(Intl.supportedValuesOf ? Intl.supportedValuesOf('timeZone') : []),
      ].filter(Boolean))];
    },
  },
  created() { this.load(); },
  methods: {
    showDrawer() { EventBus.$emit('i-show-drawer'); },
    apply(settings) {
      this.settings = settings;
      this.useRemoteRunner = settings.use_remote_runner;
      this.appearanceTheme = settings.appearance_theme || 'texec';
      this.scheduleTimezone = this.timezoneLocked
        ? settings.schedule_timezone.value : settings.schedule_timezone.saved;
      EventBus.$emit('i-server-settings', settings);
    },
    async load() {
      this.loading = true;
      this.error = '';
      this.settings = null;
      try {
        this.apply((await axios.get('/api/settings')).data);
      } catch (err) {
        this.error = getErrorMessage(err);
      } finally {
        this.loading = false;
      }
    },
    async save() {
      if (!this.dirty || this.saving) return;
      this.error = '';
      this.saving = true;
      try {
        const data = {
          use_remote_runner: this.useRemoteRunner,
          appearance_theme: this.appearanceTheme,
        };
        if (!this.timezoneLocked) data.schedule_timezone = this.scheduleTimezone || '';
        this.apply((await axios.put('/api/settings', data)).data);
        EventBus.$emit('i-snackbar', { color: 'success', text: this.$t('serverSettingsSaved') });
      } catch (err) {
        this.error = getErrorMessage(err);
      } finally {
        this.saving = false;
      }
    },
  },
};
</script>

<style scoped>
.server-settings-page { max-width: 1120px; padding-bottom: 32px; }
</style>
