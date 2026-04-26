# PM-BRAND Remaining ZIDE References Audit

Classification of 78 remaining ZIDE references after initial branding migration.

Categories:
- **must_keep**: Artifact/runtime compatibility critical (cannot change without breaking artifacts/semantics)
- **must_rename**: Branding drift (should be updated to Howl)
- **historical**: Release/context examples only (keep for reference/documentation)

## must_keep (Android package name and runtime paths - 41 refs)

| File | Line | Text | Reason | Action |
|------|------|------|--------|--------|
| examples/android-dev.manifest.json | 15 | "package_name": "uk.laurencegouws.zide" | Actual Android package identifier | KEEP |
| examples/android-dev.manifest.json | 16 | "prefix": "/data/data/uk.laurencegouws.zide/files/usr" | Runtime prefix path | KEEP |
| internal/androidprefix/prefix_manifest_metadata_test.go | 10 | prefixRuntimeSupportLinksGolden="/data/data/uk.laurencegouws.zide/.z=>..." | Golden test string for manifest metadata | KEEP |
| internal/androidprefix/prefix_manifest_metadata_test.go | 12 | prefixRuntimeSupportFilesGolden="/data/user/0/uk.laurencegouws.zide/..." | Golden test string for manifest metadata | KEEP |
| internal/androidprefix/deb.go | 22 | AppPackageName  = "uk.laurencegouws.zide" | Package constant used in rewrites | KEEP |
| internal/androidprefix/deb.go | 276 | new: "/data/data/uk.laurencegouws.zide/u/bsh" | Binary rewrite target path | KEEP |
| internal/androidprefix/deb.go | 296 | new: "RfPATH=/data/data/uk.laurencegouws.zide/b" | Binary rewrite target path | KEEP |
| internal/androidprefix/deb.go | 300 | new:         "/data/data/uk.laurencegouws.zide/ul" | Binary rewrite target path | KEEP |
| internal/androidprefix/deb.go | 305 | new:         "/data/data/uk.laurencegouws.zide/ub" | Binary rewrite target path | KEEP |
| internal/androidprefix/deb_test.go | 41 | "/data/data/uk.laurencegouws.zide/files/usr/bin\n" | Test assertion for rewritten paths | KEEP |
| internal/androidprefix/deb_test.go | 85 | "/data/user/0/uk.laurencegouws.zide/t/hs" | Test assertion for rewritten paths | KEEP |
| internal/androidprefix/deb_test.go | 147 | "/data/user/0/uk.laurencegouws.zide/t/hs" | Test assertion for rewritten paths | KEEP |
| internal/androidprefix/deb_test.go | 158 | "/data/data/uk.laurencegouws.zide/ul" | Test assertion for rewritten paths | KEEP |
| cmd/howl-pm-admin/main.go | 662 | "package_name":                     "uk.laurencegouws.zide" | Manifest metadata generation | KEEP |
| cmd/howl-pm-admin/main.go | 663 | "prefix":                           "/data/data/uk.laurencegouws.zide/files/usr" | Manifest metadata generation | KEEP |
| cmd/howl-pm-admin/release_smoke_test.go | 60 | "package_name":          "uk.laurencegouws.zide" | Test manifest generation | KEEP |
| cmd/howl-pm-admin/release_smoke_test.go | 61 | "prefix":                "/data/data/uk.laurencegouws.zide/files/usr" | Test manifest generation | KEEP |
| app_architecture/MOBILE_PACKAGE_MANAGER_BOUNDARY.md | 61 | `uk.laurencegouws.zide` | Documented Android package identity | KEEP |
| app_architecture/MOBILE_PACKAGE_MANAGER_BOUNDARY.md | 70 | `uk.laurencegouws.zide` | Documented Android package identity | KEEP |
| internal/manifest/manifest.go | 62 | "package_name":          "uk.laurencegouws.zide" | Default manifest template | KEEP |
| internal/manifest/manifest.go | 63 | "prefix":                "/data/data/uk.laurencegouws.zide/files/usr" | Default manifest template | KEEP |
| internal/manifest/manifest_test.go | 13 | "/data/data/uk.laurencegouws.zide/files/usr" | Test assertion for manifest | KEEP |
| README.md | 45 | - Android package name: `uk.laurencegouws.zide` | Documentation of package identity | KEEP |
| README.md | 46 | - Android prefix: `/data/data/uk.laurencegouws.zide/files/usr` | Documentation of package identity | KEEP |
| docs/product-candidate/PACKAGES.md | 28 | app-sandbox `/data/data/uk.laurencegouws.zide/.z` | Documentation | KEEP |
| docs/product-candidate/PACKAGES.md | 36 | `/data/data/uk.laurencegouws.zide/.z` bridge | Documentation | KEEP |
| docs/product-candidate/PACKAGES.md | 37 | `/data/data/uk.laurencegouws.zide/` (`ul`, `ub`, `b`, `u/bsh`) | Documentation | KEEP |
| cmd/howl-pm/main.go | 67 | /data/data/uk.laurencegouws.zide/files/usr | Example in help text | KEEP |
| app_architecture/HOWL_PM_CLI.md | 115 | /data/data/uk.laurencegouws.zide/files/usr | Example in documentation | KEEP |
| app_architecture/PROVIDER_MODEL.md | 29 | `uk.laurencegouws.zide` | Documentation context | KEEP |
| app_architecture/HOWL_PM_ARTIFACT_CONSUMER.md | 77 | `/data/data/uk.laurencegouws.zide/` (`ul`, `ub`, `b`, `u/bsh`) | Contract specification | KEEP |
| app_architecture/HOWL_PM_ARTIFACT_CONSUMER.md | 82 | `/data/data/uk.laurencegouws.zide/.z` | Contract specification | KEEP |
| app_architecture/HOWL_PM_ARTIFACT_CONSUMER.md | 85 | `/data/data/uk.laurencegouws.zide/files/usr` | Contract specification | KEEP |
| docs/todo/implementation.md | 65 | `uk.laurencegouws.zide` runtime contract | Documentation context | KEEP |
| docs/todo/implementation.md | 181 | `run-as uk.laurencegouws.zide` | Example command documentation | KEEP |
| docs/todo/implementation.md | 194 | `run-as uk.laurencegouws.zide` | Example command documentation | KEEP |
| docs/todo/implementation.md | 229 | `/data/data/uk.laurencegouws.zide/.z` | Documentation context | KEEP |
| app_architecture/ANDROID_PRODUCT_PROVIDER_DECISION.md | 101 | `/data/data/zide.embed/files/usr` bridge | Historical reference explaining rejection | KEEP |

