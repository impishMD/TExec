<template>
  <span
    class="brand-logo"
    :class="{ 'brand-logo--compact': compact, 'brand-logo--large': large }"
  >
    <svg
      class="brand-logo__symbol"
      viewBox="0 0 410 280"
      :width="large ? 52 : 36"
      :height="large ? 52 : 36"
      :role="compact ? 'img' : null"
      :aria-label="compact ? 'TExec' : null"
      :aria-hidden="compact ? null : 'true'"
      focusable="false"
    >
      <g fill="none">
        <path
          class="brand-logo__accent"
          d="M 103,88 H 193"
          stroke="currentColor" stroke-width="23" stroke-linecap="round"
        />
        <path
          d="M 35,140 H 193"
          stroke="currentColor" stroke-width="23" stroke-linecap="round"
        />
        <path
          class="brand-logo__accent"
          d="M 35,128.5 H 148 V 151.5 H 35 A 11.5,11.5 0 0 1 35,128.5 Z"
          fill="currentColor"
        />
        <path
          class="brand-logo__accent"
          d="M 103,192 H 193"
          stroke="currentColor" stroke-width="23" stroke-linecap="round"
        />
        <path
          d="M 246,20 V 260"
          stroke="currentColor" stroke-width="24" stroke-linecap="round"
        />
        <path
          class="brand-logo__accent"
          d="M 298,65 Q 298,61 302,64 L 391,132 Q 401,140 391,148 L 302,216 Q 298,219 298,215 Z"
          fill="currentColor"
        />
      </g>
    </svg>
    <span v-if="!compact" class="brand-logo__text">
      <svg
        class="brand-logo__name"
        viewBox="6 34 356 96"
        :width="large ? 112 : 90"
        :height="large ? 30 : 24"
        role="img"
        aria-label="TExec"
        focusable="false"
      >
        <g transform="translate(6 127)" fill="currentColor">
          <path
            class="brand-logo__accent"
            d="M42 1184H481V0H841V1184H1279V1490H42Z"
            transform="translate(0.000,0) scale(0.0610352,-0.0610352)"
          />
          <path
            d="M96 0H1206V306H456V600H1149V897H456V1184H1206V1490H96Z"
            transform="translate(77.527,0) scale(0.0610352,-0.0610352)"
          />
          <path
            d="M4 0 H377 L460 131 C498 196 531 254 564 312 C597 254 630 196 667 131 L747 0
              H1120 L754 547 L1104 1056 H738 L663 933 C628 873 598 818 568 764 C539 818
              509 873 476 933 L403 1056 H32 L376 538 Z"
            transform="translate(151.820,0) scale(0.0610352,-0.0610352)"
          />
          <path
            d="M596 -24 C863 -24 1076 125 1115 338 H806 C781 268 709 224 608 224 C472 224
              392 309 388 443 H1123 V536 C1123 856 904 1080 587 1080 C274 1080 52 850 52
              528 C52 206 272 -24 596 -24 Z M390 649 C405 760 480 826 596 826 C711 826 787
              760 802 649 Z"
            transform="translate(217.385,0) scale(0.0610352,-0.0610352)"
          />
          <path
            d="M598 -24 C897 -24 1109 160 1116 418 H783 C770 315 707 249 605 249 C477 249
              401 355 401 528 C401 701 476 807 605 807 C707 807 773 742 784 641 H1116
              C1101 900 892 1080 598 1080 C271 1080 52 856 52 528 C52 199 271 -24 598 -24
              Z"
            transform="translate(285.757,0) scale(0.0610352,-0.0610352)"
          />
        </g>
      </svg>
      <span v-if="tagline" class="brand-logo__tagline">{{ tagline }}</span>
    </span>
  </span>
</template>

<script>
export default {
  name: 'BrandLogo',
  props: {
    compact: Boolean,
    large: Boolean,
    tagline: { type: String, default: '' },
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
  color: var(--texec-text, currentColor);

  &__accent {
    color: var(--texec-accent, currentColor);
  }

  &__symbol {
    display: block;
    flex: 0 0 auto;
    width: 36px;
    height: 36px;
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
