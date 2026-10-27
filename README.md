# My Push

<img src="internal/webui/static/logo.svg" alt="My Push" width="88">

Private notifications for your One Workspace apps. Link your Android phone,
check its connection and see when notification wake signals reach it.
My Chat keeps your conversations; My Push carries only an opaque sync signal.

The console brings project overview, Cloud Messaging, connected devices,
delivery activity and settings into the Workspace design system. It uses the
open source [Centrifuge](https://github.com/centrifugal/centrifuge) engine.

## Get started

Install My Push with One Workspace and open it from Apps. Workspace connects
Chat automatically, using your existing MyAccounts sign-in. Open Chat in the
Android launcher and enable background notifications. No Firebase account or
Google service key is needed for this route.

Download evaluation builds from [Releases](https://github.com/oneworkspacestudio/my-push/releases).
Use Workspace preview 94 or newer for managed installation. Physical phone sleep,
reboot and notification-action acceptance is still pending.

## What it includes

- Private project-scoped server keys and short-lived phone credentials.
- Authenticated WebSocket connections, durable wake queue and acknowledgements.
- Per-phone connection checks, disconnect controls and recent delivery activity.
- Automatic Chat integration, shared sign-in and separate protected app data.
- Windows x64 executable and Linux x64 build; see the build guide for platform status.

Android runs a visible foreground connection for self hosted delivery. Force-stop,
power restrictions and an unavailable network/server may interrupt it. Offline
phones recover messages from Chat. iOS closed-app delivery requires APNs; a
self hosted service cannot grant itself Apple's system wake privileges.

This increment implements push messaging, not the entire Firebase product family
(Firestore, Functions, Analytics, etc.). Delivery acknowledgement is not a Chat
message read receipt.

See [architecture and setup](docs/ARCHITECTURE.md) and [local build](docs/BUILD.md).

## License

Apache-2.0 for My Push, with the MIT-licensed Centrifuge engine. See LICENSE,
NOTICE and THIRD-PARTY-LICENSES.txt. No Google artwork or branding is bundled.
