import { expect } from 'chai';
import { mount } from '@vue/test-utils';
import axios from 'axios';
import Vuetify from 'vuetify';
import EventBus from '@/event-bus';
import ServerSettings from '@/views/ServerSettings.vue';

const settingsFor = (enabled, saved = '') => ({
  use_remote_runner: enabled,
  appearance_theme: 'texec',
  schedule_timezone: {
    value: saved || 'UTC', saved, source: saved ? 'database' : 'system', system: 'UTC',
  },
});

const tick = () => new Promise((resolve) => { setTimeout(resolve, 0); });

describe('Server settings', () => {
  let wrapper;
  let originalGet;
  let originalPut;
  let saved;
  let updates;
  const onUpdate = (settings) => { updates.push(settings); };
  const render = () => {
    wrapper = mount(ServerSettings, {
      vuetify: new Vuetify(), mocks: { $t: (key) => key }, stubs: ['router-link'],
    });
  };

  beforeEach(() => {
    originalGet = axios.get;
    originalPut = axios.put;
    saved = [];
    updates = [];
    axios.get = async (url) => {
      expect(url).to.equal('/api/settings');
      return { data: settingsFor(false) };
    };
    axios.put = async (url, data) => {
      saved.push({ url, data });
      return { data: settingsFor(data.use_remote_runner, data.schedule_timezone) };
    };
    EventBus.$on('i-server-settings', onUpdate);
  });

  afterEach(() => {
    if (wrapper) wrapper.destroy();
    axios.get = originalGet;
    axios.put = originalPut;
    EventBus.$off('i-server-settings', onUpdate);
  });

  it('loads the server value and saves both enabling and disabling', async () => {
    render();
    await tick();
    expect(wrapper.find('[data-testid="server-settings-save"]').attributes('disabled'))
      .to.equal('disabled');
    const setAndSave = async (enabled) => {
      await wrapper.find('[data-testid="server-remote-runners"]').setChecked(enabled);
      expect(wrapper.vm.dirty).to.equal(true);
      await wrapper.find('form').trigger('submit');
      await tick();
      expect(saved[saved.length - 1]).to.deep.equal({
        url: '/api/settings', data: { use_remote_runner: enabled, schedule_timezone: '', appearance_theme: 'texec' },
      });
      expect(updates[updates.length - 1]).to.deep.equal(settingsFor(enabled));
      expect(wrapper.vm.dirty).to.equal(false);
    };
    await setAndSave(true);
    await setAndSave(false);
  });

  it('saves the global theme and updates the application only after success', async () => {
    render();
    await tick();
    axios.put = async (url, data) => {
      saved.push({ url, data });
      return { data: { ...settingsFor(false), appearance_theme: data.appearance_theme } };
    };
    await wrapper.find('[data-testid="theme-dracula"] input').setChecked();
    expect(updates).to.have.length(1);
    await wrapper.find('form').trigger('submit');
    await tick();
    expect(saved[0].data.appearance_theme).to.equal('dracula');
    expect(updates[1].appearance_theme).to.equal('dracula');
    expect(wrapper.vm.dirty).to.equal(false);
  });

  it('offers retry without showing editable defaults when loading fails', async () => {
    axios.get = async () => { throw new Error('Load failed'); };
    render();
    expect(wrapper.find('[data-testid="server-remote-runners"]').exists()).to.equal(false);
    await tick();
    expect(wrapper.find('[data-testid="server-settings-error"]').text()).to.equal('Load failed');
    expect(wrapper.find('[data-testid="server-remote-runners"]').exists()).to.equal(false);
    axios.get = async () => ({ data: settingsFor(true) });
    await wrapper.find('[data-testid="server-settings-retry"]').trigger('click');
    await tick();
    expect(wrapper.vm.useRemoteRunner).to.equal(true);
    expect(wrapper.vm.dirty).to.equal(false);
  });

  it('keeps an unsaved change after a failed write and leaves the app state unchanged', async () => {
    render();
    await tick();
    axios.put = async () => { throw new Error('Save failed'); };
    await wrapper.find('[data-testid="server-remote-runners"]').setChecked(true);
    await wrapper.find('form').trigger('submit');
    await tick();
    expect(wrapper.find('[data-testid="server-settings-error"]').text()).to.equal('Save failed');
    expect(wrapper.vm.useRemoteRunner).to.equal(true);
    expect(wrapper.vm.dirty).to.equal(true);
    expect(updates).to.deep.equal([settingsFor(false)]);
  });

  it('locks editing and duplicate submissions while saving', async () => {
    render();
    await tick();
    let finishSave;
    axios.put = (url, data) => {
      saved.push({ url, data });
      return new Promise((resolve) => {
        finishSave = () => resolve({
          data: settingsFor(data.use_remote_runner, data.schedule_timezone),
        });
      });
    };
    await wrapper.find('[data-testid="server-remote-runners"]').setChecked(true);
    await wrapper.find('form').trigger('submit');
    expect(wrapper.find('[data-testid="server-remote-runners"]').attributes('disabled'))
      .to.equal('disabled');
    await wrapper.find('form').trigger('submit');
    expect(saved).to.have.length(1);
    finishSave();
    await tick();
    expect(wrapper.vm.dirty).to.equal(false);
  });
  it('saves a timezone and can reset to the system default', async () => {
    render();
    await tick();
    await wrapper.setData({ scheduleTimezone: 'Europe/Moscow' });
    await wrapper.find('form').trigger('submit');
    await tick();
    expect(saved[0].data.schedule_timezone).to.equal('Europe/Moscow');
    expect(wrapper.vm.dirty).to.equal(false);
    expect(wrapper.vm.settings.schedule_timezone.value).to.equal('Europe/Moscow');
    await wrapper.setData({ scheduleTimezone: null });
    await wrapper.find('form').trigger('submit');
    await tick();
    expect(saved[1].data.schedule_timezone).to.equal('');
    expect(wrapper.vm.settings.schedule_timezone.source).to.equal('system');
    expect(wrapper.vm.dirty).to.equal(false);
  });

  ['environment', 'configuration'].forEach((source) => {
    it(`shows and locks the ${source} timezone without overwriting the database preference`, async () => {
      const settings = settingsFor(false, 'America/New_York');
      settings.schedule_timezone.value = 'Europe/Moscow';
      settings.schedule_timezone.source = source;
      axios.get = async () => ({ data: settings });
      axios.put = async (url, data) => {
        saved.push({ url, data });
        return { data: { ...settings, use_remote_runner: data.use_remote_runner } };
      };
      render();
      await tick();
      expect(wrapper.vm.scheduleTimezone).to.equal('Europe/Moscow');
      expect(wrapper.find('[data-testid="server-schedule-timezone"]').attributes('disabled'))
        .to.equal('disabled');
      expect(wrapper.find('[data-testid="schedule-timezone-locked"]').text())
        .to.equal(source === 'environment'
          ? 'scheduleTimezoneEnvironmentOverride' : 'scheduleTimezoneConfigurationOverride');
      await wrapper.find('[data-testid="server-remote-runners"]').setChecked(true);
      await wrapper.find('form').trigger('submit');
      await tick();
      expect(saved[0].data).to.deep.equal({ use_remote_runner: true, appearance_theme: 'texec' });
      expect(wrapper.vm.settings.schedule_timezone.saved).to.equal('America/New_York');
    });
  });
});
