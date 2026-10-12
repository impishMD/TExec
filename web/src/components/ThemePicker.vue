<template>
  <div class="theme-picker">
    <p class="text-body-2 text--secondary">{{ $t('appearanceModesHint') }}</p>
    <fieldset :disabled="disabled" class="theme-picker__fieldset">
      <legend class="sr-only">{{ $t('appearance') }}</legend>
      <label
        :for="`${groupName}-default`"
        v-if="inherit"
        class="theme-picker__inherit"
        :class="{ 'is-selected': value === '' }"
      >
        <input
          :id="`${groupName}-default`"
          type="radio"
          :name="groupName"
          value=""
          :checked="value === ''"
          data-testid="theme-default"
          @change="$emit('input', '')"
        />
        <span>
          <strong>{{ $t('appearanceDefault') }}</strong>
          <span class="theme-picker__hint">{{
            $t('appearanceInheritHint', { theme: themeName(defaultTheme) })
          }}</span>
        </span>
      </label>
      <template v-for="group in groups">
        <h3 :key="`${group.key}-title`" class="text-subtitle-1 mt-5 mb-3">
          {{ $t(group.key) }}
        </h3>
        <p v-if="group.accessible" :key="`${group.key}-hint`" class="text-body-2 text--secondary">
          {{ $t('appearanceContrastHint') }}
        </p>
        <div :key="group.key" class="theme-picker__grid">
          <label
            :for="`${groupName}-${theme.id}`"
            v-for="theme in group.themes"
            :key="theme.id"
            class="theme-picker__choice"
            :class="{ 'is-selected': value === theme.id }"
            :data-testid="`theme-${theme.id}`"
          >
            <span class="theme-picker__swatches" aria-hidden="true">
              <span
                v-for="mode in ['light', 'dark']"
                :key="mode"
                class="theme-picker__swatch"
                :style="previewStyle(theme.id, mode === 'dark')"
              >
                <span class="theme-picker__sidebar"><i /><i /><i /></span>
                <span class="theme-picker__content"><b /><i /><i /><em /></span>
                <v-icon size="12" class="theme-picker__mode">
                  {{ mode === 'dark' ? 'mdi-weather-night' : 'mdi-white-balance-sunny' }}
                </v-icon>
              </span>
            </span>
            <span class="theme-picker__name">
              <input
                :id="`${groupName}-${theme.id}`"
                type="radio"
                :name="groupName"
                :value="theme.id"
                :checked="value === theme.id"
                @change="$emit('input', theme.id)"
              />
              <span>{{
                theme.id === 'texec' && !inherit ? $t('appearanceDefault') : themeName(theme.id)
              }}</span>
              <v-icon v-if="value === theme.id" size="18" color="primary">mdi-check-circle</v-icon>
            </span>
          </label>
        </div>
      </template>
    </fieldset>
    <div
      class="theme-picker__preview mt-6"
      :style="previewStyle(effectiveTheme, $vuetify.theme.dark)"
      data-testid="theme-preview"
      :data-theme="effectiveTheme"
    >
      <div class="theme-picker__preview-heading">
        <strong>{{ $t('appearancePreview') }} · {{ themeName(effectiveTheme) }}</strong>
        <span>{{ $t($vuetify.theme.dark ? 'appearanceDark' : 'appearanceLight') }}</span>
      </div>
      <div class="theme-picker__sample">
        <div>
          <strong>{{ $t('taskTemplates') }}</strong>
          <p>{{ $t('appearanceSample') }}</p>
        </div>
        <span class="theme-picker__sample-button">{{ $t('run') }}</span>
      </div>
      <div class="theme-picker__statuses">
        <span :style="{ color: palette.success }"
          ><v-icon size="18" :color="palette.success">mdi-check-circle</v-icon>
          {{ $t('status_success') }}</span
        >
        <span :style="{ color: palette.error }"
          ><v-icon size="18" :color="palette.error">mdi-alert-circle</v-icon>
          {{ $t('status_failed') }}</span
        >
      </div>
    </div>
  </div>
</template>

<script>
import {
  THEMES, themeById, themePalette, themeVariables, resolveTheme,
} from '@/lib/appearance';

