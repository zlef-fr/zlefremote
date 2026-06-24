'use strict';
const { dict } = require('./i18n');

const DESC = {
  en: "ZlefRemote turns your phone into a wireless trackpad and keyboard for any computer — over local Wi-Fi or end-to-end encrypted from anywhere. A small portable agent for Linux, Windows and macOS; nothing to install on the phone.",
  fr: "ZlefRemote transforme votre téléphone en trackpad et clavier sans fil pour n'importe quel ordinateur — sur Wi-Fi local ou chiffré de bout en bout, où que vous soyez. Un petit agent portable pour Linux, Windows et macOS ; rien à installer sur le téléphone.",
};
const OGT = {
  en: "ZlefRemote — your phone is the trackpad",
  fr: "ZlefRemote — votre téléphone devient le trackpad",
};

const SEO = (lang) => `  <!--zlef-seo-->
  <meta name="description" content="${DESC[lang] || DESC.en}">
  <meta name="keywords" content="remote mouse, phone keyboard, control computer from phone, wireless trackpad, remote desktop input, end-to-end encrypted, Linux, Windows, macOS, télécommande PC, souris téléphone">
  <meta name="author" content="zlef.fr">
  <meta name="robots" content="index, follow">
  <link rel="canonical" href="https://remote.zlef.fr/">
  <meta name="theme-color" content="#06060a">
  <meta name="color-scheme" content="dark">
  <link rel="icon" href="https://assets.zlef.fr/favicon.svg" type="image/svg+xml">
  <link rel="icon" href="https://assets.zlef.fr/favicon.ico" sizes="any">
  <link rel="apple-touch-icon" href="https://assets.zlef.fr/apple-touch-icon.png">
  <link rel="manifest" href="https://assets.zlef.fr/site.webmanifest">
  <meta property="og:type" content="website">
  <meta property="og:site_name" content="ZLEF">
  <meta property="og:title" content="${OGT[lang] || OGT.en}">
  <meta property="og:description" content="${DESC[lang] || DESC.en}">
  <meta property="og:url" content="https://remote.zlef.fr/">
  <meta property="og:image" content="https://remote.zlef.fr/og.png?v=2">
  <meta property="og:image:width" content="1200">
  <meta property="og:image:height" content="630">
  <meta property="og:image:alt" content="ZlefRemote — a phone acting as a trackpad and keyboard, signalling wirelessly to a computer.">
  <meta property="og:locale" content="${lang === 'fr' ? 'fr_FR' : 'en_US'}">
  <meta name="twitter:card" content="summary_large_image">
  <meta name="twitter:title" content="${OGT[lang] || OGT.en}">
  <meta name="twitter:description" content="Remote mouse & keyboard control from your phone. LAN or end-to-end-encrypted relay. No phone app.">
  <meta name="twitter:image" content="https://remote.zlef.fr/og.png?v=2">
  <!--/zlef-seo-->`;

const JSONLD = (lang) => `<script type="application/ld+json">${JSON.stringify({
  '@context': 'https://schema.org',
  '@type': 'SoftwareApplication',
  name: 'ZlefRemote',
  applicationCategory: 'UtilitiesApplication',
  operatingSystem: 'Linux, Windows, macOS',
  description: DESC[lang] || DESC.en,
  url: 'https://remote.zlef.fr/',
  image: 'https://remote.zlef.fr/og.png?v=2',
  downloadUrl: 'https://remote.zlef.fr/#download',
  softwareVersion: '1.0.0',
  inLanguage: ['en', 'fr'],
  isAccessibleForFree: true,
  offers: { '@type': 'Offer', price: '0', priceCurrency: 'USD' },
  featureList: [
    'Wireless touch trackpad', 'Full keyboard & shortcuts', 'Media controls',
    'Local-network mode', 'End-to-end encrypted remote mode (AES-256-GCM)',
  ],
  author: { '@type': 'Organization', name: 'ZLEF', url: 'https://zlef.fr' },
})}</script>`;

