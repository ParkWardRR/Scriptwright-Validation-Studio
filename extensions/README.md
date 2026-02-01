# Extension bundles (placeholders)

Add unpacked MV3 userscript manager builds here for deterministic loading:
- `extensions/tampermonkey-mv3` — unpacked MV3 build (Chrome/Chromium).
- `extensions/violentmonkey-firefox` — XPI unpacked directory for Firefox.

Set `USERSCRIPT_ENGINE_EXT_DIR=extensions/tampermonkey-mv3` (or `--ext` flag) to load via `--disable-extensions-except/--load-extension` in the runner.

Note: No binaries are bundled in this repo; drop your vetted versions here.
