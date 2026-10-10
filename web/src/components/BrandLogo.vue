<template>
  <span
    class="brand-logo"
    :class="{ 'brand-logo--compact': compact, 'brand-logo--large': large }"
  >
    <img
      class="brand-logo__symbol"
      :src="symbol"
      :width="large ? 52 : 36"
      :height="large ? 52 : 36"
      :alt="compact ? 'TExec' : ''"
      draggable="false"
    />
    <span v-if="!compact" class="brand-logo__text">
      <img
        class="brand-logo__name"
        :src="wordmark"
        :width="large ? 112 : 90"
        :height="large ? 30 : 24"
        alt="TExec"
        draggable="false"
      />
      <span v-if="tagline" class="brand-logo__tagline">{{ tagline }}</span>
    </span>
  </span>
</template>

<script>
import wordmarkLight from '@/assets/brand/wordmark-light.svg';
import wordmarkDark from '@/assets/brand/wordmark-dark.svg';
import symbolLight from '@/assets/brand/symbol-light.svg';
import symbolDark from '@/assets/brand/symbol-dark.svg';

export default {
  name: 'BrandLogo',
  props: {
    dark: Boolean,
    compact: Boolean,
    large: Boolean,
    tagline: { type: String, default: '' },
  },
  computed: {
    symbol() {
      return this.dark ? symbolDark : symbolLight;
    },
    wordmark() {
      return this.dark ? wordmarkDark : wordmarkLight;
    },
  },
};
</script>

<style lang="scss" scoped>
.brand-logo {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  max-width: 100%;

  &__symbol {
    display: block;
    flex: 0 0 auto;
    width: 36px;
    height: 36px;
    object-fit: contain;
  }

  &__text {
    display: block;
    min-width: 0;
  }

  &__name {
    display: block;
    width: 90px;
    max-width: 100%;
    height: auto;
  }

  &__tagline {
    display: block;
    margin-top: 5px;
    font-size: 10px;
    line-height: 1.35;
    white-space: normal;
    color: var(--texec-nav-muted, var(--texec-muted));
  }

  &--compact .brand-logo__symbol {
    max-width: 100%;
    flex-shrink: 1;
  }

  &--large {
    gap: 14px;

    .brand-logo__symbol { width: 52px; height: 52px; }
    .brand-logo__name { width: 112px; }
    .brand-logo__tagline { font-size: 12px; color: var(--texec-muted); }
  }
}
</style>
