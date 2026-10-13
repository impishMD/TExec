<template>
  <v-container fluid class="pb-0 mt-8">
    <v-row class="mb-2">
      <v-col>
        <v-list subheader>
          <v-list-item>
            <v-list-item-icon>
              <v-icon>{{ fields.playbook.icon }}</v-icon>
            </v-list-item-icon>

            <v-list-item-content>
              <v-list-item-title>{{ $t(fields.playbook.title) }}</v-list-item-title>
              <v-list-item-subtitle>
                {{ template.playbook || (fields.playbook.directories ? '.' : '—') }}
              </v-list-item-subtitle>
            </v-list-item-content>
          </v-list-item>
        </v-list>
      </v-col>
      <v-col>
        <v-list subheader>
          <v-list-item>
            <v-list-item-icon>
              <v-icon>{{ TEMPLATE_TYPE_ICONS[template.type] }}</v-icon>
            </v-list-item-icon>

            <v-list-item-content>
              <v-list-item-title>{{ $t('type') }}</v-list-item-title>
              <v-list-item-subtitle>{{ $t(TEMPLATE_TYPE_TITLES[template.type]) }}
              </v-list-item-subtitle>
            </v-list-item-content>
          </v-list-item>
        </v-list>
      </v-col>
      <v-col v-if="needField('inventory')">
        <v-list subheader>
          <v-list-item>
            <v-list-item-icon>
              <v-icon>mdi-monitor</v-icon>
            </v-list-item-icon>

            <v-list-item-content>
              <v-list-item-title>{{ $t(fields.inventory.title) }}</v-list-item-title>
              <v-list-item-subtitle>
                {{ (inventory.find((x) => x.id === template.inventory_id) || {name: '—'}).name }}
              </v-list-item-subtitle>
            </v-list-item-content>
          </v-list-item>
        </v-list>
      </v-col>
      <v-col>
        <v-list subheader>
          <v-list-item>
            <v-list-item-icon>
              <v-icon>mdi-code-braces</v-icon>
            </v-list-item-icon>
            <v-list-item-content>
              <v-list-item-title>{{ $t('environment') }}</v-list-item-title>
              <v-list-item-subtitle>
                {{ environmentNames }}
              </v-list-item-subtitle>
            </v-list-item-content>
          </v-list-item>
        </v-list>
      </v-col>
      <v-col>
        <v-list subheader>
          <v-list-item>
            <v-list-item-icon>
              <v-icon>mdi-git</v-icon>
            </v-list-item-icon>
            <v-list-item-content>
              <v-list-item-title>{{ $t('repository2') }}</v-list-item-title>
              <v-list-item-subtitle>
                {{ repositories.find((x) => x.id === template.repository_id).name }}
              </v-list-item-subtitle>
            </v-list-item-content>
          </v-list-item>
        </v-list>
      </v-col>
    </v-row>

    <TaskStats :project-id="template.project_id" :template-id="template.id" />

  </v-container>

</template>
<script>
import {
  TEMPLATE_TYPE_ACTION_TITLES,
  TEMPLATE_TYPE_ICONS,
  TEMPLATE_TYPE_TITLES,
} from '@/lib/constants';
import TaskStats from '@/components/TaskStats.vue';
import AppFieldsMixin from '@/components/AppFieldsMixin';

export default {
  components: { TaskStats },
  mixins: [AppFieldsMixin],

  props: {
    template: Object,
    repositories: Array,
    inventory: Array,
    environment: Array,
  },

  data() {
    return {
      dateRanges: [{
        text: this.$t('uiPastWeek'),
        value: 'last_week',
      }, {
        text: this.$t('uiPastMonth'),
        value: 'last_month',
      }, {
        text: this.$t('uiPastYear'),
        value: 'last_year',
      }],
      users: [{
        text: this.$t('uiAllUsers'),
        value: null,
      }],
      TEMPLATE_TYPE_ICONS,
      TEMPLATE_TYPE_TITLES,
      TEMPLATE_TYPE_ACTION_TITLES,
    };
  },

  computed: {
    app() {
      return this.template.app || '';
    },

    environmentNames() {
      if (!this.template || !Array.isArray(this.template.environment_ids)
          || this.template.environment_ids.length === 0) {
        return '—';
      }
      const names = this.template.environment_ids
        .map((id) => {
          const env = this.environment.find((x) => x.id === id);
          return env ? env.name : null;
        })
        .filter((name) => name != null);
      return names.length > 0 ? names.join(', ') : '—';
    },
  },
};
</script>