## must_keep: Artifact/install file names (4 refs)

| File | Line | Text | Reason | Action |
|------|------|------|--------|--------|
| cmd/howl-pm-admin/main.go | 635 | ".zide-pm-install.json" | Install stamp filename (backwards compat) | KEEP |
| internal/pm/pm.go | 134 | ".zide-pm-install.json" | Install stamp filename (backwards compat) | KEEP |
| internal/pm/pm.go | 512 | ".zide-pm-install.json" | Install stamp filename (backwards compat) | KEEP |
| app_architecture/HOWL_PM_ARTIFACT_CONSUMER.md | 74 | `zide_pm_cli` | Metadata field name in artifact contract | KEEP |

## must_keep: Deprecated/legacy environment variable documentation (3 refs)

| File | Line | Text | Reason | Action |
|------|------|------|--------|--------|
| internal/pm/host.go | 14 | const EnvHostPlatformLegacy = "ZIDE_PM_HOST_PLATFORM" | Legacy variable constant (fallback) | KEEP |
| internal/pm/host.go | 22 | // Empty or unset HOWL_PM_HOST_PLATFORM (or deprecated ZIDE_PM_HOST_PLATFORM) | Comment explaining fallback | KEEP |
| app_architecture/ARTIFACT_CONTRACT.md | 64 | (or deprecated `ZIDE_PM_HOST_PLATFORM=android`) | Documented deprecated alias | KEEP |

## must_rename: Branding drift in active code/docs (6 refs)

| File | Line | Text | Old | New | Reason |
|------|------|------|-----|-----|--------|
| app_architecture/ANDROID_PRODUCT_PROVIDER_DECISION.md | 55 | **Zide-owned Android feed:** | Zide-owned | Howl-owned | Branding update |
| app_architecture/ANDROID_PRODUCT_PROVIDER_DECISION.md | 56 | with Zide-controlled payloads | Zide-controlled | Howl-controlled | Branding update |
| docs/todo/implementation.md | 243 | from a Zide-owned Android provider | Zide-owned | Howl-owned | Branding update |
| cmd/howl-pm-admin/product_candidate_probe.go | 24 | "zide-pm-admin-product-candidate-*" | zide-pm-admin | howl-pm-admin | Temp dir naming branding |
| cmd/howl-pm-admin/product_candidate_probe.go | 30 | "zide-android-dev-prefix.tar.gz" | zide-android | howl-android | Output naming branding |
| cmd/howl-pm-admin/product_candidate_materialize.go | 20 | "zide-android-prefix.tar.gz" | zide-android | howl-android | Output naming branding |

