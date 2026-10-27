# Local build

Use Go 1.27.3. Dependencies are pinned to public source in go.mod/go.sum, including
the matching Workspace apphost integration and Linux permission helpers. No sibling
checkout or local module replacement is required.

```powershell
./scripts/build.ps1 -Go /path/to/go.exe -Out I:/Projects/tooling/my-push
```

The app compiles to Windows x64 and Linux x64 with CGO disabled. Linux private
state must be on a local Linux filesystem with Unix ownership/permissions; do
not copy Windows DPAPI state into it. Linux machine/service acceptance is pending.
No whole-suite Linux runtime compatibility claim follows from an app cross-build.

For standalone use, launch with `--data` pointing to a new, dedicated private-data
folder. On Windows this must be NTFS. The app prints a one-time local setup URL.
Managed use gets MyAccounts automatically and needs no separate password.
The console's Chat project is connected by the Workspace manager. Other projects
can be created in the console; their service key is shown only once.

Build the Android source with the Workspace `scripts/build-android.ps1`. A debug
APK is for evaluation; release signing requires a permanent private signing key.
Build matching Chat source with its local `scripts/workspace/build.py` or the
existing runtime payload and Go `postgres,workspace` tags. Never deploy stale
runtime ZIPs or publish a source-dirty development binary as an immutable release.

My Push's `compatibility.json` and generated `workspace-update.json` specify
the integration contract. The app manifest requires the new manager capability
`self-hosted-push-v1`; a current old manager must not install it prematurely.
The full bundle accepts MyPush payload/notices through `scripts/build.ps1`.
Organization builds require an explicit local package or a verified public pin.
