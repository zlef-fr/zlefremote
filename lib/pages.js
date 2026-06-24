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
  <meta property="og:image" content="https://remote.zlef.fr/og.png">
  <meta property="og:image:width" content="1200">
  <meta property="og:image:height" content="630">
  <meta property="og:image:alt" content="ZlefRemote — a phone acting as a trackpad and keyboard, signalling wirelessly to a computer.">
  <meta property="og:locale" content="${lang === 'fr' ? 'fr_FR' : 'en_US'}">
  <meta name="twitter:card" content="summary_large_image">
  <meta name="twitter:title" content="${OGT[lang] || OGT.en}">
  <meta name="twitter:description" content="Remote mouse & keyboard control from your phone. LAN or end-to-end-encrypted relay. No phone app.">
  <meta name="twitter:image" content="https://remote.zlef.fr/og.png">
  <!--/zlef-seo-->`;

const JSONLD = (lang) => `<script type="application/ld+json">${JSON.stringify({
  '@context': 'https://schema.org',
  '@type': 'SoftwareApplication',
  name: 'ZlefRemote',
  applicationCategory: 'UtilitiesApplication',
  operatingSystem: 'Linux, Windows, macOS',
  description: DESC[lang] || DESC.en,
  url: 'https://remote.zlef.fr/',
  image: 'https://remote.zlef.fr/og.png',
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
        <span class="dlicon">${osIcon(p.key)}</span>
        <span class="dlname">${p.label}</span>
        <span class="dlsub">${p.sub}</span>
        <span class="dlget">${t['dl_' + (p.key === 'win' ? 'win' : p.key === 'mac' ? 'mac' : 'linux')]} ↓</span>
      </a>`;
    }
    return `<a class="dlcard soon" href="https://github.com/zlef-fr/zlefremote#build-from-source">
        <span class="dlicon">${osIcon(p.key)}</span>
        <span class="dlname">${p.label}</span>
        <span class="dlsub">${p.sub}</span>
        <span class="dlget muted">${t.dl_source} →</span>
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
<link rel="stylesheet" href="/css/site.css?v=6">
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
        <a class="zl-btn zl-btn--primary zl-btn--lg" href="#download">${t.cta_download}</a>
        <a class="zl-btn zl-btn--ghost zl-btn--lg" href="#how">${t.cta_how}</a>
      </div>
      <div class="badges">
        <span class="badge">🔒 ${t.badge_e2ee}</span>
        <span class="badge">📱 ${t.badge_noinstall}</span>
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
        <div class="mode-ico">🏠</div>
        <h3>${t.mode_lan_t}</h3>
        <p>${t.mode_lan_b}</p>
      </div>
      <div class="mode-card reveal accent2">
        <div class="mode-ico">🌍</div>
        <h3>${t.mode_remote_t}</h3>
        <p>${t.mode_remote_b}</p>
      </div>
    </div>
    <div class="e2ee reveal">
      <h3>🔐 ${t.e2ee_title}</h3>
      <p>${t.e2ee_b}</p>
    </div>
  </section>

  <section id="features" class="features">
    <h2 class="zl-h2 sect-title reveal">${t.feat_title}</h2>
    <div class="feat-grid">
      <div class="feat reveal"><h3>🖱️ ${t.feat_1_t}</h3><p>${t.feat_1_b}</p></div>
      <div class="feat reveal"><h3>⌨️ ${t.feat_2_t}</h3><p>${t.feat_2_b}</p></div>
      <div class="feat reveal"><h3>🎚️ ${t.feat_3_t}</h3><p>${t.feat_3_b}</p></div>
      <div class="feat reveal"><h3>🛡️ ${t.feat_4_t}</h3><p>${t.feat_4_b}</p></div>
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

<script src="/js/landing.js?v=1" defer></script>
</body>
</html>`;
}

function osIcon(key) {
  if (key === 'win') return '⊞';
  if (key === 'mac') return '';
  return '🐧';
}

module.exports = { landing, SEO, platformLinks };
