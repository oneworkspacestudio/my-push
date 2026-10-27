# My Push and native Chat integration

## Request and scope

User requests a new self hosted Firebase-style app based on open source, added
to Workspace and used for Android Chat notifications. Implemented the messaging
service and console, not Firestore/Functions/Analytics or all Firebase products.
No current push/release/install or testing request. No Actions, owner fixtures,
phone/device use, external repo creation or publication performed.

## Source

New local repository I:/Projects/my-push, no commits/remotes yet. Apache-2.0 with
Centrifuge v0.40.1 MIT, upstream commit79c893e8692d5caf29e4c1e5983191e6e5ee3530.
Windows/Linux native Go executable, apphost MyAccounts owner console and separate
private push data. Original Workspace blue/mint SVG and generated ICO. Console
research, exact boundaries and API documented under docs/. Project overview,
Cloud Messaging, phones, queue/ack activity, project creation and key rotation.

Apphost dependencies intentionally use local replacements: one-workspace-android
and my-calendar. New source must be published/pinned together before a clean
public dependency build. This task did not modify my-calendar; its existing
Linux secure source is used to cross-build apphost. Windows uses unchanged DPAPI
through path-aware wrappers. Android/Chat changes live in the isolated worktrees
named below, preserving unrelated original workspace-chat Linux work.

## Security and data

Owner console uses apphost OIDC/CSRF/role checks. Project service credentials and
phone credentials stored hashed. Device channels are server-selected; client
publish/subscribe handlers deny by default. 30-minute grants, two connections per
device, bounded outbound queue. Project-scoped sender API only receives recipient
IDs; phone events contain random ID/time and opaque sync/check, never messages.
SQLite queue/ack coalesces pending wakes per phone, keeps24h; Chat owns content.
Disconnects close sockets and persist31-day tombstones; relink Chat to re-enable.
Chat revocation requests removal; unreachable broker expires old access in30min.
Acknowledgement means phone received a wake, never a Chat read receipt.

## Companion integration

- I:/Projects/one-workspace-android remains on codex/android-workspace based on
  b6f88cc with previous uncommitted Android work. Adds push app catalog/build,
  optional verified org pin/local package path, update grouping/requirements,
  storage/recovery profile, SSO/logout, navigation/README/branding and remote
  device transport exemptions authenticated by My Push. No published pins changed.
- Workspace derives Chat-scoped service key from protected Chat client secret,
  injects loopback and HTTPS public endpoints. Shared tunnel carries WebSocket.
- I:/Projects/chat-android codex/android-native from d8ec777: Tinode workspace push
  plugin, loopback device provisioning, native bootstrap tokens and revoke hooks.
  Matching managed source prioritizes My Push. Legacy optional FCM server adapter
  retained for other clients, no browser migration. Existing web runtime reused.
- Android preview2 removes Firebase SDK/dependencies, uses native foreground
  remoteMessaging service, private ongoing notification, capped jittered backoff,
  pending recovery/ack/dedup, identity fencing, WorkManager fallback, boot/package
  restore when permitted, native setting/battery-settings link. No silent permanent
  background privilege or iOS/APNs implementation. Force-stop/sleep/network and
  vendor battery restrictions can interrupt delivery. Native Telecom remains out.

## Build and acceptance

Windows My Push, Linux x64 My Push, Windows manager and matching Chat compile
locally. Android assembleDebug succeeds; package uk.co.m8e.workspace, version2 /
0.1.0-preview.2, min28/target35, debug signing only. No Firebase library remains
in Gradle lock. Source mirror and APK signature/metadata inspected. Visual-only
console screenshots reviewed at1440x1000 and390x844 with sample empty state;
these are not live service acceptance. No automated or real-phone acceptance run.

Artifacts/receipts/source archives: I:/Projects/tooling/my-push-20261027. Final
hashes and archive membership belong in build-receipt.json. C task build mirror
is C:/Projects/.cache/my-push-20261027; only that named disposable root may be
removed after artifact/source preservation and active process/reparse checks.
No owner installs/data/backup removed, no production deployment performed.

## Resume

Phone Doze/reboot/reply/revoke/reconnect and synthetic server acceptance still
required before reliability claim. Need production Android signing key and a
fresh authorized publish/install task. Reconcile manager branch with latest public
main (owner already93/MyAI14/Maps6/Tasks4 per peer); preserve all newer pins.94 in
development compatibility manifest is a provisional minimum, not a reserved or
published release. Set actual requirements during release coordination. Publish
My Push repo/app, matching Chat and manager before org pin/channel promotion.
Update channel locally, no Actions, normal protected owner install only. Full
suite Linux integration remains separate from this app's cross-build.

## Authorized release continuation, 27 October

User authorizes attached-phone testing, release and install. Public dependency is
Workspace source206ee30425f3, with self-contained Apache Linux helpers; no local
module replacements remain. Windows and Linux x64 release cross-builds succeed.
Isolated NTFS unit tests pass for scoped queue, ACK, coalescing, expiry, revocation
and anonymous rejection. Three-app fresh HTTPS fixture passes Alice/Bob SSO,
native link/bootstrap, bearer-only public transport, Centrifuge WebSocket wake
from a real authorized Tinode message, queue/ACK and immediate revocation. Phone
has debug2 installed (initially no package); UI acceptance awaits human unlock.
Permanent Android signing key outside Git. No physical sleep/reboot/action or
full-suite Linux acceptance claim. Builds/publishing remain local, Actions off.
Release My Push1, Chat11, Workspace94; preserve Maps8/MyAI14/Tasks4 pins. Deployment
receipts to be added after normal protected owner updates; no owner fixture use.

## Published and installed, 27 October

My Push2 source23161db is public with Windows/Linux x64 binaries, canonical
workspace-update.json (legacy alias retained), complete pinned dependency licence
texts and SHA256 receipts. Preview1 is withheld in Workspace's channel before
owner installation; its9019 catalogue port conflicted with My AI. Current9021 /
managed base+31 is distinct. All13 Push2 uploaded assets match local size/digest.
Public dependency remains Workspace206ee30425f3, no sibling replacements.

Workspace94/Chat11/Android2 are published without Actions. Final binary tests
pass scoped authorization/queue/expiry/revocation and16 native/public HTTPS
protocol checks, including server publish ACK, native sync/reply and fresh browser
session restoration. An optimistic direct-fallback message is not a server ACK;
the acceptance harness explicitly waits for commit before expecting a wake.
Console SSO/navigation/branding and390px no-overflow checks pass on real synthetic
service data. Primary owner PC now runs94/11/2, all28 services healthy; credentials,
settings and unselected executable bytes retained. Normal protected update adds
Push automatically; fixture also proves26 explicit app exclusions retained.

Android release-signed2 is installed and its phone APK hash equals the public
Workspace94 asset. App closed/Appium session deleted. Physical notification,
reply, sleep/reboot tests still blocked by locked phone. Secondary deployment
still pending owner MyAccounts sign-in. Do not claim both PCs/device acceptance.
Synthetic C state and all task preview services/tunnels were removed; permanent
release key is protected outside Git. Recreate fresh synthetic state for phone
acceptance; never use owner data as a fixture. Linux x64 cross-build passed, no
Linux machine/service or whole-suite Linux claim. Delivery evidence:
https://github.com/oneworkspacestudio/one-workspace/blob/main/docs/ANDROID-DELIVERY.json
