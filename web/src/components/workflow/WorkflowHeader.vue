<template>
  <header class="WorkflowHeader">
    <div class="WorkflowHeader__navigation">
      <v-app-bar-nav-icon @click="$emit('show-drawer')" />
      <router-link :to="`/project/${projectId}/workflows`">
        {{ $t('workflows') }}
      </router-link>
    </div>
    <div class="WorkflowHeader__title"><slot name="title" /></div>
    <div class="WorkflowHeader__toolbar">
      <div class="WorkflowHeader__metadata"><slot name="metadata" /></div>
      <div class="WorkflowHeader__actions"><slot name="actions" /></div>
    </div>
  </header>
</template>

<script>
export default {
  props: { projectId: Number },
};
</script>

<style lang="scss">
.WorkflowHeader {
  flex: 0 0 auto;
  min-width: 0;
  padding: 16px 0;

  &__navigation {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;
    font-size: 14px;
    line-height: 20px;

    a { text-decoration: none; }
    a:hover { text-decoration: underline; }
  }

  &__title {
    min-width: 0;
    font-size: 24px;
    font-weight: 600;
    line-height: 1.35;
    overflow-wrap: anywhere;

    h1 { font: inherit; margin: 0; }
  }

  &__toolbar, &__metadata, &__actions {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 8px;
    min-width: 0;
  }

  &__toolbar { margin-top: 12px; gap: 12px 24px; }
  &__metadata { flex: 1 1 auto; }
  &__actions { margin-left: auto; }

  // Chips must keep their full labels when the toolbar wraps.
  .v-chip {
    flex: 0 0 auto;
    max-width: 100%;
    height: auto;
    min-height: 28px;
    padding: 4px 10px;
    font-size: 13px;
    line-height: 20px;

    .v-chip__content {
      height: auto;
      min-width: 0;
      white-space: normal;
      overflow-wrap: anywhere;
    }
  }

  .v-btn, .v-badge { flex: 0 0 auto; }

  @media (max-width: 959px) {
    padding: 8px 0 16px;
    &__title { font-size: 20px; }
  }
}
</style>
