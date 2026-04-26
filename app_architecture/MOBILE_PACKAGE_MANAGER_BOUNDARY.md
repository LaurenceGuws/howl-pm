# Mobile Package Manager Boundary

Purpose: define why `howl-pm` exists and where it stops.

## Product Role

`howl-pm` is the mobile package authority and repo identity for the Howl
project family.

It is used by Howl, but it should remain portable enough to support other
Howl-family mobile consumers that need the same artifact discipline.

## Ownership Split

`howl-pm` owns:

- the `howl-pm` package CLI
- the `howl-pm-admin` backend/admin tool
- artifact manifests
- provider metadata and trust boundaries
- package/bootstrap metadata
- package index snapshots
- checksums
- artifact cache layout
- host-side materialization tools
- release/channel policy for mobile artifacts

`howl` owns:

- runtime integration
- terminal/editor UX
- Android lifecycle/input/insets
- native rendering
- PTY/session semantics
- user-facing bootstrap progress and errors

Artifact staging obligations for mobile prefixes are constrained by
`HOWL_PM_ARTIFACT_CONSUMER.md` (MP-A7): manifests and hashes are the
contract; package internals stay in `howl-pm` admin tooling.

## Android First

Android currently needs:

- a Bash-capable app-private prefix
- Neovim and terminal-dev tools for implementation testing
- package metadata that is not silently tied to `com.termux`
- exact artifact versions and checksums

Android may use:

- providers such as `termux-main`
- prefix archives
- package index snapshots
- package recipes or binary payloads
- SDK 28 execution posture for terminal userland compatibility

Android must not claim:

- unmodified `com.termux` package payload roots are product-correct for
  `uk.laurencegouws.zide`
- host-side relocation hacks are a final package-manager contract

## Providers

Providers are build-time artifact sources, not Howl runtime package managers.

The current Android provider is `termux-main`. It is allowed as a development
and bootstrap source because its package index and payloads can be pinned and
audited. It is not automatically a product provider for `uk.laurencegouws.zide`.

Provider outputs must become Howl artifact contracts before Howl consumes them.
That keeps Android-specific mechanics out of the Howl runtime and leaves room
for a future Howl-owned Android feed or completely different iOS artifact
sources.

## Package CLI

`howl-pm` is the product-facing command intended for Howl mobile shells.

It consumes artifact contracts and reports provider provenance. It must not make
Termux, apt, or any future provider look like the Howl product surface.

`howl-pm-admin` is not a product shell command. It is host-side/admin tooling
for validation, provider snapshotting, archive generation, and snapshot
publishing.

The current MVP is `dev-baseline` only. Arbitrary package install, upgrade, and
remove semantics are later work.

## iOS Later

iOS should be treated as a separate execution model.

Likely iOS artifact types may include:

- bundled read-only tools
- editor/IDE resource bundles
- LSP or syntax asset bundles
- platform-approved helper artifacts

iOS should not inherit Android assumptions such as:

- apt/dpkg as a baseline
- writable executable userland prefixes
- arbitrary downloaded binary execution

The shared boundary is the manifest and trust contract, not the package
mechanics.

## Stop Line

If code needs to run inside the Howl app at runtime, it probably does not belong
here unless it is generated artifact metadata.

If code builds, verifies, signs, snapshots, or publishes mobile artifacts, it
belongs here unless it is still a short-lived probe.

MP-A6 product-candidate tooling (`android-product-candidate-*`) and operator
notes (`docs/product-candidate/`) live in this repo as **host-side** artifact
discipline, not app runtime code.
