import { themePalette } from '@/lib/appearance';

export function themeFavicon(id, dark) {
  const { bg, accent, text } = themePalette(id, dark);
  // A themed background keeps the mark readable when browser chrome and app modes differ.
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 440 440" width="64" height="64">
<title>TExec</title>
<rect width="440" height="440" rx="96" fill="${bg}"/>
<g transform="translate(17 82)" fill="none" stroke-linecap="round">
<path d="M 103,88 H 193" stroke="${accent}" stroke-width="23"/>
<path d="M 35,140 H 193" stroke="${text}" stroke-width="23"/>
<path d="M 35,128.5 H 148 V 151.5 H 35 A 11.5,11.5 0 0 1 35,128.5 Z" fill="${accent}"/>
<path d="M 103,192 H 193" stroke="${accent}" stroke-width="23"/>
<path d="M 246,20 V 260" stroke="${text}" stroke-width="24"/>
<path d="M 298,65 Q 298,61 302,64 L 391,132 Q 401,140 391,148 L 302,216 Q 298,219 298,215 Z" fill="${accent}"/>
</g></svg>`;
  return `data:image/svg+xml,${encodeURIComponent(svg)}`;
}

export function applyFavicon(href) {
  let link = document.getElementById('texec-favicon');
  if (!link) {
    link = document.querySelector('link[rel="icon"]') || document.createElement('link');
    link.id = 'texec-favicon';
    link.rel = 'icon';
    link.type = 'image/svg+xml';
    if (!link.isConnected) document.head.appendChild(link);
  }
  if (link.getAttribute('href') !== href) link.setAttribute('href', href);
}
