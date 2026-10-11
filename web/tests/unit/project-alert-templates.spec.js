import { expect } from 'chai';
import { mount } from '@vue/test-utils';
import axios from 'axios';
import Vuetify from 'vuetify';
import VueI18n from 'vue-i18n';
import ProjectAlertTemplates from '@/components/ProjectAlertTemplates.vue';
import defaultMessages from '@/lib/taskNotifications';

const tick = () => new Promise((resolve) => { setTimeout(resolve, 0); });
const settings = {
  success: 'Мой успех {{task.id}}',
  error: 'Моя ошибка {{task.error}}',
  success_locale: 'ru',
  error_locale: 'ru',
  timezone: 'UTC',
};
const defaults = {
  ru: { ...settings, success: 'Успешно', error: 'Ошибка' },
  ja: {
    ...settings, success: '成功', error: 'エラー', success_locale: 'ja', error_locale: 'ja',
  },
  zh_cn: {
    ...settings, success: '成功', error: '错误', success_locale: 'zh_cn', error_locale: 'zh_cn',
  },
};

describe('Project message templates', () => {
  let wrapper;
  let originalPost;
  let sent;
  beforeEach(() => {
    originalPost = axios.post;
    sent = [];
    axios.post = async (url, data) => {
      sent.push({ url, data }); return { data: { text: '<b>example</b>' } };
    };
  });
  afterEach(() => { wrapper?.destroy(); axios.post = originalPost; });
  async function render(extra = {}) {
    wrapper = mount(ProjectAlertTemplates, {
      propsData: {
        projectId: 1,
        value: { ...settings },
        defaults,
        variables: ['task.id', 'task.error', 'timezone'],
        ...extra,
      },
      vuetify: new Vuetify(),
      i18n: new VueI18n({ locale: 'ja' }),
      mocks: { $t: (key) => key },
    });
    await tick();
  }

  it('keeps Russian text in a Japanese interface and resets only the selected text', async () => {
    await render();
    expect(wrapper.vm.value).to.deep.equal(settings);
    expect(wrapper.emitted('input')).to.equal(undefined);
    wrapper.vm.resetTemplate();
    const changed = wrapper.emitted('input')[0][0];
    expect(changed.success).to.equal('成功');
    expect(changed.success_locale).to.equal('ja');
    expect(changed.error).to.equal(settings.error);
    expect(changed.error_locale).to.equal('ru');
    expect(changed.timezone).to.equal('UTC');
    expect(defaultMessages(defaults, 'zh-CN')).to.deep.equal(defaults.zh_cn);
    expect(defaultMessages(defaults, 'ja-JP')).to.deep.equal(defaults.ja);
  });

  it('disables editing, reset and previews while the channel is off', async () => {
    await render({ disabled: true });
    expect(wrapper.find('textarea').element.disabled).to.equal(true);
    expect(wrapper.find('[data-testid="message-reset"]').attributes('disabled')).to.equal('disabled');
    wrapper.vm.resetTemplate();
    wrapper.vm.patch({ success: 'changed' });
    expect(wrapper.emitted('input')).to.equal(undefined);
    expect(wrapper.vm.previewLoading).to.equal(false);
    expect(sent).to.have.length(0);
    await wrapper.setProps({ disabled: false });
    expect(wrapper.find('textarea').element.disabled).to.equal(false);
    expect(wrapper.vm.previewLoading).to.equal(true);
  });

  it('previews plain text with no separate save or send controls', async () => {
    await render();
    await wrapper.vm.loadPreview(wrapper.vm.sequence);
    await tick();
    expect(wrapper.find('[data-testid="message-preview"]').text()).to.equal('<b>example</b>');
    expect(wrapper.find('[data-testid="message-preview"] b').exists()).to.equal(false);
    expect(sent[0].url).to.equal('/api/project/1/alerts/telegram/templates/preview');
    expect(sent[0].data.templates).to.deep.equal(settings);
    expect(wrapper.find('[data-testid="message-test"]').exists()).to.equal(false);
    expect(wrapper.find('[data-testid="message-save"]').exists()).to.equal(false);
    await wrapper.setProps({ value: { ...settings, error: '{{secret}}' } });
    expect(wrapper.vm.otherProblem).to.equal('workflowNotificationVariableInvalid');
    expect(wrapper.vm.previewLoading).to.equal(false);
  });

  it('ignores late previews after switching tabs or disabling the channel', async () => {
    await render();
    const pending = [];
    axios.post = () => new Promise((resolve) => { pending.push(resolve); });
    const first = wrapper.vm.loadPreview(wrapper.vm.sequence);
    await wrapper.setProps({ status: 'error' });
    const second = wrapper.vm.loadPreview(wrapper.vm.sequence);
    pending[1]({ data: { text: 'new' } });
    await second;
    pending[0]({ data: { text: 'old' } });
    await first;
    expect(wrapper.vm.preview).to.equal('new');
    const third = wrapper.vm.loadPreview(wrapper.vm.sequence);
    await wrapper.setProps({ disabled: true });
    pending[2]({ data: { text: 'disabled' } });
    await third;
    expect(wrapper.vm.preview).to.equal('');
  });
});