// Minimal line-icon set (Lucide-style, stroke=currentColor) — used instead of
// emoji for a calmer, more intentional look that matches the design language.
const ICONS = {
  cursor: '<path d="M3 3l7.2 17.4 2.3-7 7-2.3z"/>',
  keyboard: '<rect x="2" y="5" width="20" height="14" rx="2"/><path d="M6 9h.01M10 9h.01M14 9h.01M18 9h.01M7 13h10"/>',
  sliders: '<line x1="21" x2="14" y1="6" y2="6"/><line x1="10" x2="3" y1="6" y2="6"/><line x1="21" x2="12" y1="12" y2="12"/><line x1="8" x2="3" y1="12" y2="12"/><line x1="21" x2="16" y1="18" y2="18"/><line x1="12" x2="3" y1="18" y2="18"/><circle cx="12" cy="6" r="2"/><circle cx="10" cy="12" r="2"/><circle cx="14" cy="18" r="2"/>',
  shield: '<path d="M12 22s8-3.5 8-9.5V5l-8-3-8 3v7.5C4 18.5 12 22 12 22z"/><path d="m9 12 2 2 4-4"/>',
  home: '<path d="M3 10.2 12 3l9 7.2V20a1 1 0 0 1-1 1h-5v-6H9v6H4a1 1 0 0 1-1-1z"/>',
  globe: '<circle cx="12" cy="12" r="9"/><path d="M3 12h18"/><path d="M12 3a14 14 0 0 1 0 18 14 14 0 0 1 0-18"/>',
  lock: '<rect x="4" y="10.5" width="16" height="10" rx="2"/><path d="M8 10.5V7a4 4 0 0 1 8 0v3.5"/>',
  phone: '<rect x="6" y="2" width="12" height="20" rx="2"/><path d="M11 18h2"/>',
  mail: '<rect x="2" y="4.5" width="20" height="15" rx="2"/><path d="m3 6 9 6 9-6"/>',
  send: '<path d="M21.5 2.5 11 13"/><path d="M21.5 2.5 15 21l-4-8-8-4z"/>',
  share: '<circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><path d="m8.6 13.5 6.8 4M15.4 6.5l-6.8 4"/>',
  link: '<path d="M10 13a5 5 0 0 0 7.5.5l3-3A5 5 0 0 0 13.5 3.5L12 5"/><path d="M14 11a5 5 0 0 0-7.5-.5l-3 3A5 5 0 0 0 10.5 20.5L12 19"/>',
  download: '<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><path d="m7 10 5 5 5-5"/><path d="M12 15V3"/>',
  arrow: '<path d="M5 12h14"/><path d="m12 5 7 7-7 7"/>',
  vol: '<path d="M11 5 6 9H3v6h3l5 4z"/><path d="M15.5 8.5a5 5 0 0 1 0 7"/>',
  mute: '<path d="M11 5 6 9H3v6h3l5 4z"/><path d="m17 9 4 6M21 9l-4 6"/>',
  play: '<path d="M6 4l14 8-14 8z"/>',
  prev: '<path d="M19 5 9 12l10 7zM5 5v14"/>',
  next: '<path d="M5 5l10 7L5 19zM19 5v14"/>',
};
function icon(name, cls) {
  return `<svg class="ico${cls ? ' ' + cls : ''}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${ICONS[name] || ''}</svg>`;
}

// which agent binaries actually exist in dist/ (so we don't advertise fakes)
function platformLinks(have) {
  const item = (key, label, file, sub) => {
    const ready = have.has(file);
    return { key, label, file, sub, ready };
  };
  return [
    item('linux', 'Linux', 'zlefremote-agent-linux-amd64', 'x86-64'),
    item('win',   'Windows', 'zlefremote-agent-windows-amd64.exe', 'x86-64'),
    item('mac',   'macOS', 'zlefremote-agent-darwin-arm64', 'Apple Silicon'),
  ];
}

