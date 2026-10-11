export default function defaultMessages(defaults, locale) {
  const normalized = (locale || 'en').toLowerCase().replace(/-/g, '_');
  return defaults[normalized] || defaults[normalized.split('_')[0]] || defaults.en;
}
