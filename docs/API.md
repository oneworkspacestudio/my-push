# Service and device API

Service requests require `Authorization: Bearer SERVICE_KEY` and
`X-Push-Project: PROJECT_ID`. Keep that key on the app server. Managed applications
use the loopback endpoint automatically. The managed public proxy does not expose
service APIs without a Workspace session. Device transport uses its own bearer.

`POST /api/service/devices` registers/renews a device:

```json
{"id":"random-device-id","recipient":"opaque-account-id","label":"Android phone","token":"64-character-random-device-token","expires":1900000000}
```

The app server must generate the 64-character token securely and bind recipient
and device to its authenticated account. Expiry must be in the next35 minutes;
Chat grants30. ID and recipient use letters, digits, `_` or `-`, max128 characters.
Returns the public URL and project-scoped device ID. The app server passes only
the device token to that phone; the service key never leaves the app server.

`POST /api/service/publish` with `{"recipient":"opaque-account-id"}` stores
opaque wakes for active phones and returns202 with accepted count. Zero means no
active registered phones. This is not an acknowledgement or a message read receipt.

`DELETE /api/service/devices/DEVICE_ID` blocks/removes that project's device.

Phones send Centrifuge's JSON connect command to `/connection/websocket`:

```json
{"id":1,"connect":{"token":"DEVICE_TOKEN","name":"your-app"}}
```

The server selects the private device channel. No arbitrary subscribe/publish
commands are authorized. Reply to a protocol ping `{}` with `{}`. Publications
contain only `id`, `created` and `what` (`sync` or a console connection `check`).

`GET /api/device/pending` returns `{"events":[...]}` using the same device bearer.
`POST /api/device/ack` accepts `{"id":"EVENT_ID"}` and acknowledges only that
device's event. Duplicate acknowledgements are safe. The phone fetches message
content from its actual app after a wake; this service never stores Chat messages.