function landing(lang, have) {
  const t = dict(lang);
  const plats = platformLinks(have);
  const dlCards = plats.map(p => {
    if (p.ready) {
      return `<a class="dlcard ready" href="/download/${p.file}" download>
        <span class="dlname">${p.label}</span>
        <span class="dlsub">${p.sub}</span>
        <span class="dlget">${icon('download')} ${t['dl_' + (p.key === 'win' ? 'win' : p.key === 'mac' ? 'mac' : 'linux')]}</span>
      </a>`;
    }
    return `<a class="dlcard soon" href="https://github.com/zlef-fr/zlefremote#build-from-source">
        <span class="dlname">${p.label}</span>
        <span class="dlsub">${p.sub}</span>
        <span class="dlget muted">${t.dl_source} ${icon('arrow')}</span>
      </a>`;
  }).join('\n');

  return `<!doctype html>
<html lang="${lang}">
<head>
<meta charset="utf-8">
${SEO(lang)}
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${OGT[lang] || OGT.en}</title>
${JSONLD(lang)}
<link rel="stylesheet" href="https://da.zlef.fr/tokens.css">
<link rel="stylesheet" href="/css/site.css?v=8">
</head>
<body>
<script defer src="https://assets.zlef.fr/track.js"></script>
<div class="bg-field" id="bg"></div>

<header class="nav">
  <a class="brand" href="/">
    <span class="brand-mark">◑</span>
    <span class="brand-name">Zlef<b>Remote</b></span>
  </a>
  <nav class="nav-links">
    <a href="#features">${t.nav_features}</a>
    <a href="#download">${t.nav_download}</a>
    <a class="zl-btn zl-btn--sm zl-btn--ghost" href="https://github.com/zlef-fr/zlefremote" target="_blank" rel="noopener">GitHub</a>
  </nav>
</header>

<main>
  <section class="hero">
    <div class="hero-copy reveal">
      <div class="kicker">${t.badge_e2ee} · ${t.badge_cross}</div>
      <h1 class="zl-h1 hero-title">${t.tagline}</h1>
      <p class="lede">${t.lede}</p>
      <div class="hero-cta">
        <a class="zl-btn zl-btn--primary zl-btn--lg cta-desktop" href="#download">${icon('download')} ${t.cta_download}</a>
        <a class="zl-btn zl-btn--primary zl-btn--lg cta-mobile" href="/start">${t.cta_start} ${icon('arrow')}</a>
        <a class="zl-btn zl-btn--ghost zl-btn--lg" href="#how">${t.cta_how}</a>
      </div>
      <div class="badges">
        <span class="badge">${icon('lock')} ${t.badge_e2ee}</span>
        <span class="badge">${icon('phone')} ${t.badge_noinstall}</span>
      </div>
    </div>
    <div class="hero-art reveal" aria-hidden="true">
      <div class="phone">
        <div class="phone-screen">
          <div class="pad"><span class="cursor"></span></div>
          <div class="keys">
            <i></i><i></i><i></i><i></i><i></i><i></i><i></i><i></i>
            <i></i><i></i><i></i><i></i><i></i><i></i><i></i><i></i>
          </div>
        </div>
      </div>
      <div class="signal" aria-hidden="true">
        <span class="emitter"></span>
        <span class="wave w1"></span>
        <span class="wave w2"></span>
        <span class="wave w3"></span>
        <span class="wave w4"></span>
      </div>
      <div class="monitor"><div class="monitor-screen"></div><div class="monitor-stand"></div></div>
    </div>
  </section>

  <section id="how" class="how">
    <h2 class="zl-h2 sect-title reveal">${t.how_title}</h2>
    <div class="steps">
      <div class="step reveal"><div class="step-no">①</div><h3>${t.how_1_t}</h3><p>${t.how_1_b}</p></div>
      <div class="step reveal"><div class="step-no">②</div><h3>${t.how_2_t}</h3><p>${t.how_2_b}</p></div>
      <div class="step reveal"><div class="step-no">③</div><h3>${t.how_3_t}</h3><p>${t.how_3_b}</p></div>
    </div>
  </section>

  <section class="modes">
    <h2 class="zl-h2 sect-title reveal">${t.modes_title}</h2>
    <div class="mode-grid">
      <div class="mode-card reveal">
        <div class="mode-ico">${icon('home')}</div>
        <h3>${t.mode_lan_t}</h3>
        <p>${t.mode_lan_b}</p>
      </div>
      <div class="mode-card reveal accent2">
        <div class="mode-ico">${icon('globe')}</div>
        <h3>${t.mode_remote_t}</h3>
        <p>${t.mode_remote_b}</p>
      </div>
    </div>
    <div class="e2ee reveal">
      <h3>${icon('lock')} ${t.e2ee_title}</h3>
      <p>${t.e2ee_b}</p>
    </div>
  </section>

  <section id="features" class="features">
    <h2 class="zl-h2 sect-title reveal">${t.feat_title}</h2>
    <div class="feat-grid">
      <div class="feat reveal"><h3>${icon('cursor')} ${t.feat_1_t}</h3><p>${t.feat_1_b}</p></div>
      <div class="feat reveal"><h3>${icon('keyboard')} ${t.feat_2_t}</h3><p>${t.feat_2_b}</p></div>
      <div class="feat reveal"><h3>${icon('sliders')} ${t.feat_3_t}</h3><p>${t.feat_3_b}</p></div>
      <div class="feat reveal"><h3>${icon('shield')} ${t.feat_4_t}</h3><p>${t.feat_4_b}</p></div>
    </div>
  </section>

  <section id="download" class="download">
    <h2 class="zl-h2 sect-title reveal">${t.dl_title}</h2>
    <p class="dl-lede reveal">${t.dl_b}</p>
    <div class="dl-grid reveal">
      ${dlCards}
    </div>
    <p class="dl-note reveal">${t.dl_note}</p>
  </section>
</main>

<footer class="foot">
  <span>${t.footer_made}</span>
  <a href="https://zlef.fr">zlef.fr</a>
</footer>

<script src="/js/landing.js?v=2" defer></script>
</body>
</html>`;
}

