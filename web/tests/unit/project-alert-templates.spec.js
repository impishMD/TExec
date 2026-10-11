import { expect } from 'chai';
import { mount } from '@vue/test-utils';
import axios from 'axios';
import Vuetify from 'vuetify';
import ProjectAlertTemplates from '@/components/ProjectAlertTemplates.vue';

const tick = () => new Promise((resolve) => { setTimeout(resolve, 0); });
const settings = {
  success: 'OK {{task.id}}', error: 'Error {{task.error}}', locale: 'en', timezone: 'UTC',
};
const response = () => ({
  settings: { ...settings },
  configured: true,
  variables: ['task.id', 'task.error', 'timezone'],
  defaults: { en: settings, ru: { ...settings, success: 'Успешно', error: 'Ошибка' } },
});

describe('Project message templates', () => {
  let wrapper;
  let original;
  let sent;
  beforeEach(() => {
    original = { get: axios.get, put: axios.put, post: axios.post };
    sent = [];
    axios.get = async () => ({ data: response() });
    axios.post = async (url, data) => { sent.push({ url, data }); return { data: { text: '<b>example</b>' } }; };
    axios.put = async (url, data) => ({ data: { ...data.settings } });
  });
  afterEach(() => {
    wrapper?.destroy();
    Object.assign(axios, original);
  });
  async function render() {
    wrapper = mount(ProjectAlertTemplates, {
      propsData: { projectId: 1 }, vuetify: new Vuetify(), mocks: { $t: (key) => key },
    });
    await tick();
  }

  it('keeps previews plain, tests only on request, and saves both templates together', async () => {
    await render();
    expect(wrapper.vm.dirty).to.equal(false);
    await wrapper.vm.loadPreview(wrapper.vm.sequence);
    await tick();
    expect(wrapper.find('[data-testid="message-preview"]').text()).to.equal('<b>example</b>');
    expect(wrapper.find('[data-testid="message-preview"] b').exists()).to.equal(false);
    expect(sent.some((call) => call.url.endsWith('/test'))).to.equal(false);
    await wrapper.setData({ settings: { ...settings, success: 'New {{task.id}}' } });
    await wrapper.vm.testNotification();
    expect(sent.filter((call) => call.url.endsWith('/test'))).to.have.length(1);
    expect(wrapper.vm.dirty).to.equal(true, 'test does not save');
    let saved;
    axios.put = async (url, data) => {
      saved = data.settings;
      return { data: { ...data.settings } };
    };
    await wrapper.vm.save();
    expect(saved.error).to.equal(settings.error);
    expect(saved.success).to.equal('New {{task.id}}');
    expect(wrapper.vm.dirty).to.equal(false);
  });

  it('validates both tabs, resets only the selected template, and cancels edits', async () => {
    await render();
    await wrapper.setData({ settings: { ...settings, error: '{{secret}}' } });
    expect(wrapper.vm.invalid).to.equal(true);
    expect(wrapper.vm.otherProblem).to.equal('workflowNotificationVariableInvalid');
    expect(wrapper.find('[data-testid="message-save"]').attributes('disabled')).to.equal('disabled');
    await wrapper.setData({ status: 'error' });
    wrapper.vm.resetTemplate();
    await tick();
    expect(wrapper.vm.invalid).to.equal(false);
    expect(wrapper.vm.settings.success).to.equal(settings.success);
    await wrapper.setData({ settings: { ...settings, success: 'Edited', timezone: 'not/a/zone' } });
    expect(wrapper.vm.problem).to.equal('workflowNotificationTimezoneInvalid');
    wrapper.vm.cancel();
    await tick();
    expect(wrapper.vm.settings).to.deep.equal(settings);
  });

  it('does not let late previews replace a newer tab or project', async () => {
    await render();
    const pending = [];
    axios.post = () => new Promise((resolve) => { pending.push(resolve); });
    const first = wrapper.vm.loadPreview(wrapper.vm.sequence);
    await wrapper.setData({ status: 'error' });
    const second = wrapper.vm.loadPreview(wrapper.vm.sequence);
    pending[1]({ data: { text: 'new' } });
    await second;
    pending[0]({ data: { text: 'old' } });
    await first;
    expect(wrapper.vm.preview).to.equal('new');
    const third = wrapper.vm.loadPreview(wrapper.vm.sequence);
    await wrapper.setProps({ projectId: 2 });
    await tick();
    pending[2]({ data: { text: 'wrong project' } });
    await third;
    expect(wrapper.vm.preview).not.to.equal('wrong project');
  });

  it('shows load failures and permits saving initial defaults', async () => {
    axios.get = async () => { throw new Error('offline'); };
    await render();
    expect(wrapper.vm.settings).to.equal(null);
    expect(wrapper.vm.error).not.to.equal('');
    axios.get = async () => ({ data: { ...response(), configured: false } });
    await wrapper.vm.load();
    await tick();
    expect(wrapper.vm.dirty).to.equal(true);
    expect(wrapper.find('[data-testid="message-save"]').attributes('disabled')).to.equal(undefined);
  });
});