export default {
  props: {
    value: { type: String, default: 'texec' },
    defaultTheme: { type: String, default: 'texec' },
    inherit: Boolean,
    disabled: Boolean,
    groupName: { type: String, default: 'appearance-theme' },
  },
  computed: {
    groups() {
      return [false, true].map((accessible) => ({
        accessible,
        key: accessible ? 'appearanceAccessible' : 'appearancePalettes',
        themes: THEMES.filter((theme) => theme.accessible === accessible),
      }));
    },
    effectiveTheme() {
      return resolveTheme(this.value, this.defaultTheme);
    },
    palette() {
      return themePalette(this.effectiveTheme, this.$vuetify.theme.dark);
    },
  },
  methods: {
    previewStyle: themeVariables,
    themeName(id) {
      const theme = themeById(id);
      return theme.accessible ? this.$t(`appearanceTheme_${theme.id}`) : theme.name;
    },
  },
};
</script>

<style lang="scss" scoped>
.theme-picker {
  &__fieldset {
    border: 0;
    padding: 0;
    min-width: 0;
  }
  &__fieldset:disabled {
    opacity: 0.6;
  }
  &__grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(175px, 1fr));
    gap: 14px;
  }
  &__choice,
  &__inherit {
    cursor: pointer;
    border: 2px solid var(--texec-border);
    border-radius: 12px;
    background: var(--texec-surface);
    color: var(--texec-text);
    min-width: 0;
    &:hover {
      border-color: var(--texec-muted);
    }
    &:focus-within {
      outline: 3px solid var(--texec-accent);
      outline-offset: 3px;
    }
    &.is-selected {
      border-color: var(--texec-accent);
      background: var(--texec-accent-soft);
    }
  }
  &__fieldset:disabled label {
    cursor: default;
  }
  &__inherit {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 14px 16px;
  }
  &__hint {
    display: block;
    color: var(--texec-muted);
    font-size: 13px;
    margin-top: 4px;
  }
  input {
    width: 17px;
    height: 17px;
    accent-color: var(--texec-accent);
    flex-shrink: 0;
  }
  &__swatches {
    display: flex;
    gap: 2px;
    padding: 6px 6px 0;
  }
  &__swatch {
    flex: 1;
    height: 72px;
    background: var(--texec-bg);
    display: flex;
    position: relative;
    border-radius: 6px;
    overflow: hidden;
  }
  &__sidebar {
    width: 22%;
    background: var(--texec-soft);
    padding: 14px 4px;
  }
  &__sidebar i {
    display: block;
    height: 4px;
    background: var(--texec-muted);
    margin-bottom: 7px;
    border-radius: 2px;
  }
  &__sidebar i:first-child {
    background: var(--texec-accent);
  }
  &__content {
    padding: 12px 9px;
    flex: 1;
  }
  &__content b,
  &__content i,
  &__content em {
    display: block;
    height: 4px;
    background: var(--texec-muted);
    border-radius: 2px;
    margin-bottom: 6px;
  }
  &__content b {
    width: 70%;
    background: var(--texec-text);
  }
  &__content i {
    opacity: 0.6;
  }
  &__content em {
    background: var(--texec-accent);
    width: 40%;
    height: 8px;
  }
  &__mode.v-icon {
    position: absolute;
    bottom: 5px;
    right: 5px;
    color: var(--texec-muted);
  }
  &__name {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px;
    font-size: 14px;
    line-height: 1.4;
  }
  &__name > span {
    flex: 1;
  }
  &__preview {
    background: var(--texec-bg);
    color: var(--texec-text);
    border: 1px solid var(--texec-border);
    border-radius: 14px;
    padding: 20px;
  }
  &__preview-heading {
    display: flex;
    gap: 12px;
    justify-content: space-between;
    flex-wrap: wrap;
    font-size: 13px;
  }
  &__preview-heading > span {
    color: var(--texec-muted);
  }
  &__sample {
    margin-top: 16px;
    display: flex;
    align-items: center;
    gap: 12px;
    justify-content: space-between;
    background: var(--texec-surface);
    border: 1px solid var(--texec-border);
    border-radius: 10px;
    padding: 16px;
  }
  &__sample p {
    color: var(--texec-muted);
    margin: 6px 0 0;
    font-size: 13px;
  }
  &__sample-button {
    background: var(--texec-accent);
    color: var(--texec-on-primary);
    padding: 8px 16px;
    border-radius: 8px;
    font-weight: 600;
  }
  &__statuses {
    display: flex;
    flex-wrap: wrap;
    gap: 20px;
    margin-top: 14px;
    font-size: 13px;
  }
  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip-path: inset(50%);
  }
  @media (max-width: 420px) {
    &__grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 10px;
    }
    &__name {
      font-size: 12px;
      padding: 8px;
      flex-wrap: wrap;
    }
    &__preview {
      padding: 12px;
    }
    &__sample {
      flex-wrap: wrap;
    }
  }
}
</style>
