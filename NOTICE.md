# Upstream inspiration and attribution

`codex-shadow-guard` was designed after reviewing [Pi Shadow Mind](https://github.com/liuzhengdongfortest/pi-shadow-mind), copyright (c) 2026 `liuzhengdongfortest`, licensed under MIT.

This repository does **not** contain a copied Pi Shadow Mind runtime, Pi Extension API implementation, TypeScript source, package metadata, tests, or artwork.

The concepts deliberately adapted here are:

- persistent, specialized review responsibilities;
- least-privilege reviewer capabilities;
- evidence-based reporting instead of noisy intervention;
- explicit user-managed configuration.

The implementation is new Go code for Codex lifecycle Hooks and project `AGENTS.md` blocks. See [docs/MIGRATION.md](docs/MIGRATION.md) for the detailed feature mapping and intentional omissions.

The upstream MIT license permits this kind of derivative design work. Its original copyright and license remain applicable to copies of upstream material; no such source material is distributed in this repository.
