<template>
  <div class="WorkflowRun">
    <WorkflowHeader :project-id="projectId" @show-drawer="showDrawer()">
      <template #title>
        <h1>{{ workflow ? workflow.name : $t('workflowRun') }}</h1>
      </template>
      <template #metadata>
        <v-chip small outlined>{{ $t('workflowRun') }} #{{ runId }}</v-chip>
        <v-chip v-if="details && details.run.version" small outlined>
          {{ details.run.version }}
        </v-chip>
        <v-chip
          v-if="details && details.revision"
          small outlined :title="$t('workflowRunRevisionHint')"
        >{{ $t('workflowRevisionLabel', { number: details.revision }) }}</v-chip>
        <v-chip
          v-if="details"
          :color="statusColor(details.run.status)"
          small
          outlined
          class="WorkflowRun__status"
        >{{ statusTitle(details.run.status) }}</v-chip>
      </template>
      <template #actions>
        <v-btn
          v-if="canStopRun"
          color="error"
          small
          outlined
          :loading="stopping"
          @click="stopRun()"
        >
          <v-icon left small>mdi-stop</v-icon>
          {{ $t('stop') }}
        </v-btn>
        <v-btn icon @click="loadData()" :title="$t('refresh')">
          <v-icon>mdi-refresh</v-icon>
        </v-btn>
      </template>
    </WorkflowHeader>

    <v-divider />

    <div class="WorkflowRun__body">
      <template v-if="details != null">
        <v-expansion-panels v-if="deliveries.length" flat class="WorkflowRun__deliveries">
          <v-expansion-panel>
            <v-expansion-panel-header>
              {{ $t('notifications') }} ({{ deliveries.length }})
            </v-expansion-panel-header>
            <v-expansion-panel-content>
              <div v-for="delivery in deliveries" :key="delivery.id" class="mb-3">
                <strong>
                  #{{ delivery.workflow_node_id }} · {{ $t(`workflowDelivery_${delivery.status}`) }}
                </strong>
                <span class="ml-2 text--secondary">
                  {{ $t('workflowNotificationAttempts', { count: delivery.attempts }) }}
                </span>
                <div v-if="delivery.last_error" class="error--text">
                  {{ notificationError(delivery.last_error) }}
                </div>
              </div>
            </v-expansion-panel-content>
          </v-expansion-panel>
        </v-expansion-panels>
        <div class="WorkflowRun__graph">
          <WorkflowGraph
            v-if="workflow"
            ref="graph"
            :nodes="workflow.nodes || []"
            :edges="workflow.edges || []"
            :templates="templates"
            :node-runs="nodeRuns"
            :can-resolve-approvals="canResolveApprovals"
            :editable="false"
            @node-click="onNodeClicked"
            @resolve-approval="onResolveApproval"
          />

          <!-- Narrow screens: the in-card buttons are hard to hit, keep the bar. -->
          <div
            v-if="canResolveApprovals && pendingApprovals.length && $vuetify.breakpoint.smAndDown"
            class="WorkflowRun__approvals"
          >
            <v-card
              v-for="a in pendingApprovals"
              :key="`approval-${a.nodeId}`"
              class="WorkflowRun__approval px-3 py-2"
              outlined
            >
              <div class="WorkflowRun__approvalText">
                <strong>#{{ a.nodeId }}</strong>
                <span class="ml-2">{{ a.message || $t('workflowApprovalPending') }}</span>
              </div>
              <v-spacer />
              <v-btn
                small
                color="success"
                class="ml-3"
                @click="resolveApproval(a.nodeId, 'approved')"
              >{{ $t('workflowApprove') }}</v-btn>
              <v-btn
                small
                color="error"
                class="ml-2"
                @click="resolveApproval(a.nodeId, 'rejected')"
              >{{ $t('workflowReject') }}</v-btn>
            </v-card>
          </div>
        </div>
      </template>

      <div v-else class="pa-4 text-center">
        <v-progress-circular indeterminate color="primary" />
      </div>
    </div>
  </div>
</template>

<style lang="scss">
.WorkflowRun {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 48px);
  height: calc(100dvh - 48px);
  min-height: 480px;
  min-width: 0;

  @media (max-width: 959px) {
    height: calc(100vh - 32px);
    height: calc(100dvh - 32px);
  }

  &__deliveries { flex: 0 0 auto; max-height: 220px; overflow-y: auto; overflow-wrap: anywhere; }

  &__body {
    flex: 1 1 auto;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }

  &__graph {
    position: relative;
    flex: 1 1 auto;
    min-height: 0;
  }

  &__approvals {
    position: absolute;
    left: 50%;
    bottom: 16px;
    transform: translateX(-50%);
    display: flex;
    flex-direction: column;
    gap: 8px;
    width: min(420px, 92%);
    z-index: 5;
  }

  &__approval {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 8px;
    border-left: 3px solid #ff9800 !important;
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.25) !important;
  }

  &__approvalText {
    flex: 0 0 100%;
    min-width: 0;
    font-size: 13px;
    word-break: break-word;
  }
}
</style>

<script>
import DisplayLabelsMixin from '@/components/DisplayLabelsMixin';
import axios from 'axios';
import EventBus from '@/event-bus';
import { getErrorMessage } from '@/lib/error';
import PermissionsCheck from '@/components/PermissionsCheck';
import WorkflowGraph from '@/components/WorkflowGraph.vue';
import WorkflowHeader from '@/components/workflow/WorkflowHeader.vue';
import { USER_PERMISSIONS } from '@/lib/constants';
import { notificationRunStatus } from '@/lib/workflowNotifications';

