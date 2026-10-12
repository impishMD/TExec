import { expect } from 'chai';
import { mount } from '@vue/test-utils';
import Vuetify from 'vuetify';
import axios from 'axios';
import EventBus from '@/event-bus';
import Appearance from '@/views/Appearance.vue';
import ThemePicker from '@/components/ThemePicker.vue';
import {
  THEMES,
  themePalette,
  contrast,
  resolveTheme,
  personalThemeFromOptions,
  applyTheme,
} from '@/lib/appearance';

const tick = () => new Promise((resolve) => {
  setTimeout(resolve, 0);
});

describe('Appearance palettes', () => {
  THEMES.forEach((theme) => [false, true].forEach((dark) => {
    it(`${theme.id} ${dark ? 'dark' : 'light'} keeps text and semantic colors readable`, () => {
      const p = themePalette(theme.id, dark);
      const minimum = theme.accessible ? 7 : 4.5;
      [p.bg, p.surface, p.soft, p.accentSoft].forEach((background) => {
        ['text', 'muted', 'accent', 'success', 'error', 'warning', 'info'].forEach((name) => {
          expect(contrast(p[name], background), `${name} on ${background}`).to.be.at.least(
            minimum,
          );
        });
        if (theme.accessible) expect(contrast(p.border, background)).to.be.at.least(3);
      });
      expect(contrast(p.primaryText, p.accent)).to.be.at.least(4.5);
    });
  }));

  it('inherits the global choice, lets users override it and recovers unknown stored values', () => {
    expect(resolveTheme('', 'nord')).to.equal('nord');
    expect(resolveTheme('ocean', 'nord')).to.equal('ocean');
    expect(resolveTheme('', 'unknown')).to.equal('texec');
    ['null', '{}', '42', '"future-theme"', 'invalid'].forEach((value) => {
      expect(personalThemeFromOptions({ 'appearance.theme': value })).to.equal('');
    });
    expect(personalThemeFromOptions({ 'appearance.theme': '"dracula"' })).to.equal('dracula');
    expect(personalThemeFromOptions({})).to.equal('');
  });

  it('changes both palettes without changing the selected light/dark mode', () => {
    const vuetify = new Vuetify().framework;
    vuetify.theme.dark = true;
    applyTheme(vuetify, 'blue-orange');
    expect(vuetify.theme.dark).to.equal(true);
    expect(vuetify.theme.themes.light.primary).to.equal(themePalette('blue-orange', false).accent);
    expect(vuetify.theme.themes.dark.primary).to.equal(themePalette('blue-orange', true).accent);
  });
});

describe('Personal appearance', () => {
  let wrapper;
  let originalGet;
  let originalPost;
  let saved;
  let emitted;
  const onChange = (id) => emitted.push(id);
  const render = () => mount(Appearance, {
    vuetify: new Vuetify(),
    mocks: { $t: (key) => key },
    propsData: { systemInfo: { appearance_theme: 'nord' } },
  });
  beforeEach(() => {
    originalGet = axios.get;
    originalPost = axios.post;
    saved = [];
    emitted = [];
    axios.get = async () => ({ data: {} });
    axios.post = async (url, data) => {
      saved.push({ url, data });
    };
    EventBus.$on('i-appearance', onChange);
  });
  afterEach(() => {
    if (wrapper) wrapper.destroy();
    axios.get = originalGet;
    axios.post = originalPost;
    EventBus.$off('i-appearance', onChange);
  });

  it('offers inherited defaults and saves an explicit choice only on submit', async () => {
    wrapper = render();
    await tick();
    expect(wrapper.find('[data-testid="theme-default"]').element.checked).to.equal(true);
    expect(wrapper.find('[data-testid="theme-preview"]').attributes('data-theme')).to.equal('nord');
    await wrapper.find('[data-testid="theme-ocean"] input').setChecked();
    expect(saved).to.have.length(0);
    expect(emitted).to.have.length(0);
    await wrapper.find('form').trigger('submit');
    await tick();
    expect(saved).to.deep.equal([
      { url: '/api/user/options', data: { key: 'appearance.theme', value: '"ocean"' } },
    ]);
    expect(emitted).to.deep.equal(['ocean']);
    expect(wrapper.vm.dirty).to.equal(false);
  });

  it('restores a saved theme and can return to the global default', async () => {
    axios.get = async () => ({ data: { 'appearance.theme': '"amber"' } });
    wrapper = render();
    await tick();
    expect(wrapper.vm.selected).to.equal('amber');
    await wrapper.find('[data-testid="theme-default"]').setChecked();
    await wrapper.find('form').trigger('submit');
    await tick();
    expect(saved[0].data.value).to.equal('""');
    expect(emitted).to.deep.equal(['']);
  });

  it('does not apply an unsaved or failed preference and prevents duplicate submissions', async () => {
    wrapper = render();
    await tick();
    let reject;
    axios.post = (url, data) => {
      saved.push(data);
      return new Promise((resolve, rejectSave) => {
        reject = rejectSave;
      });
    };
    await wrapper.setData({ selected: 'nord' });
    wrapper.vm.save();
    await tick();
    expect(wrapper.find('fieldset').attributes('disabled')).to.equal('disabled');
    await wrapper.vm.save();
    expect(saved).to.have.length(1);
    reject(new Error('Save failed'));
    await tick();
    expect(wrapper.vm.error).to.equal('Save failed');
    expect(wrapper.vm.dirty).to.equal(true);
    expect(emitted).to.have.length(0);
  });

  it('does not offer a save when preferences could not be loaded', async () => {
    axios.get = async () => {
      throw new Error('Load failed');
    };
    wrapper = render();
    await tick();
    expect(wrapper.findComponent(ThemePicker).exists()).to.equal(false);
    expect(wrapper.text()).to.include('Load failed');
  });
});
