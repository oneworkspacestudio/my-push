# Architecture and setup

## Console research

The Firebase project/app hierarchy and Cloud Messaging console informed the
project selector, overview, messaging section, delivery activity and settings:

- https://firebase.google.com/docs/projects/learn-more
- https://firebase.google.com/docs/cloud-messaging/send/firebase-console
- https://firebase.google.com/docs/cloud-messaging/understand-delivery

The implementation uses original Workspace branding and existing blue/mint
palette. It does not copy Google's artwork or claim Firebase SDK compatibility.

ntfy/UnifiedPush were also considered. Centrifuge v0.40.1 was selected as an
embedded Go real-time engine for standalone builds without a separate push
daemon or Google account. Its upstream source and MIT license are preserved.

## Flow

1. Workspace launches My Push with its own MyAccounts client, owner identity and
   a Chat-scoped service capability derived from the existing protected Chat
   client secret. The capability is never sent to a browser or phone.
2. Chat authenticates a linked native phone and asks loopback My Push to register
   that device/recipient. Chat derives a separate token from its phone credential.
   My Push stores only token hashes, and grants at most 30 minutes of access.
3. Android receives the HTTPS My Push endpoint and scoped credential through the
   authenticated native Chat bootstrap, stores it using Keystore and connects
   through Centrifuge's authenticated WebSocket protocol. The server selects the
   single device channel. Client publications and arbitrary subscriptions are
   disabled by the engine's default-deny handlers.
4. Tinode's authorized recipient list drives the My Push plugin. Only new,
   non-silent message receipts enqueue a wake. The loopback publish body contains
   only the recipient ID. A phone receives only random event ID, timestamp and
   `sync` (or an owner connection `check`), never message/sender/topic contents.
5. Android displays a private generic alert, schedules the existing authenticated
   Chat sync and acknowledges the wake. Chat's normal membership checks still
   govern message access and native reply/read actions.

## Data and controls

Apphost stores SQLite state, hashes, audit records and queue in the dedicated
`${DATA}/push` private-data folder, separate from program files and user files.
SQLite writes use apphost's database and update-backup freeze lock. Recent queue
entries expire after 24 hours. New pending signals coalesce per phone; Chat holds
the complete durable conversation history. Old device entries and revocation
tombstones are bounded by retention. There are 50 projects and 2,000 active
devices/project. Service credentials are per-project; device credentials cannot
publish, list other devices, use the console or select another channel.

Owner console writes require normal MyAccounts owner role, same-origin and
CSRF checks. Project keys are shown once and stored hashed. Chat's managed key
cannot be replaced through an ordinary project-key action. Phone disconnects
close live transports and leave a tombstone so the old grant cannot immediately
re-register. Relink in Chat to obtain a different native-device identity.
Chat revocation also requests push deletion; if unreachable, push access expires
within 30 minutes and cannot read any conversation content.

The Workspace proxy exempts only the three phone transport endpoints from its
portal cookie; My Push still authenticates them. Service endpoints and console
APIs require normal portal access externally, and service authentication inside.
The HTTPS tunnel can carry this WebSocket connection. No public router port is
needed. Disabling My Push does not stop Chat; periodic native Chat sync remains.

## Android operating-system limits

The self hosted route has no Firebase Android SDK or Google account dependency.
Android uses a visible `remoteMessaging` foreground service and WorkManager
fallback. It reconnects with capped jittered backoff, recovers pending wakes,
deduplicates events, refreshes native grants, and ties every action to the current
linked device. Reboot restoration is attempted only for an already enabled
connection and when Android permits it. A force-stop, denied notification
permission, battery restrictions or no network can interrupt delivery. No app
server can grant itself an Android/iOS system push privilege.

References:
- https://developer.android.com/training/monitoring-device-state/doze-standby
- https://developer.android.com/develop/background-work/services/fgs/service-types
- https://docs.ntfy.sh/subscribe/phone/#instant-delivery
- https://unifiedpush.org/users/distributors/

This increment is the notification service, not Firebase's database, functions,
analytics or hosting products. It does not implement native iOS/APNs. Physical
device sleep/reboot/notification/reply acceptance remains pending. Automated tests
cover recipient isolation, expiry, coalescing, acknowledgements and revocation.
An isolated HTTPS Chat/Accounts fixture verifies sign-in, native linking, public
WebSocket wake delivery, durable queue/ACK and immediate revocation. Owner data
was not used as a fixture.
