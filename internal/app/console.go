package app

import (
	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
	"net/http"
	"strings"
	"time"
)

func (a *App) overview(w http.ResponseWriter, r *http.Request) {
	projects := []map[string]any{}
	rows, e := a.H.DB.Query(`SELECT p.id,p.name,p.created,p.key_hash!='',coalesce(t.accepted,0),coalesce(t.acknowledged,0) FROM push_projects p LEFT JOIN push_totals t ON t.project=p.id ORDER BY p.created,p.id`)
	if e != nil {
		apphost.Problem(w, 503, "Console unavailable")
		return
	}
	for rows.Next() {
		var id, name string
		var created, accepted, ack int64
		var configured bool
		if rows.Scan(&id, &name, &created, &configured, &accepted, &ack) != nil {
			rows.Close()
			apphost.Problem(w, 503, "Console unavailable")
			return
		}
		projects = append(projects, map[string]any{"id": id, "name": name, "created": created, "configured": configured, "accepted": accepted, "acknowledged": ack, "managed": id == "chat" && len(strings.TrimSpace(a.H.Config.WorkspaceURL)) > 0})
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		apphost.Problem(w, 503, "Console unavailable")
		return
	}
	devices := []Device{}
	rows, e = a.H.DB.Query("SELECT id,project,recipient,label,expires,seen FROM push_devices ORDER BY seen DESC LIMIT 2000")
	if e != nil {
		apphost.Problem(w, 503, "Console unavailable")
		return
	}
	for rows.Next() {
		var d Device
		if rows.Scan(&d.ID, &d.Project, &d.Recipient, &d.Label, &d.Expires, &d.Seen) != nil {
			rows.Close()
			apphost.Problem(w, 503, "Console unavailable")
			return
		}
		a.mu.Lock()
		d.Online = a.online[d.ID] > 0
		a.mu.Unlock()
		devices = append(devices, d)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		apphost.Problem(w, 503, "Console unavailable")
		return
	}
	events := []map[string]any{}
	rows, e = a.H.DB.Query("SELECT e.id,d.project,d.label,e.created,e.acked FROM push_events e JOIN push_devices d ON d.id=e.device ORDER BY e.created DESC LIMIT 100")
	if e != nil {
		apphost.Problem(w, 503, "Console unavailable")
		return
	}
	for rows.Next() {
		var id, p, label string
		var created, ack int64
		if rows.Scan(&id, &p, &label, &created, &ack) != nil {
			rows.Close()
			apphost.Problem(w, 503, "Console unavailable")
			return
		}
		events = append(events, map[string]any{"id": id, "project": p, "label": label, "created": created, "acknowledged": ack})
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		apphost.Problem(w, 503, "Console unavailable")
		return
	}
	apphost.JSON(w, 200, map[string]any{"projects": projects, "devices": devices, "events": events, "engine": "Centrifuge 0.40.1", "origin": a.H.Config.Origin, "retentionHours": 24, "now": time.Now().Unix()})
}
func (a *App) project(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if !apphost.Decode(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len(in.Name) > 80 {
		apphost.Problem(w, 400, "Choose a project name up to 80 characters")
		return
	}
	var count int
	if a.H.DB.QueryRow("SELECT count(*) FROM push_projects").Scan(&count) != nil || count >= 50 {
		apphost.Problem(w, 409, "Project limit reached")
		return
	}
	id := apphost.ID()[:16]
	key := apphost.ID() + apphost.ID()
	tx, e := a.H.DB.Begin()
	if e != nil {
		apphost.Problem(w, 503, "Could not create project")
		return
	}
	defer tx.Rollback()
	_, e = tx.Exec("INSERT INTO push_projects VALUES(?,?,?,?)", id, in.Name, apphost.Hash(key), time.Now().Unix())
	if e == nil {
		_, e = tx.Exec("INSERT INTO push_totals(project)VALUES(?)", id)
	}
	if e == nil {
		e = tx.Commit()
	}
	if e != nil {
		apphost.Problem(w, 503, "Could not create project")
		return
	}
	a.H.Audit(apphost.Current(r).User.ID, "push.project.create", id)
	apphost.JSON(w, 201, map[string]string{"id": id, "key": key})
}
func (a *App) rotate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("project")
	if id == "chat" {
		apphost.Problem(w, 409, "Workspace manages the Chat integration key automatically")
		return
	}
	key := apphost.ID() + apphost.ID()
	res, e := a.H.DB.Exec("UPDATE push_projects SET key_hash=? WHERE id=?", apphost.Hash(key), id)
	if e != nil {
		apphost.Problem(w, 503, "Could not rotate key")
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		apphost.Problem(w, 404, "Project not found")
		return
	}
	a.H.Audit(apphost.Current(r).User.ID, "push.key.rotate", id)
	apphost.JSON(w, 200, map[string]string{"id": id, "key": key})
}