// A non-interactive mockup of the in-room remote (the trackpad view), so a
// visitor on their phone can see exactly what they'll be holding.
function previewPhone(t) {
  return `<div class="pv-phone" aria-hidden="true">
    <div class="pv-screen">
      <div class="pv-top">
        <span class="pv-dot"></span><span class="pv-st">${t.pv_status}</span>
        <span class="pv-host">${t.pv_host}</span>
      </div>
      <div class="pv-pad"><span class="pv-cursor"></span><span class="pv-hint">${t.pv_hint}</span></div>
      <div class="pv-btns"><span>${t.pv_left}</span><span>${t.pv_right}</span><span class="pv-lock">${t.pv_drag}</span></div>
      <div class="pv-tabs">
        <span class="on">${icon('cursor')}${t.pv_pad}</span>
        <span>${icon('keyboard')}${t.pv_keys}</span>
        <span>${icon('sliders')}${t.pv_media}</span>
      </div>
    </div>
  </div>`;
}

function startPage(lang) {
  const t = dict(lang);
  const DL = 'https://remote.zlef.fr/#download';
  const msg = lang === 'fr'
    ? 'Installez ZlefRemote sur votre ordinateur pour utiliser votre téléphone comme trackpad et clavier :'
    : 'Install ZlefRemote on your computer to use your phone as its trackpad and keyboard:';
  const mailto = `mailto:?subject=${encodeURIComponent('ZlefRemote — desktop agent')}&body=${encodeURIComponent(msg + '\n\n' + DL)}`;
  const tg = `https://t.me/share/url?url=${encodeURIComponent(DL)}&text=${encodeURIComponent(msg)}`;

  return `<!doctype html>
<html lang="${lang}">
<head>
<meta charset="utf-8">
<meta name="robots" content="noindex, follow">
<meta name="description" content="${t.start_sub}">
<link rel="canonical" href="https://remote.zlef.fr/start">
<meta name="theme-color" content="#06060a">
<link rel="icon" href="https://assets.zlef.fr/favicon.svg" type="image/svg+xml">
<link rel="apple-touch-icon" href="https://assets.zlef.fr/apple-touch-icon.png">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${t.start_title} — ZlefRemote</title>
<link rel="stylesheet" href="https://da.zlef.fr/tokens.css">
<link rel="stylesheet" href="/css/site.css?v=8">
</head>
<body class="start">
<script defer src="https://assets.zlef.fr/track.js"></script>
<div class="bg-field"></div>

<header class="nav">
  <a class="brand" href="/"><span class="brand-mark">◑</span><span class="brand-name">Zlef<b>Remote</b></span></a>
  <nav class="nav-links">
    <a class="zl-btn zl-btn--sm zl-btn--ghost" href="/">${icon('arrow', 'flip')} ${t.start_back}</a>
  </nav>
</header>

<main class="start-main">
  <section class="start-hero">
    <div class="start-preview">
      <div class="pv-label">${t.start_preview}</div>
      ${previewPhone(t)}
    </div>
    <div class="start-copy">
      <h1 class="zl-h1 start-h1">${t.start_title}</h1>
      <p class="start-sub">${t.start_sub}</p>

      <div class="getbox">
        <h2 class="getbox-title">${t.start_get_title}</h2>
        <p class="getbox-sub">${t.start_get_sub}</p>
        <div class="getbtns">
          <a class="zl-btn zl-btn--primary getbtn" href="${mailto}">${icon('mail')} ${t.start_email}</a>
          <a class="zl-btn zl-btn--secondary getbtn" href="${tg}" target="_blank" rel="noopener">${icon('send')} ${t.start_telegram}</a>
          <button class="zl-btn zl-btn--ghost getbtn" id="shareBtn" hidden data-msg="${msg}" data-url="${DL}">${icon('share')} ${t.start_share}</button>
          <button class="zl-btn zl-btn--ghost getbtn" id="copyBtn" data-url="${DL}" data-copied="${t.start_copied}">${icon('link')} <span>${t.start_copy}</span></button>
        </div>
        <p class="getbox-desktop">${t.start_ondesktop} <a href="/#download">${t.start_download_now} ${icon('arrow')}</a></p>
      </div>
    </div>
  </section>
</main>

<footer class="foot">
  <span>${t.footer_made}</span>
  <a href="https://zlef.fr">zlef.fr</a>
</footer>
<script src="/js/start.js?v=1" defer></script>
</body>
</html>`;
}

module.exports = { landing, startPage, SEO, platformLinks };
