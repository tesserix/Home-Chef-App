package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFeedbackIssueLabelsByKind(t *testing.T) {
	cases := []struct {
		kind  string
		app   string
		want  string
		label string
	}{
		{FeedbackKindFeedback, "customer", "USER FEEDBACK", "app: customer"},
		{FeedbackKindIdea, "vendor", "USER IDEA", "app: vendor"},
	}
	for _, tc := range cases {
		got := feedbackIssueLabels(FeedbackSubmission{Kind: tc.kind, App: tc.app})
		if len(got) != 2 || got[0] != tc.want || got[1] != tc.label {
			t.Fatalf("labels for %s/%s = %v", tc.kind, tc.app, got)
		}
	}
}

func TestFeedbackIssueTitleIsPrefixedAndTrimmed(t *testing.T) {
	title := feedbackIssueTitle(FeedbackSubmission{Kind: FeedbackKindIdea, Title: strings.Repeat("a", 200)})
	if !strings.HasPrefix(title, "[Idea] ") {
		t.Fatalf("title not prefixed: %q", title)
	}
	if len(title) > 120 {
		t.Fatalf("title not capped: %d chars", len(title))
	}
}

func TestFeedbackIssueBodyCarriesContextButNoEmail(t *testing.T) {
	body := feedbackIssueBody(FeedbackSubmission{
		Kind:       FeedbackKindFeedback,
		Message:    "The receipt is hard to read",
		App:        "customer",
		Platform:   "ios",
		AppVersion: "1.4.0",
		UserID:     "8f14e45f-ea1b-4b2c-9c1e-000000000001",
		UserEmail:  "someone@example.com",
	})
	for _, want := range []string{"The receipt is hard to read", "customer", "ios", "1.4.0", "8f14e45f"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "someone@example.com") {
		t.Fatalf("body leaked the reporter's email:\n%s", body)
	}
}

func TestSubmitFeedbackIssuePostsToRepoAndReturnsURL(t *testing.T) {
	var gotPath, gotAuth string
	var payload map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &payload)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"html_url":"https://github.com/tesserix/Home-Chef-App/issues/7","number":7}`))
	}))
	defer srv.Close()

	c := &GitHubFeedbackClient{Token: "tok", Repo: "tesserix/Home-Chef-App", BaseURL: srv.URL, HTTP: srv.Client()}
	res, err := c.Submit(context.Background(), FeedbackSubmission{
		Kind: FeedbackKindIdea, Title: "Dark mode", Message: "Please add dark mode", App: "vendor",
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if res.Number != 7 || res.URL == "" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if gotPath != "/repos/tesserix/Home-Chef-App/issues" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotAuth != "Bearer tok" {
		t.Fatalf("auth = %q", gotAuth)
	}
	labels, _ := payload["labels"].([]any)
	if len(labels) != 2 || labels[0] != "USER IDEA" {
		t.Fatalf("labels = %v", payload["labels"])
	}
}

func TestSubmitFeedbackIssueSurfacesAPIFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"Bad credentials"}`))
	}))
	defer srv.Close()

	c := &GitHubFeedbackClient{Token: "tok", Repo: "tesserix/Home-Chef-App", BaseURL: srv.URL, HTTP: srv.Client()}
	if _, err := c.Submit(context.Background(), FeedbackSubmission{Kind: FeedbackKindFeedback, Title: "x", Message: "y"}); err == nil {
		t.Fatal("expected an error for a 401 response")
	}
}

func TestSubmitFeedbackIssueRequiresConfiguration(t *testing.T) {
	c := &GitHubFeedbackClient{Repo: "tesserix/Home-Chef-App"}
	if _, err := c.Submit(context.Background(), FeedbackSubmission{Kind: FeedbackKindFeedback, Title: "x", Message: "y"}); err != ErrFeedbackNotConfigured {
		t.Fatalf("err = %v, want ErrFeedbackNotConfigured", err)
	}
}