export default {
  components: { WorkflowGraph, WorkflowHeader },
  mixins: [DisplayLabelsMixin, PermissionsCheck],
  props: {
    projectId: Number,
  },
  data() {
    return {
      details: null,
      workflow: null,
      templates: [],
      pollHandle: null,
      stopping: false,
      USER_PERMISSIONS,
    };
  },
  computed: {
    workflowId() {
      return parseInt(this.$route.params.workflowId, 10);
    },
    runId() {
      return parseInt(this.$route.params.runId, 10);
    },
    canResolveApprovals() {
      return this.can(USER_PERMISSIONS.runProjectTasks);
    },
    // A run can be stopped while it is still in progress (executing tasks or
    // blocked on an approval) and the user may run project tasks.
    canStopRun() {
      if (!this.details) return false;
      const { status } = this.details.run;
      return (status === 'running' || status === 'approval')
        && this.can(USER_PERMISSIONS.runProjectTasks);
    },
    nodeRuns() {
      const map = {};
      (this.details?.nodes || []).forEach((n) => {
        const id = n.node.id;
        if (n.task) {
          map[id] = {
            status: n.task.status,
            start: n.task.start || n.task.created,
            end: n.task.end,
            taskId: n.task.id,
          };
        } else if (n.approval) {
          map[id] = {
            status: n.approval.status,
            start: n.approval.created,
            end: n.approval.resolved,
            message: n.node.approval_message,
          };
        } else if (n.delay) {
          map[id] = {
            status: n.delay.status,
            start: n.delay.created,
            end: n.delay.resolved,
            resumeAt: n.delay.status === 'waiting' ? n.delay.resume_at : null,
          };
        } else if (n.notification) {
          map[id] = {
            status: notificationRunStatus(n.notification, n.node.notification),
            start: n.notification.created,
            end: n.notification.resolved,
          };
        }
        if (map[id] && n.notification) map[id].delivery = n.notification;
      });
      return map;
    },
    deliveries() {
      return (this.details?.nodes || []).filter((n) => n.notification).map((n) => n.notification);
    },
    pendingApprovals() {
      return (this.details?.nodes || [])
        .filter((n) => n.approval && n.approval.status === 'pending')
        .map((n) => ({ nodeId: n.node.id, message: n.node.approval_message }));
    },
  },
  async created() {
    await this.loadData();
    this.pollHandle = setInterval(() => {
      const status = this.details && this.details.run.status;
      if (status === 'running' || status === 'approval') {
        this.loadData();
      } else if (this.pollHandle) {
        clearInterval(this.pollHandle);
        this.pollHandle = null;
      }
    }, 5000);
  },
  beforeDestroy() {
    if (this.pollHandle) clearInterval(this.pollHandle);
  },
  methods: {
    notificationError(error) { return getErrorMessage({ response: { data: { error } } }); },
    showDrawer() {
      EventBus.$emit('i-show-drawer');
    },
    // Clicking a node that has a task (running or finished) opens its task log.
    onNodeClicked(nodeId) {
      if (nodeId == null) return;
      const entry = (this.details?.nodes || []).find((n) => n.node.id === nodeId);
      if (entry && entry.task) {
        EventBus.$emit('i-show-task', { taskId: entry.task.id });
      }
    },
    onResolveApproval({ nodeId, status }) {
      this.resolveApproval(nodeId, status);
    },
    statusColor(status) {
      switch (status) {
        case 'success':
        case 'approved':
          return 'success';
        case 'failed':
        case 'error':
        case 'stopped':
        case 'rejected':
          return 'error';
        case 'running':
        case 'pending':
          return 'primary';
        case 'approval':
          return 'warning';
        default:
          return 'grey';
      }
    },
    async stopRun() {
      this.stopping = true;
      try {
        await axios.post(
          `/api/project/${this.projectId}/workflows/${this.workflowId}/runs/${this.runId}/stop`,
        );
        await this.loadData();
      } catch (err) {
        EventBus.$emit('i-snackbar', {
          color: 'error',
          text: getErrorMessage(err),
        });
      } finally {
        this.stopping = false;
      }
    },
    async resolveApproval(nodeId, status) {
      try {
        await axios.post(
          `/api/project/${this.projectId}/workflows/${this.workflowId}/runs/${this.runId}/approvals/${nodeId}`,
          { status },
        );
        await this.loadData();
      } catch (err) {
        EventBus.$emit('i-snackbar', {
          color: 'error',
          text: getErrorMessage(err),
        });
      }
    },
    async loadData() {
      try {
        const [details, templates] = await Promise.all([
          axios.get(
            `/api/project/${this.projectId}/workflows/${this.workflowId}/runs/${this.runId}`,
          ),
          this.templates.length
            ? Promise.resolve({ data: this.templates })
            : axios.get(`/api/project/${this.projectId}/templates`),
        ]);
        this.details = details.data;
        this.templates = templates.data || [];
        // The graph comes from the revision the run pinned (not from the
        // template's current graph, which may have been edited since) and is
        // built once from the first payload; later polls only refresh
        // statuses, so the user's pan/zoom survives.
        if (!this.workflow) {
          this.workflow = {
            name: this.details.workflow_name,
            nodes: (this.details.nodes || []).map((n) => n.node),
            edges: this.details.edges || [],
          };
        }
      } catch (err) {
        EventBus.$emit('i-snackbar', {
          color: 'error',
          text: getErrorMessage(err),
        });
      }
    },
  },
};
</script>
