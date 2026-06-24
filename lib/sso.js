'use strict';
// Optional SSO: replay the visitor's Authelia session cookie against the
// forward-auth endpoint. 200 → identity headers; anything else → anonymous.
// Used only for a friendly greeting + a higher room quota for signed-in users.
const http = require('http');

function whoami(req) {
  return new Promise((resolve) => {
    const cookie = req.headers.cookie || '';
    if (!cookie.includes('authelia_session')) return resolve(null);
    const r = http.request({
      host: '127.0.0.1', port: 10004, path: '/api/authz/forward-auth', method: 'GET',
      headers: {
        'X-Forwarded-Method': 'GET',
        'X-Forwarded-Proto': 'https',
        'X-Forwarded-Host': 'remote.zlef.fr',
        'X-Forwarded-Uri': '/',
        Cookie: cookie,
      },
      timeout: 1500,
    }, (res) => {
      if (res.statusCode === 200) {
        resolve({ user: res.headers['remote-user'] || null, name: res.headers['remote-name'] || null });
      } else resolve(null);
      res.resume();
    });
    r.on('error', () => resolve(null));
    r.on('timeout', () => { r.destroy(); resolve(null); });
    r.end();
  });
}

module.exports = { whoami };
