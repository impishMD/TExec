import './setup';
import { expect } from 'chai';
import { mount, createLocalVue } from '@vue/test-utils';
import Vuetify from 'vuetify';
import axios from 'axios';
import WorkflowNotificationForm from '@/components/workflow/WorkflowNotificationForm.vue';
import {
  defaultNotification, notificationProblem, notificationSources, notificationRunStatus,
} from '@/lib/workflowNotifications';

describe('Workflow notifications', () => {
  it('validates substitutions, Unicode length and timezones', () => {
    const config = defaultNotification('ru', true);
    expect(notificationProblem(config)).to.equal(null);
    expect(config.message).to.include('Требуется подтверждение');
    expect(notificationProblem({ ...config, message: '{{ secrets.token }}' }))
      .to.equal('workflowNotificationVariableInvalid');
    expect(notificationProblem({ ...config, message: '{{ date' }))
      .to.equal('workflowNotificationVariableInvalid');
    expect(notificationProblem({ ...config, message: '😀'.repeat(3000) })).to.equal(null);
    expect(notificationProblem({ ...config, timezone: 'Invalid/Zone' }))
      .to.equal('workflowNotificationTimezoneInvalid');
  });

  it('offers only upstream sources across branches', () => {
    const nodes = [1, 2, 3, 4, 5].map((id) => ({ id, kind: 'task' }));
    const edges = [[1, 2], [1, 3], [2, 4], [3, 5]].map(([s, d]) => ({
      source_node_id: s, destination_node_id: d,
    }));
    expect(notificationSources(nodes, edges, 4).map((n) => n.id)).to.deep.equal([1, 2]);
    expect(notificationSources(nodes, edges, 1)).to.deep.equal([]);
  });

  it('distinguishes delivery failure from the configured graph result', () => {
    expect(notificationRunStatus({ status: 'failed' }, { on_error: 'continue' })).to.equal('success');
    expect(notificationRunStatus({ status: 'failed' }, { on_error: 'fail' })).to.equal('error');
    expect(notificationRunStatus({ status: 'queued' }, {})).to.equal('running');
  });

  it('ignores stale previews, escapes output and sends tests only on explicit action', async () => {
    const original = axios.post;
    const calls = [];
    axios.post = (url) => new Promise((resolve) => { calls.push({ url, resolve }); });
    const wrapper = mount(WorkflowNotificationForm, {
      localVue: createLocalVue(),
      vuetify: new Vuetify(),
      propsData: { projectId: 1, value: defaultNotification('ru') },
      mocks: { $t: (key) => key },
    });
    try {
      clearTimeout(wrapper.vm.timer);
      const old = wrapper.vm.loadPreview(wrapper.vm.sequence);
      await wrapper.setProps({ value: { ...defaultNotification('ru'), message: 'Updated' } });
      clearTimeout(wrapper.vm.timer);
      const current = wrapper.vm.loadPreview(wrapper.vm.sequence);
      calls[1].resolve({ data: { text: '<img src=x onerror=alert(1)>' } });
      await current;
      calls[0].resolve({ data: { text: 'stale preview' } });
      await old;
      await wrapper.vm.$nextTick();
      expect(wrapper.vm.preview).to.equal('<img src=x onerror=alert(1)>');
      expect(wrapper.find('.WorkflowNotificationForm__text img').exists()).to.equal(false);
      expect(calls.every((call) => call.url.endsWith('/preview'))).to.equal(true);
      const send = wrapper.vm.testNotification();
      expect(calls[2].url).to.equal('/api/project/1/workflows/notifications/test');
      calls[2].resolve({ data: {} });
      await send;
      expect(wrapper.vm.testResult.type).to.equal('success');
    } finally {
      wrapper.destroy();
      axios.post = original;
    }
  });
});