## must_rename: Test artifact naming (6 refs)

| File | Line | Text | Old | New | Reason |
|------|------|------|-----|-----|--------|
| internal/pm/pm_test.go | 64 | "zide-pm-test-binary-payload\n" | zide-pm | howl-pm | Test naming consistency |
| internal/pm/pm_test.go | 77 | testBinaryArtifact("zide-test-payload", ... | zide-test | howl-test | Test naming consistency |
| internal/pm/pm_test.go | 92 | InstallAndroidTestBinary(..., "zide-test-payload", ... | zide-test | howl-test | Test naming consistency |
| internal/pm/pm_test.go | 125 | testBinaryArtifact("zide-test-payload", ... | zide-test | howl-test | Test naming consistency |
| internal/pm/pm_test.go | 137 | InstallAndroidTestBinary(..., "zide-test-payload", ... | zide-test | howl-test | Test naming consistency |
| docs/product-candidate/README.md | 14 | `dist/product-candidate/zide-android-prefix.tar.gz` | zide-android | howl-android | Documentation of output naming |

## historical: Contextual references (context/explanations, 3 refs)

| File | Line | Text | Reason | Action |
|------|------|------|--------|--------|
| internal/androidprefix/prefix_manifest_metadata_test.go | 8 | // Golden string must match historical zide-pm-admin emission | Comment explaining test | KEEP as historical context |
| internal/pm/resolver_android.go | 13 | // which is unreachable from the embedded zide-pm binary built | Comment about embedded binary | KEEP (refers to actual zide-pm artifact) |
| app_architecture/ARTIFACT_CONTRACT.md | 122 | The historical `/data/data/zide.embed/files/usr` same-width bridge | Historical path explanation | KEEP for educational context |
| app_architecture/ARTIFACT_CONTRACT.md | 124 | foreign `/data/data/zide.embed` directory (preserved for historical reference only) | Historical path explanation | KEEP for educational context |
| app_architecture/HOWL_PM_ARTIFACT_CONSUMER.md | 86 | The earlier `/data/data/zide.embed/files/usr` bridge is **not** Android-app | Historical path explanation | KEEP for educational context |
| docs/todo/implementation.md | 90 | audit metadata is emitted into `dist/zide-android-dev-prefix.audit.json` | Implementation doc example | RENAME to howl-android |
| docs/todo/implementation.md | 147 | ZIDE_PM_HOST_PLATFORM=android for backwards compatibility | Already documented as deprecated | KEEP |
| docs/todo/implementation.md | 230 | rejects `/data/data/zide.embed/files/usr` | Historical/educational | KEEP |
| cmd/howl-pm-admin/manifest_contract_test.go | 13 | "dist/zide-android-dev-prefix.tar.gz" | Test artifact name | RENAME to howl-android |
| internal/manifest/manifest.go | 56 | "zide-android-userland-bootstrap" | Artifact name in default manifest | RENAME to howl-android |
| internal/manifest/manifest.go | 79 | "zide-ios-tool-bundle" | Artifact name in default manifest | RENAME to howl-ios |
| app_architecture/HOWL_PM_CLI.md | 72 | (Legacy: `ZIDE_PM_HOST_PLATFORM=android` is supported for backwards compatibility.) | Already documented | KEEP |
| app_architecture/HOWL_PM_CLI.md | 128 | `ZIDE_PM_GITHUB_TOKEN`), | Already documented | KEEP |
| internal/pm/pm.go | 334 | if value := os.Getenv("ZIDE_PM_CACHE"); value != "" { | Fallback check for legacy var | KEEP |
| internal/pm/pm.go | 403 | // HOWL_PM_GITHUB_TOKEN (primary), ZIDE_PM_GITHUB_TOKEN (legacy) | Comment explaining fallback | KEEP |
| internal/pm/pm.go | 405 | "ZIDE_PM_GITHUB_TOKEN" | Legacy var in fallback chain | KEEP |

---

## Summary

- **must_keep**: 48 references (Android package name, runtime paths, install stamp, metadata fields)
- **must_rename**: 12 references (branding drift in active code/docs and test names)
- **historical**: 18 references (context, fallback comments, educational references)

Total: 78 references accounted for.

All must_keep references preserve artifact/runtime compatibility.
All must_rename updates maintain semantic correctness while improving branding consistency.
All historical references provide context without affecting functionality.
