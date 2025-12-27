# Changelog

## [0.0.9] - 2025-12-27

- CLI: svg export.
- CLI: `reassign` command.

## [0.0.8] - 2025-12-26

- Feature: New commands `core-ranking"`, `core-ranking-summary`, `core-vm-affinity`
- CLI: Show the data of the new commands.
- BUG: `pvesh` executable pointed to a wrong path - this broke `0.0.7`.

## [0.0.7] - 2025-12-26

RELEASE (DELETED: this was broken)

- Testing: Fuzz testing for all external commands
- Testing: increased coverage
- Feature: arm64 support (https://github.com/jiangcuo/pxvirt / https://gitee.com/jiangcuo/Proxmox-Port)

## [0.0.6] - 2025-12-26

- Refactor: Renamed internal command `vm-started` to `update-affinity`.
- Doc: Added optional Proxmox Patch link.

## [0.0.5] - 2025-12-26

- CLI: `ps` command now displays VM cores, sockets, and NUMA configuration.
- CLI: `hookscript` commands now verify storage existence and status (override with `--force`).

## [0.0.4] - 2025-12-26

- CLI: Added json output, spinner, dry-run.
