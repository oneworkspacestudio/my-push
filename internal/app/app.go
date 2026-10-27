package app

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/centrifugal/centrifuge"
	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
)

var validID = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)

type App struct {
	H      *apphost.Host
	node   *centrifuge.Node
	mu     sync.Mutex
	online map[string]int
}
type Device struct {
	ID        string `json:"id"`
	Project   string `json:"project"`
	Recipient string `json:"recipient"`
	Label     string `json:"label"`
	Expires   int64  `json:"expires"`
	Seen      int64  `json:"seen"`
	Online    bool   `json:"online"`
}
type Event struct {
	ID      string `json:"id"`
	Created int64  `json:"created"`
	What    string `json:"what"`
}

func New(h *apphost.Host) (*App, error) {
	if e := h.Migrate(1, `CREATE TABLE push_projects(id TEXT PRIMARY KEY,name TEXT NOT NULL,key_hash TEXT NOT NULL,created INTEGER NOT NULL);
 CREATE TABLE push_devices(id TEXT PRIMARY KEY,project TEXT NOT NULL REFERENCES push_projects(id),recipient TEXT NOT NULL,label TEXT NOT NULL,token_hash TEXT NOT NULL UNIQUE,expires INTEGER NOT NULL,seen INTEGER NOT NULL DEFAULT 0);
 CREATE INDEX push_recipient ON push_devices(project,recipient);
 CREATE TABLE push_events(id TEXT PRIMARY KEY,device TEXT NOT NULL REFERENCES push_devices(id) ON DELETE CASCADE,created INTEGER NOT NULL,acked INTEGER NOT NULL DEFAULT 0,what TEXT NOT NULL);
 CREATE INDEX push_queue ON push_events(device,created);
 CREATE TABLE push_totals(project TEXT PRIMARY KEY REFERENCES push_projects(id),accepted INTEGER NOT NULL DEFAULT 0,acknowledged INTEGER NOT NULL DEFAULT 0);
 CREATE TABLE push_revoked(id TEXT PRIMARY KEY,until_at INTEGER NOT NULL);
 INSERT INTO push_projects VALUES('chat','My Chat','',unixepoch()); INSERT INTO push_totals(project)VALUES('chat');`); e != nil {
		return nil, e
	}
	if key := os.Getenv("WORKSPACE_PUSH_CHAT_KEY"); key != "" {
		if len(key) < 48 {
			return nil, errors.New("managed Chat push key is invalid")
		}
		if _, e := h.DB.Exec("UPDATE push_projects SET key_hash=? WHERE id='chat'", apphost.Hash(key)); e != nil {
			return nil, e
		}
	}
	n, e := centrifuge.New(centrifuge.Config{LogLevel: centrifuge.LogLevelNone, UserConnectionLimit: 2, ClientQueueMaxSize: 64 << 10})
	if e != nil {
		return nil, e
	}
	a := &App{H: h, node: n, online: map[string]int{}}
	n.OnConnecting(func(ctx context.Context, e centrifuge.ConnectEvent) (centrifuge.ConnectReply, error) {
		d, ok := a.device(e.Token)
		if !ok {
			return centrifuge.ConnectReply{}, centrifuge.DisconnectInvalidToken
		}
		a.mu.Lock()
		count := a.online[d.ID]
		a.mu.Unlock()
		if count >= 2 {
			return centrifuge.ConnectReply{}, centrifuge.DisconnectConnectionLimit
		}
		return centrifuge.ConnectReply{Credentials: &centrifuge.Credentials{UserID: d.ID, ExpireAt: d.Expires}, Subscriptions: map[string]centrifuge.SubscribeOptions{channel(d.ID): {}}}, nil
	})
	n.OnConnect(func(c *centrifuge.Client) {
		a.mu.Lock()
		a.online[c.UserID()]++
		a.mu.Unlock()
		h.Freeze.RLock()
		_, _ = h.DB.Exec("UPDATE push_devices SET seen=? WHERE id=?", time.Now().Unix(), c.UserID())
		h.Freeze.RUnlock()
		c.OnRefresh(func(e centrifuge.RefreshEvent, cb centrifuge.RefreshCallback) {
			cb(centrifuge.RefreshReply{Expired: true}, nil)
		})
		c.OnDisconnect(func(e centrifuge.DisconnectEvent) {
			a.mu.Lock()
			a.online[c.UserID()]--
			if a.online[c.UserID()] <= 0 {
				delete(a.online, c.UserID())
			}
			a.mu.Unlock()
		})
	})
	h.Mux.Handle("GET /connection/websocket", centrifuge.NewWebsocketHandler(n, centrifuge.WebsocketConfig{CheckOrigin: func(r *http.Request) bool {
		o := r.Header.Get("Origin")
		return o == "" || o == origin(h.Config.Origin)
	}}))
	h.Mux.HandleFunc("GET /api/overview", h.Owner(a.overview))
	h.Mux.HandleFunc("POST /api/projects", h.Owner(a.project))
	h.Mux.HandleFunc("POST /api/projects/{project}/key", h.Owner(a.rotate))
	h.Mux.HandleFunc("DELETE /api/devices/{id}", h.Owner(a.remove))
	h.Mux.HandleFunc("POST /api/devices/{id}/wake", h.Owner(a.manual))
	h.Mux.HandleFunc("POST /api/service/devices", a.register)
	h.Mux.HandleFunc("DELETE /api/service/devices/{id}", a.serviceRemove)
	h.Mux.HandleFunc("POST /api/service/publish", a.publish)
	h.Mux.HandleFunc("GET /api/device/pending", a.pending)
	h.Mux.HandleFunc("POST /api/device/ack", a.ack)
	return a, nil
}
func origin(s string) string {
	u, e := url.Parse(s)
	if e != nil {
		return ""
	}
	return u.Scheme + "://" + u.Host
}
func channel(id string) string { return "device:" + id }
func (a *App) Run(ctx context.Context) error {
	if e := a.node.Run(); e != nil {
		return e
	}
	defer a.node.Shutdown(context.Background())
	tick := time.NewTicker(time.Minute)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
			a.H.Freeze.RLock()
			_, _ = a.H.DB.Exec("DELETE FROM push_events WHERE created<?", time.Now().Add(-24*time.Hour).Unix())
			_, _ = a.H.DB.Exec("DELETE FROM push_devices WHERE expires<?", time.Now().Add(-24*time.Hour).Unix())
			_, _ = a.H.DB.Exec("DELETE FROM push_revoked WHERE until_at<?", time.Now().Unix())
			a.H.Freeze.RUnlock()
		}
	}
}
func (a *App) device(token string) (Device, bool) {
	var d Device
	if len(token) != 64 {
		return d, false
	}
	e := a.H.DB.QueryRow("SELECT id,project,recipient,label,expires,seen FROM push_devices WHERE token_hash=? AND expires>?", apphost.Hash(token), time.Now().Unix()).Scan(&d.ID, &d.Project, &d.Recipient, &d.Label, &d.Expires, &d.Seen)
	return d, e == nil
}
func bearer(r *http.Request) string {
	return strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
}
func (a *App) service(w http.ResponseWriter, r *http.Request) (string, bool) {
	project := r.Header.Get("X-Push-Project")
	key := bearer(r)
	var hash string
	if !validID.MatchString(project) || len(key) < 48 || len(key) > 256 || a.H.DB.QueryRow("SELECT key_hash FROM push_projects WHERE id=?", project).Scan(&hash) != nil || hash == "" || subtle.ConstantTimeCompare([]byte(hash), []byte(apphost.Hash(key))) != 1 {
		apphost.Problem(w, 401, "Service authentication required")
		return "", false
	}
	return project, true
}
func (a *App) register(w http.ResponseWriter, r *http.Request) {
	project, ok := a.service(w, r)
	if !ok {
		return
	}
	a.H.Freeze.RLock()
	defer a.H.Freeze.RUnlock()
	var in struct {
		ID        string `json:"id"`
		Recipient string `json:"recipient"`
		Label     string `json:"label"`
		Token     string `json:"token"`
		Expires   int64  `json:"expires"`
	}
	if !apphost.Decode(w, r, &in) {
		return
	}
	if !validID.MatchString(in.ID) || !validID.MatchString(in.Recipient) || len(in.Token) != 64 || len(in.Label) > 80 || in.Expires <= time.Now().Unix() || in.Expires > time.Now().Add(35*time.Minute).Unix() {
		apphost.Problem(w, 400, "Invalid scoped device registration")
		return
	}
	id := project + "_" + in.ID
	var blocked int
	if a.H.DB.QueryRow("SELECT count(*) FROM push_revoked WHERE id=? AND until_at>?", id, time.Now().Unix()).Scan(&blocked) != nil || blocked > 0 {
		apphost.Problem(w, 403, "This phone was disconnected. Relink it in Chat.")
		return
	}
	var count int
	if a.H.DB.QueryRow("SELECT count(*) FROM push_devices WHERE project=? AND expires>?", project, time.Now().Unix()).Scan(&count) != nil {
		apphost.Problem(w, 503, "Device storage unavailable")
		return
	}
	if count >= 2000 {
		var exists int
		if a.H.DB.QueryRow("SELECT count(*) FROM push_devices WHERE id=?", id).Scan(&exists) != nil || exists == 0 {
			apphost.Problem(w, 429, "Project device limit reached")
			return
		}
	}
	_, e := a.H.DB.Exec(`INSERT INTO push_devices(id,project,recipient,label,token_hash,expires)VALUES(?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET recipient=excluded.recipient,label=excluded.label,token_hash=excluded.token_hash,expires=excluded.expires`, id, project, in.Recipient, in.Label, apphost.Hash(in.Token), in.Expires)
	if e != nil {
		apphost.Problem(w, 503, "Device registration could not be stored")
		return
	}
	apphost.JSON(w, 200, map[string]any{"provider": "workspace", "url": a.H.Config.Origin, "device": id, "expires": in.Expires})
}
func (a *App) delete(id string) error {
	tx, e := a.H.DB.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.Exec("INSERT OR REPLACE INTO push_revoked(id,until_at)VALUES(?,?)", id, time.Now().Add(31*24*time.Hour).Unix()); e != nil {
		return e
	}
	if _, e = tx.Exec("DELETE FROM push_devices WHERE id=?", id); e != nil {
		return e
	}
	if e = tx.Commit(); e != nil {
		return e
	}
	_ = a.node.Disconnect(id, centrifuge.WithCustomDisconnect(centrifuge.DisconnectForceNoReconnect))
	return nil
}
func (a *App) serviceRemove(w http.ResponseWriter, r *http.Request) {
	p, ok := a.service(w, r)
	if !ok {
		return
	}
	a.H.Freeze.RLock()
	defer a.H.Freeze.RUnlock()
	if !validID.MatchString(r.PathValue("id")) {
		apphost.Problem(w, 400, "Invalid device")
		return
	}
	if a.delete(p+"_"+r.PathValue("id")) != nil {
		apphost.Problem(w, 503, "Could not disconnect device")
		return
	}
	w.WriteHeader(204)
}
func (a *App) remove(w http.ResponseWriter, r *http.Request) {
	if a.delete(r.PathValue("id")) != nil {
		apphost.Problem(w, 503, "Could not disconnect device")
		return
	}
	a.H.Audit(apphost.Current(r).User.ID, "push.device.remove", r.PathValue("id"))
	w.WriteHeader(204)
}
func (a *App) queue(id, project string, what string) (string, error) {
	event := Event{apphost.ID(), time.Now().Unix(), what}
	tx, e := a.H.DB.Begin()
	if e != nil {
		return "", e
	}
	defer tx.Rollback()
	// Coalesce pending wake signals. Chat, not this queue, owns message history.
	_, e = tx.Exec("DELETE FROM push_events WHERE device=? AND acked=0", id)
	if e != nil {
		return "", e
	}
	_, e = tx.Exec("INSERT INTO push_events(id,device,created,what)VALUES(?,?,?,?)", event.ID, id, event.Created, event.What)
	if e != nil {
		return "", e
	}
	_, e = tx.Exec("UPDATE push_totals SET accepted=accepted+1 WHERE project=?", project)
	if e != nil {
		return "", e
	}
	if e = tx.Commit(); e != nil {
		return "", e
	}
	b, _ := json.Marshal(event)
	_, _ = a.node.Publish(channel(id), b)
	return event.ID, nil
}
func (a *App) publish(w http.ResponseWriter, r *http.Request) {
	p, ok := a.service(w, r)
	if !ok {
		return
	}
	a.H.Freeze.RLock()
	defer a.H.Freeze.RUnlock()
	var in struct {
		Recipient string `json:"recipient"`
	}
	if !apphost.Decode(w, r, &in) {
		return
	}
	if !validID.MatchString(in.Recipient) {
		apphost.Problem(w, 400, "Invalid recipient")
		return
	}
	rows, e := a.H.DB.Query("SELECT id FROM push_devices WHERE project=? AND recipient=? AND expires>?", p, in.Recipient, time.Now().Unix())
	if e != nil {
		apphost.Problem(w, 503, "Queue unavailable")
		return
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) != nil {
			rows.Close()
			apphost.Problem(w, 503, "Queue unavailable")
			return
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		apphost.Problem(w, 503, "Queue unavailable")
		return
	}
	accepted := 0
	for _, id := range ids {
		if _, e = a.queue(id, p, "sync"); e != nil {
			apphost.Problem(w, 503, "Some wake signals could not be stored; retry safely")
			return
		}
		accepted++
	}
	apphost.JSON(w, 202, map[string]any{"accepted": accepted, "status": "queued"})
}
func (a *App) manual(w http.ResponseWriter, r *http.Request) {
	var p string
	if a.H.DB.QueryRow("SELECT project FROM push_devices WHERE id=? AND expires>?", r.PathValue("id"), time.Now().Unix()).Scan(&p) != nil {
		apphost.Problem(w, 404, "Device is expired or disconnected")
		return
	}
	id, e := a.queue(r.PathValue("id"), p, "check")
	if e != nil {
		apphost.Problem(w, 503, "Could not queue notification check")
		return
	}
	a.H.Audit(apphost.Current(r).User.ID, "push.wake", r.PathValue("id"))
	apphost.JSON(w, 202, map[string]any{"id": id, "status": "queued"})
}
func (a *App) pending(w http.ResponseWriter, r *http.Request) {
	d, ok := a.device(bearer(r))
	if !ok {
		apphost.Problem(w, 401, "Reconnect through your app")
		return
	}
	a.H.Freeze.RLock()
	defer a.H.Freeze.RUnlock()
	rows, e := a.H.DB.Query("SELECT id,created,what FROM push_events WHERE device=? AND acked=0 AND created>? ORDER BY created LIMIT 100", d.ID, time.Now().Add(-24*time.Hour).Unix())
	if e != nil {
		apphost.Problem(w, 503, "Queue unavailable")
		return
	}
	defer rows.Close()
	items := []Event{}
	for rows.Next() {
		var x Event
		if rows.Scan(&x.ID, &x.Created, &x.What) != nil {
			apphost.Problem(w, 503, "Queue unavailable")
			return
		}
		items = append(items, x)
	}
	if rows.Err() != nil {
		apphost.Problem(w, 503, "Queue unavailable")
		return
	}
	apphost.JSON(w, 200, map[string]any{"events": items})
}
func (a *App) ack(w http.ResponseWriter, r *http.Request) {
	d, ok := a.device(bearer(r))
	if !ok {
		apphost.Problem(w, 401, "Reconnect through your app")
		return
	}
	a.H.Freeze.RLock()
	defer a.H.Freeze.RUnlock()
	var in struct {
		ID string `json:"id"`
	}
	if !apphost.Decode(w, r, &in) {
		return
	}
	tx, e := a.H.DB.Begin()
	if e != nil {
		apphost.Problem(w, 503, "Queue unavailable")
		return
	}
	defer tx.Rollback()
	res, e := tx.Exec("UPDATE push_events SET acked=? WHERE id=? AND device=? AND acked=0", time.Now().Unix(), in.ID, d.ID)
	if e != nil {
		apphost.Problem(w, 503, "Queue unavailable")
		return
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		_, e = tx.Exec("UPDATE push_totals SET acknowledged=acknowledged+1 WHERE project=?", d.Project)
	}
	if e == nil {
		_, e = tx.Exec("UPDATE push_devices SET seen=? WHERE id=?", time.Now().Unix(), d.ID)
	}
	if e == nil {
		e = tx.Commit()
	}
	if e != nil {
		apphost.Problem(w, 503, "Acknowledgement not saved")
		return
	}
	w.WriteHeader(204)
}
