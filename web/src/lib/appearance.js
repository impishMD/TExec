import catalog from '../../../util/appearance/themes.json';

export const DEFAULT_THEME = 'texec';
export const THEMES = catalog;
export const themeById = (id) => THEMES.find((theme) => theme.id === id) || THEMES[0];
export const resolveTheme = (personal, global) => themeById(personal || global).id;

export function personalThemeFromOptions(options) {
  try {
    const id = JSON.parse(options['appearance.theme']);
    return THEMES.some((theme) => theme.id === id) ? id : '';
  } catch {
    return '';
  }
}

const rgb = (hex) => hex
  .replace('#', '')
  .match(/.{2}/g)
  .map((channel) => parseInt(channel, 16));
const hex = (channels) => {
  const values = channels.map((channel) => Math.round(channel).toString(16).padStart(2, '0'));
  return `#${values.join('')}`;
};
const mix = (a, b, amount) => {
  const target = rgb(b);
  const channels = rgb(a).map((channel, i) => channel + (target[i] - channel) * amount);
  return hex(channels);
};
const luminance = (color) => rgb(color)
  .map((channel) => {
    const value = channel / 255;
    return value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4;
  })
  .reduce((sum, value, i) => sum + value * [0.2126, 0.7152, 0.0722][i], 0);

export function contrast(a, b) {
  const [lighter, darker] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (lighter + 0.05) / (darker + 0.05);
}

export function contrastingText(background) {
  return contrast('#ffffff', background) >= contrast('#000000', background) ? '#ffffff' : '#000000';
}

// Adapt source palette accents for readable UI text, including tinted surfaces.
function readable(color, backgrounds, minimum) {
  const score = (candidate) => Math.min(...backgrounds.map((bg) => contrast(candidate, bg)));
  if (score(color) >= minimum) return color;
  const target = score('#ffffff') > score('#000000') ? '#ffffff' : '#000000';
  for (let step = 1; step <= 100; step += 1) {
    const adjusted = mix(color, target, step / 100);
    if (score(adjusted) >= minimum) return adjusted;
  }
  return target;
}

const palettes = new Map();
export function themePalette(id, dark = false) {
  const theme = themeById(id);
  const key = `${theme.id}:${dark}`;
  if (palettes.has(key)) return palettes.get(key);
  const base = theme[dark ? 'dark' : 'light'];
  const accentSoft = base.accentSoft || mix(base.surface, base.accent, dark ? 0.12 : 0.09);
  const backgrounds = [base.bg, base.surface, base.soft, accentSoft];
  const minimum = theme.accessible ? 7 : 4.5;
  const accent = readable(base.accent, backgrounds, minimum);
  const status = theme.id === 'blue-orange'
    ? { success: dark ? '#8fcaff' : '#005a94', error: dark ? '#ffc18a' : '#993e00' }
    : { success: dark ? '#83d9bb' : '#218263', error: dark ? '#f1919d' : '#c94b5b' };
  const palette = {
    ...base,
    accent,
    accentSoft,
    text: readable(base.text, backgrounds, minimum),
    muted: readable(base.muted, backgrounds, minimum),
    border: theme.accessible
      ? readable(base.muted, backgrounds, 3)
      : base.border || mix(base.surface, base.text, dark ? 0.2 : 0.18),
    primaryText: contrastingText(accent),
    shadow: theme.accessible ? 'none' : base.shadow || `0 8px 28px rgba(0, 0, 0, ${dark ? '0.16' : '0.05'})`,
  };
  Object.entries({
    ...status,
    warning: dark ? '#ebbc73' : '#a56a17',
    info: dark ? '#8cbde6' : '#387baf',
  }).forEach(([name, color]) => {
    palette[name] = readable(color, backgrounds, minimum);
  });
  palettes.set(key, palette);
  return palette;
}

export function themeVariables(id, dark) {
  const p = themePalette(id, dark);
  return {
    '--texec-bg': p.bg,
    '--texec-surface': p.surface,
    '--texec-soft': p.soft,
    '--texec-text': p.text,
    '--texec-muted': p.muted,
    '--texec-border': p.border,
    '--texec-accent': p.accent,
    '--texec-accent-soft': p.accentSoft,
    '--texec-on-primary': p.primaryText,
    '--texec-shadow': p.shadow,
    '--highlighted-card-bg-color': p.soft,
    '--texec-nav-bg': p.surface,
    '--texec-nav-text': p.text,
    '--texec-nav-muted': p.muted,
    '--texec-nav-border': p.border,
    '--texec-nav-surface': p.soft,
    '--texec-nav-active': p.accentSoft,
    '--texec-nav-active-text': p.accent,
    ...Object.fromEntries(Object.entries(p.nav || {}).map(([key, value]) => [
      `--texec-nav-${key}`, value,
    ])),
    ...Object.fromEntries(
      ['success', 'error', 'warning', 'info'].flatMap((name) => [
        [`--texec-${name}`, p[name]],
        [`--texec-on-${name}`, contrastingText(p[name])],
      ]),
    ),
  };
}

export function vuetifyThemes(id) {
  return Object.fromEntries(
    ['light', 'dark'].map((mode) => {
      const p = themePalette(id, mode === 'dark');
      return [
        mode,
        {
          primary: p.accent,
          secondary: p.muted,
          accent: p.accent,
          success: p.success,
          error: p.error,
          warning: p.warning,
          info: p.info,
          // Also expose surface colors to chart components through Vuetify's reactive theme.
          surface: p.surface,
          background: p.bg,
          text: p.text,
          muted: p.muted,
          border: p.border,
        },
      ];
    }),
  );
}

export function applyTheme(vuetify, id) {
  const modes = vuetifyThemes(id);
  Object.keys(modes).forEach((mode) => Object.assign(vuetify.theme.themes[mode], modes[mode]));
}
