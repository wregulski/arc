# CHANGELOG

All notable changes to this project will be documented in this file. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Table of Contents
- [Unreleased](#unreleased)
- [1.0.62](#1062---2023-11-23)
- [1.0.60](#1060---2023-11-21)
- [1.0.0 - YYYY-MM-DD](#100---yyyy-mm-dd)

## [Unreleased]

## [1.0.62] - 2023-11-23

### Added

- Transaction status `SEEN_IN_ORPHAN_MEMPOOL`. The transaction has been sent to at least 1 Bitcoin node but parent transaction was not found. This status means that inputs are currently missing, but the transaction is not yet rejected.

### Changed

- A transaction for which a ZMQ message `missing inputs` of topic `invalidtx` is received, that transaction gets status `SEEN_IN_ORPHAN_MEMPOOL`.

## [1.0.60] - 2023-11-21

### Changed
- BREAKING CHANGE: The Merkle Path Binary format previously calculated and stored in BlockTx has been updated to a new standard encoding format referred to as BUMP and detailed here: [BRC-74](https://brc.dev/74). This means that the BlockTx database ought to be dumped prior to updating to using this version, since the structs are incompatible.

### Deprecated
- This has the effect of deprecating the previously used Merkle Path Binary format detailed here: [BRC-71](https://brc.dev/71) which is not used anywhere else in the ecosystem to our knowledge.

---

## [1.0.0] - YYYY-MM-DD

### Added
- Initial release

---

### Template for New Releases:

Replace `X.X.X` with the new version number and `YYYY-MM-DD` with the release date:

```
## [X.X.X] - YYYY-MM-DD

### Added
-

### Changed
-

### Deprecated
-

### Removed
-

### Fixed
-

### Security
-
```

Use this template as the starting point for each new version. Always update the "Unreleased" section with changes as they're implemented, and then move them under the new version header when that version is released.
