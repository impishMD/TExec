<template>
  <div>
    <v-toolbar flat>
      <v-app-bar-nav-icon @click="showDrawer" />
      <v-toolbar-title>{{ $t('appearance') }}</v-toolbar-title>
    </v-toolbar>
    <v-divider />
    <div class="appearance-page mx-auto mt-8">
      <p>{{ $t('appearancePersonalHint') }}</p>
      <v-progress-linear v-if="loading" indeterminate />
      <v-alert v-if="error" type="error" text>{{ error }}</v-alert>
      <v-btn v-if="!loaded && !loading" text @click="load">{{ $t('alertsRetry') }}</v-btn>
      <v-card v-if="loaded" outlined class="pa-6">
        <v-form @submit.prevent="save">
          <ThemePicker
            v-model="selected"
            inherit
            :default-theme="defaultTheme"
            :disabled="saving"
          />
          <div class="d-flex justify-end mt-6">
            <v-btn
              type="submit"
              color="primary"
              :disabled="!dirty || saving"
              :loading="saving"
              data-testid="appearance-save"
              >{{ $t('save') }}</v-btn
            >
          </div>
        </v-form>
      </v-card>
    </div>
  </div>
</template>

<script>
import axios from 'axios';
import EventBus from '@/event-bus';
import ThemePicker from '@/components/ThemePicker.vue';
import { getErrorMessage } from '@/lib/error';
import { personalThemeFromOptions } from '@/lib/appearance';

export default {
  components: { ThemePicker },
  props: { systemInfo: Object },
  data: () => ({
    selected: '', saved: '', loading: true, loaded: false, saving: false, error: '',
  }),
  computed: {
    defaultTheme() {
      return this.systemInfo?.appearance_theme || 'texec';
    },
    dirty() {
      return this.selected !== this.saved;
    },
  },
  created() {
    this.load();
  },
  methods: {
    showDrawer() {
      EventBus.$emit('i-show-drawer');
    },
    async load() {
      this.loading = true;
      this.error = '';
      try {
        const options = (await axios.get('/api/user/options')).data;
        this.saved = personalThemeFromOptions(options);
        this.selected = this.saved;
        this.loaded = true;
      } catch (err) {
        this.error = getErrorMessage(err);
      } finally {
        this.loading = false;
      }
    },
    async save() {
      if (!this.dirty || this.saving) return;
      this.saving = true;
      this.error = '';
      try {
        await axios.post('/api/user/options', {
          key: 'appearance.theme',
          value: JSON.stringify(this.selected),
        });
        this.saved = this.selected;
        EventBus.$emit('i-appearance', this.saved);
        EventBus.$emit('i-snackbar', { color: 'success', text: this.$t('appearanceSaved') });
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
.appearance-page {
  max-width: 1120px;
  padding-bottom: 32px;
}
</style>
