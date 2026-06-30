# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in ZlefRemote, please report it responsibly.

**Do not open a public GitHub issue for security vulnerabilities.**

### How to report

Email: **claude@zlef.fr**

Please include:
- A description of the vulnerability and its potential impact
- Steps to reproduce (proof-of-concept code or a detailed description)
- The version or commit hash you tested against
- Your name/handle if you would like to be credited (optional)

### What to expect

- We will acknowledge receipt within **72 hours**.
- We will provide an estimated timeline for a fix within **7 days**.
- We will notify you when a fix is released and credit you (unless you prefer to remain anonymous).
- We ask that you keep the issue confidential until a fix has been released.

## Supported Versions

Only the latest release of the relay server (`server.js`) and desktop agent is actively maintained.

## Scope

In scope:
- The relay server (`server.js`, `lib/`)
- The PWA client (`public/app/`)
- The desktop agent (`agent/`)
- End-to-end encryption implementation

Out of scope:
- Third-party dependencies — report those to the relevant upstream project
- Issues requiring physical access to the host machine

## Known design choices

- The relay is **blind** to session content (AES-256-GCM E2EE; the relay holds no key).
- The encryption key lives only in the URL fragment and is never transmitted to any server.
- The agent operates under the user account that runs it — it has the same OS privileges as that user by design.
