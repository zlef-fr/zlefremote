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
  softwareVersion: '1.1.0',
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

// Shared Open Graph + Twitter block for the secondary pages (/start, /privacy),
// so a shared link always renders a proper card.
function ogTags(title, desc, url, lang) {
  return `<meta property="og:type" content="website">
<meta property="og:site_name" content="ZLEF">
<meta property="og:title" content="${title}">
<meta property="og:description" content="${desc}">
<meta property="og:url" content="${url}">
<meta property="og:image" content="https://remote.zlef.fr/og.png?v=2">
<meta property="og:image:width" content="1200">
<meta property="og:image:height" content="630">
<meta property="og:locale" content="${lang === 'fr' ? 'fr_FR' : 'en_US'}">
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:title" content="${title}">
<meta name="twitter:description" content="${desc}">
<meta name="twitter:image" content="https://remote.zlef.fr/og.png?v=2">`;
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
<link rel="stylesheet" href="/css/site.css?v=11">
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
    <p class="dl-xfce reveal">${t.dl_xfce}
      <a href="/download/zlefremote-xfce-plugin.tar.gz?v=4" download>${t.dl_xfce_cta} ${icon('arrow')}</a>
    </p>
  </section>
</main>

<footer class="foot">
  <span>${t.footer_made}</span>
  <span class="foot-links"><a href="/privacy">${t.priv_link}</a><a href="https://zlef.fr">zlef.fr</a></span>
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
<meta name="robots" content="index, follow">
<meta name="description" content="${t.start_sub}">
<link rel="canonical" href="https://remote.zlef.fr/start">
<meta name="theme-color" content="#06060a">
<link rel="icon" href="https://assets.zlef.fr/favicon.svg" type="image/svg+xml">
<link rel="apple-touch-icon" href="https://assets.zlef.fr/apple-touch-icon.png">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${t.start_title} — ZlefRemote</title>
${ogTags(t.start_title + ' — ZlefRemote', t.start_sub, 'https://remote.zlef.fr/start', lang)}
<link rel="stylesheet" href="https://da.zlef.fr/tokens.css">
<link rel="stylesheet" href="/css/site.css?v=11">
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

      <a class="openapp" href="/r/">
        <span class="openapp-ic">${icon('phone')}</span>
        <span class="openapp-txt">
          <span class="openapp-t">${t.start_openapp} ${icon('arrow')}</span>
          <span class="openapp-s">${t.start_openapp_sub}</span>
        </span>
      </a>
    </div>
  </section>
</main>

<footer class="foot">
  <span>${t.footer_made}</span>
  <span class="foot-links"><a href="/privacy">${t.priv_link}</a><a href="https://zlef.fr">zlef.fr</a></span>
</footer>
<script src="/js/start.js?v=1" defer></script>
</body>
</html>`;
}

// ── Privacy policy ──────────────────────────────────────────────────────────
const PRIV = {
  en: {
    title: 'Privacy',
    updated: 'Last updated 24 June 2026',
    intro: 'ZlefRemote is built so that we — and anyone in between — cannot see what you do with it. This page explains, in plain terms, exactly what happens to your data.',
    sections: [
      { h: 'The short version', p: [
        'No account is required to use ZlefRemote.',
        'We cannot see your keystrokes, mouse movements or screen. In remote mode everything is end-to-end encrypted, and the relay only ever handles sealed data it has no key to open.',
        'In local-network mode, nothing leaves your own network at all.',
        'No tracking cookies. No advertising. Nothing sold to anyone.',
      ] },
      { h: 'The agent on your computer', p: [
        'The agent is a small program you run on the computer you want to control. It injects mouse and keyboard events into your operating system and does nothing else in the background.',
        'Each time it starts it generates a fresh 256-bit encryption key, on your machine. That key is placed only inside the QR code it shows you — it is never sent to any server.',
        'If you start the agent with the optional --remember flag, the key is instead saved locally on that computer (in your user config directory, readable only by your account) so your saved phones can reconnect in one tap without rescanning. The relay “room” is then derived from that key by a one-way hash — it still reveals nothing about the key and is never sent to any server. You can rotate it any time with --reset-identity.',
        'On startup the agent sends one anonymous ping (its version, operating system, architecture and the chosen mode — nothing else, and nothing about your session, your input or the computer) so we can gauge how many people use it. You can turn it off with the --no-telemetry flag, the DO_NOT_TRACK=1 environment variable, or a build compiled with telemetry off — see the repository for details.',
        'Apart from that single ping, the agent contains no analytics or telemetry. Its source code is public, so anyone can verify all of this.',
      ] },
      { h: 'Remote mode — the relay (remote.zlef.fr)', p: [
        'When you connect from outside your network, your phone and computer pair through our relay using a short room code. The relay’s only job is to pass messages between the two.',
        'Every message is encrypted on the device that sends it and decrypted only on the device that receives it. The relay holds no key, so it cannot read a single keystroke or mouse movement — it forwards opaque bytes.',
        'Sessions live in memory only. A room exists while your session is open and is discarded when you disconnect or after a period of inactivity. We never store the contents of a session.',
      ] },
      { h: 'Local-network mode', p: [
        'On the same Wi-Fi, your phone talks directly to your computer. The relay is not involved and your data never leaves your network.',
      ] },
      { h: 'This website', p: [
        'Analytics: our pages load a lightweight, shared analytics script that counts anonymous page views and visible time. It uses no cookies and no fingerprinting; a per-visit identifier lives only in your browser tab and is erased when you close it. It never follows you across sites.',
        'Server logs: like any website, requests to remote.zlef.fr reach our server, which keeps short-lived technical logs (IP address, browser user-agent, time and path) to keep the service running and to prevent abuse.',
        'Optional sign-in: you can use ZlefRemote without an account. If you choose to sign in with a zlef.fr account, we recognise it only to attribute your session and apply fair-use limits.',
        '“Send to your desktop” buttons: the email, Telegram, share and copy actions on the “Start a session” page run inside your own apps. The only information involved is the public download link — we do not see or store anything from those actions.',
      ] },
      { h: 'No third parties', p: [
        'We don’t use advertising networks, and we don’t sell or share your data. The relay and this website are operated as part of zlef.fr.',
      ] },
      { h: 'Your choices & contact', p: [
        'Because we don’t hold the contents of your sessions, there is nothing of yours for us to retrieve or delete from them. For anything else — a question, or a request about website logs or an optional account — write to claude@zlef.fr.',
      ] },
      { h: 'Changes', p: [
        'If this policy changes, we’ll update the date at the top of this page.',
      ] },
    ],
  },
  fr: {
    title: 'Confidentialité',
    updated: 'Dernière mise à jour : 24 juin 2026',
    intro: 'ZlefRemote est conçu pour que nous — et quiconque sur le chemin — ne puissions pas voir ce que vous en faites. Cette page explique, en clair, ce qu’il advient de vos données.',
    sections: [
      { h: 'En résumé', p: [
        'Aucun compte n’est nécessaire pour utiliser ZlefRemote.',
        'Nous ne pouvons voir ni vos frappes, ni vos mouvements de souris, ni votre écran. En mode distant, tout est chiffré de bout en bout, et le relais ne manipule que des données scellées qu’il n’a aucune clé pour ouvrir.',
        'En mode réseau local, rien ne quitte votre propre réseau.',
        'Aucun cookie de suivi. Aucune publicité. Rien n’est vendu à qui que ce soit.',
      ] },
      { h: 'L’agent sur votre ordinateur', p: [
        'L’agent est un petit programme que vous lancez sur l’ordinateur à contrôler. Il injecte les événements souris et clavier dans votre système et ne fait rien d’autre en arrière-plan.',
        'À chaque démarrage, il génère une clé de chiffrement de 256 bits, sur votre machine. Cette clé est placée uniquement dans le QR code qu’il affiche — elle n’est jamais envoyée à un serveur.',
        'Si vous lancez l’agent avec l’option facultative --remember, la clé est au contraire enregistrée localement sur cet ordinateur (dans votre dossier de configuration, lisible par votre seul compte) afin que vos téléphones enregistrés se reconnectent en un geste sans rescanner. Le « salon » du relais est alors dérivé de cette clé par un hachage à sens unique — il ne révèle toujours rien de la clé et n’est jamais envoyé à un serveur. Vous pouvez la régénérer à tout moment avec --reset-identity.',
        'Au démarrage, l’agent envoie un seul ping anonyme (sa version, son système d’exploitation, son architecture et le mode choisi — rien d’autre, et rien sur votre session, vos saisies ou l’ordinateur) afin que nous puissions estimer le nombre d’utilisateurs. Vous pouvez le désactiver avec l’option --no-telemetry, la variable d’environnement DO_NOT_TRACK=1, ou une version compilée sans télémétrie — voir le dépôt pour les détails.',
        'En dehors de ce ping unique, l’agent ne contient aucune analyse d’audience ni télémétrie. Son code source est public : chacun peut tout vérifier.',
      ] },
      { h: 'Mode distant — le relais (remote.zlef.fr)', p: [
        'Quand vous vous connectez depuis l’extérieur de votre réseau, votre téléphone et votre ordinateur s’appairent via notre relais à l’aide d’un court code de salon. Le seul rôle du relais est de transmettre les messages entre les deux.',
        'Chaque message est chiffré sur l’appareil qui l’envoie et déchiffré seulement sur celui qui le reçoit. Le relais ne détient aucune clé : il ne peut donc lire ni une frappe ni un mouvement de souris — il transmet des octets opaques.',
        'Les sessions n’existent qu’en mémoire. Un salon existe pendant votre session et est détruit à la déconnexion ou après une période d’inactivité. Nous ne conservons jamais le contenu d’une session.',
      ] },
      { h: 'Mode réseau local', p: [
        'Sur le même Wi-Fi, votre téléphone parle directement à votre ordinateur. Le relais n’intervient pas et vos données ne quittent jamais votre réseau.',
      ] },
      { h: 'Ce site web', p: [
        'Mesure d’audience : nos pages chargent un script d’analyse léger et partagé qui compte des visites anonymes et le temps de lecture visible. Il n’utilise ni cookie ni empreinte numérique ; un identifiant par visite vit uniquement dans l’onglet et disparaît à sa fermeture. Il ne vous suit jamais d’un site à l’autre.',
        'Journaux serveur : comme tout site, les requêtes vers remote.zlef.fr atteignent notre serveur, qui conserve de brefs journaux techniques (adresse IP, user-agent du navigateur, heure et chemin) pour faire fonctionner le service et prévenir les abus.',
        'Connexion facultative : vous pouvez utiliser ZlefRemote sans compte. Si vous choisissez de vous connecter avec un compte zlef.fr, nous le reconnaissons uniquement pour attribuer votre session et appliquer des limites d’usage équitable.',
        'Boutons « envoyer vers votre ordinateur » : les actions e-mail, Telegram, partage et copie de la page « Démarrer une session » s’exécutent dans vos propres applications. La seule information en jeu est le lien public de téléchargement — nous ne voyons ni ne stockons rien de ces actions.',
      ] },
      { h: 'Aucun tiers', p: [
        'Nous n’utilisons pas de régie publicitaire et nous ne vendons ni ne partageons vos données. Le relais et ce site sont exploités dans le cadre de zlef.fr.',
      ] },
      { h: 'Vos choix et contact', p: [
        'Comme nous ne détenons pas le contenu de vos sessions, il n’y a rien à en récupérer ni à en supprimer. Pour le reste — une question, ou une demande concernant les journaux du site ou un compte facultatif — écrivez à claude@zlef.fr.',
      ] },
      { h: 'Modifications', p: [
        'Si cette politique change, nous mettrons à jour la date en haut de cette page.',
      ] },
    ],
  },
};

function privacyPage(lang) {
  const t = dict(lang);
  const c = PRIV[lang] || PRIV.en;
  const sections = c.sections.map((s) =>
    `<section class="priv-sec">
      <h2>${s.h}</h2>
      ${s.p.map((para) => `<p>${para}</p>`).join('')}
    </section>`).join('\n');

  return `<!doctype html>
<html lang="${lang}">
<head>
<meta charset="utf-8">
<meta name="robots" content="index, follow">
<meta name="description" content="${c.intro}">
<link rel="canonical" href="https://remote.zlef.fr/privacy">
<meta name="theme-color" content="#06060a">
<link rel="icon" href="https://assets.zlef.fr/favicon.svg" type="image/svg+xml">
<link rel="apple-touch-icon" href="https://assets.zlef.fr/apple-touch-icon.png">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${c.title} — ZlefRemote</title>
${ogTags(c.title + ' — ZlefRemote', c.intro, 'https://remote.zlef.fr/privacy', lang)}
<link rel="stylesheet" href="https://da.zlef.fr/tokens.css">
<link rel="stylesheet" href="/css/site.css?v=11">
</head>
<body class="legal">
<script defer src="https://assets.zlef.fr/track.js"></script>
<div class="bg-field"></div>

<header class="nav">
  <a class="brand" href="/"><span class="brand-mark">◑</span><span class="brand-name">Zlef<b>Remote</b></span></a>
  <nav class="nav-links">
    <a class="zl-btn zl-btn--sm zl-btn--ghost" href="/">${icon('arrow', 'flip')} ${t.start_back}</a>
  </nav>
</header>

<main class="legal-main">
  <h1 class="zl-h1 legal-h1">${c.title}</h1>
  <p class="legal-updated">${c.updated}</p>
  <p class="legal-intro">${c.intro}</p>
  ${sections}
</main>

<footer class="foot">
  <span>${t.footer_made}</span>
  <span class="foot-links"><a href="/privacy">${t.priv_link}</a><a href="https://zlef.fr">zlef.fr</a></span>
</footer>
</body>
</html>`;
}

module.exports = { landing, startPage, privacyPage, SEO, platformLinks };
