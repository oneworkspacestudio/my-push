package app

import (
	"context"
	"encoding/json"
	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestScopedWakeQueueAcknowledgementAndRevocation(t *testing.T) {
	key := strings.Repeat("k", 64)
	t.Setenv("WORKSPACE_PUSH_CHAT_KEY", key)
	h, e := apphost.New(apphost.Config{ID: "push", Name: "Push test", Version: "test", Data: t.TempDir(), Listen: "127.0.0.1:41919", Origin: "http://127.0.0.1:41919"}, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer h.Close()
	a, e := New(h)
	if e != nil {
		t.Fatal(e)
	}
	if e = a.node.Run(); e != nil {
		t.Fatal(e)
	}
	defer a.node.Shutdown(context.Background())
	call := func(method, path, token, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, h.Config.Origin+path, strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("X-Push-Project", "chat")
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.Handler().ServeHTTP(w, r)
		return w
	}
	expires := time.Now().Add(30 * time.Minute).Unix()
	registration := func(id, recipient, token string) string {
		b, _ := json.Marshal(map[string]any{"id": id, "recipient": recipient, "token": token, "label": "Synthetic phone", "expires": expires})
		return string(b)
	}
	alice := strings.Repeat("a", 64)
	bob := strings.Repeat("b", 64)
	if w := call("POST", "/api/service/devices", "", registration("alice", "usrAlice", alice)); w.Code != 401 {
		t.Fatal("anonymous registration accepted", w.Code)
	}
	for _, d := range []struct{ id, recipient, token string }{{"alice", "usrAlice", alice}, {"bob", "usrBob", bob}} {
		if w := call("POST", "/api/service/devices", key, registration(d.id, d.recipient, d.token)); w.Code != 200 {
			t.Fatal("registration failed", w.Code, w.Body.String())
		}
	}
	if w := call("POST", "/api/service/publish", alice, `{"recipient":"usrBob"}`); w.Code != 401 {
		t.Fatal("device could publish", w.Code)
	}
	if w := call("POST", "/api/service/publish", key, `{"recipient":"usrAlice"}`); w.Code != 202 {
		t.Fatal("publish failed", w.Code)
	}
	var pending struct {
		Events []Event `json:"events"`
	}
	w := call("GET", "/api/device/pending", alice, "")
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &pending) != nil || len(pending.Events) != 1 {
		t.Fatal("wake not retained")
	}
	id := pending.Events[0].ID
	if strings.Contains(w.Body.String(), "usrAlice") || strings.Contains(w.Body.String(), "token") {
		t.Fatal("private routing/credential leaked")
	}
	w = call("GET", "/api/device/pending", bob, "")
	var other struct {
		Events []Event `json:"events"`
	}
	if json.Unmarshal(w.Body.Bytes(), &other) != nil || len(other.Events) != 0 {
		t.Fatal("cross-account wake exposed")
	}
	if w = call("POST", "/api/device/ack", bob, `{"id":"`+id+`"}`); w.Code != 204 {
		t.Fatal("foreign ack response")
	}
	var n int
	if e = h.DB.QueryRow("SELECT count(*) FROM push_events WHERE id=? AND acked=0", id).Scan(&n); e != nil || n != 1 {
		t.Fatal("foreign device acknowledged wake")
	}
	for i := 0; i < 2; i++ {
		if w = call("POST", "/api/device/ack", alice, `{"id":"`+id+`"}`); w.Code != 204 {
			t.Fatal("ack failed")
		}
	}
	if e = h.DB.QueryRow("SELECT acknowledged FROM push_totals WHERE project='chat'").Scan(&n); e != nil || n != 1 {
		t.Fatal("duplicate ack counted")
	}
	for i := 0; i < 3; i++ {
		call("POST", "/api/service/publish", key, `{"recipient":"usrAlice"}`)
	}
	if e = h.DB.QueryRow("SELECT count(*) FROM push_events WHERE device='chat_alice' AND acked=0").Scan(&n); e != nil || n != 1 {
		t.Fatal("pending wakes did not coalesce")
	}
	if w = call("DELETE", "/api/service/devices/alice", key, ""); w.Code != 204 {
		t.Fatal("revoke failed")
	}
	if w = call("GET", "/api/device/pending", alice, ""); w.Code != 401 {
		t.Fatal("revoked token accepted")
	}
	if w = call("POST", "/api/service/devices", key, registration("alice", "usrAlice", alice)); w.Code != 403 {
		t.Fatal("revoked identity re-registered")
	}
	_, _ = h.DB.Exec("UPDATE push_devices SET expires=? WHERE id='chat_bob'", time.Now().Add(-time.Minute).Unix())
	if w = call("GET", "/api/device/pending", bob, ""); w.Code != 401 {
		t.Fatal("expired token accepted")
	}
	r := httptest.NewRequest(http.MethodGet, h.Config.Origin+"/api/overview", nil)
	w = httptest.NewRecorder()
	h.Handler().ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatal("anonymous console exposed")
	}
}
