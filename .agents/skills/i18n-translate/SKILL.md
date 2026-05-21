---
name: i18n-translate
description: >-
  Complete and maintain frontend i18n translations for the classic frontend.
  Use when adding or fixing user-visible text in web/classic, checking missing
  translation keys, or keeping supported locale files aligned.
---

# Frontend i18n Translation Workflow

## Overview

- Locale files: `web/classic/src/i18n/locales/{en,zh,zh-CN,zh-TW,fr,ja,ru,vi}.json`
- Format: flat JSON under the `"translation"` key.
- Base locale: `en.json`; Chinese locales are `zh.json`, `zh-CN.json`, and `zh-TW.json`.
- Sync script: `bun run i18n:sync` from `web/classic/`.
- Status script: `bun run i18n:status` from `web/classic/`.
- User-visible strings in React code should use `useTranslation()` and `t('...')`.

## Workflow

1. Run sync from the classic frontend:

```bash
cd web/classic
bun run i18n:sync
```

2. Check status:

```bash
cd web/classic
bun run i18n:status
```

3. If adding translations manually, update every supported locale file under
   `web/classic/src/i18n/locales/` and keep the `{{variable}}` placeholders
   unchanged.

4. For temporary helper scripts, place them under `web/classic/scripts/`, run
   them from `web/classic/`, then delete them before finishing.

5. After changes, run:

```bash
cd web/classic
bun run i18n:sync
bun run i18n:status
```

## Translation Guidelines

| Language | Code | Notes |
|----------|------|-------|
| English | en | Base locale |
| Chinese | zh, zh-CN | Simplified Chinese |
| Traditional Chinese | zh-TW | Traditional Chinese |
| French | fr | Use natural UI French |
| Japanese | ja | Use katakana for technical loanwords |
| Russian | ru | Use formal register |
| Vietnamese | vi | Use standard Vietnamese |

Keep brand names, URLs, API paths, model names, JSON keys, and code-like strings
unchanged unless the surrounding UI text requires translation.

Always translate UI labels, button text, error messages, descriptions, time
units, and action words.
