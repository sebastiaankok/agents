package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(t *testing.T, handler http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := NewClient("test-token")
	c.baseURL = srv.URL
	return c, srv
}

func TestGetReadyIssues(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/issues" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("labels") != "ready-for-agent" {
			t.Errorf("missing label filter: %s", r.URL.RawQuery)
		}
		if r.URL.Query().Get("state") != "open" {
			t.Errorf("missing state filter")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"number":1,"title":"First","body":"do this","state":"open"},{"number":2,"title":"Second","body":"do that","state":"open"}]`))
	})

	c, _ := newTestClient(t, handler)
	issues, err := c.GetReadyIssues(context.Background(), "owner/repo")
	if err != nil {
		t.Fatalf("GetReadyIssues() error = %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("got %d issues, want 2", len(issues))
	}
	if issues[0].Number != 1 || issues[0].Title != "First" {
		t.Errorf("unexpected first issue: %+v", issues[0])
	}
}

func TestGetIssue(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/issues/42" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"number":42,"title":"Fix bug","body":"Blocked by #3","state":"open"}`))
	})

	c, _ := newTestClient(t, handler)
	issue, err := c.GetIssue(context.Background(), "owner/repo", 42)
	if err != nil {
		t.Fatalf("GetIssue() error = %v", err)
	}
	if issue.Number != 42 || issue.Title != "Fix bug" {
		t.Errorf("unexpected issue: %+v", issue)
	}
}

func TestIsIssueClosed(t *testing.T) {
	tests := []struct {
		name  string
		state string
		want  bool
	}{
		{"closed issue", "closed", true},
		{"open issue", "open", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"number":1,"title":"t","body":"","state":"` + tt.state + `"}`))
			})
			c, _ := newTestClient(t, handler)
			got, err := c.IsIssueClosed(context.Background(), "owner/repo", 1)
			if err != nil {
				t.Fatalf("IsIssueClosed() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("IsIssueClosed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPRByBranch_Found(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/pulls" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("head") != "owner:agent/issue-7" {
			t.Errorf("unexpected head param: %s", r.URL.Query().Get("head"))
		}
		if r.URL.Query().Get("state") != "open" {
			t.Errorf("unexpected state param: %s", r.URL.Query().Get("state"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"number":42,"node_id":"PR_node42","merged":false}]`))
	})

	c, _ := newTestClient(t, handler)
	pr, ok, err := c.GetPRByBranch(context.Background(), "owner/repo", "agent/issue-7")
	if err != nil {
		t.Fatalf("GetPRByBranch() error = %v", err)
	}
	if !ok {
		t.Fatal("GetPRByBranch() ok = false, want true")
	}
	if pr.Number != 42 {
		t.Errorf("PR.Number = %d, want 42", pr.Number)
	}
	if pr.NodeID != "PR_node42" {
		t.Errorf("PR.NodeID = %q, want %q", pr.NodeID, "PR_node42")
	}
}

func TestGetPRByBranch_NotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	})

	c, _ := newTestClient(t, handler)
	_, ok, err := c.GetPRByBranch(context.Background(), "owner/repo", "agent/issue-99")
	if err != nil {
		t.Fatalf("GetPRByBranch() error = %v", err)
	}
	if ok {
		t.Error("GetPRByBranch() ok = true, want false for empty result")
	}
}

func TestEnableAutoMerge(t *testing.T) {
	var graphqlCalled bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/pulls/42":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"number":42,"node_id":"PR_node42","merged":false}`))
		case r.Method == http.MethodPost && r.URL.Path == "/graphql":
			graphqlCalled = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"enablePullRequestAutoMerge":{"pullRequest":{"id":"PR_node42"}}}}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})

	c, _ := newTestClient(t, handler)
	err := c.EnableAutoMerge(context.Background(), "owner/repo", 42)
	if err != nil {
		t.Fatalf("EnableAutoMerge() error = %v", err)
	}
	if !graphqlCalled {
		t.Error("GraphQL endpoint not called")
	}
}

func TestUpdateLabel(t *testing.T) {
	var addCalled, removeCalled bool

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/repos/owner/repo/issues/5/labels":
			addCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodDelete && r.URL.Path == "/repos/owner/repo/issues/5/labels/old-label":
			removeCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})

	c, _ := newTestClient(t, handler)
	err := c.UpdateLabel(context.Background(), "owner/repo", 5, "new-label", "old-label")
	if err != nil {
		t.Fatalf("UpdateLabel() error = %v", err)
	}
	if !addCalled {
		t.Error("add label endpoint not called")
	}
	if !removeCalled {
		t.Error("remove label endpoint not called")
	}
}
