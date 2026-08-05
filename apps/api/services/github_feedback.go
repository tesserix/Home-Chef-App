package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

// In-app "Send feedback" / "Share an idea" from the customer and vendor apps,
// filed straight onto the product backlog as a labelled GitHub issue. Nothing is
// stored here — the issue IS the record, so triage happens where the work does.

const (
	FeedbackKindFeedback = "feedback"
	FeedbackKindIdea     = "idea"

	// The backlog labels product triages on.
	FeedbackLabel = "USER FEEDBACK"
	IdeaLabel     = "USER IDEA"

	feedbackTitleMaxLen   = 120
	feedbackMessageMaxLen = 5000
)

// ErrFeedbackNotConfigured means no GitHub token is present, so the feature is
// off in this environment rather than broken.
var ErrFeedbackNotConfigured = errors.New("feedback: GitHub token not configured")

// FeedbackSubmission is one report from one app.
type FeedbackSubmission struct {
	Kind       string
	Title      string
	Message    string
	App        string // customer | vendor
	Platform   string // ios | android
	AppVersion string
	UserID     string
	// UserEmail is used for de-duplication and support follow-up only; it is
	// deliberately never written into the issue body (the repo is not a PII store).
	UserEmail string
}

// FeedbackResult points at the filed issue.
type FeedbackResult struct {
	URL    string `json:"url"`
	Number int    `json:"number"`
}

type GitHubFeedbackClient struct {
	Token   string
	Repo    string
	BaseURL string
	HTTP    *http.Client
}

// NewGitHubFeedbackClient reads the environment. Repo defaults to this product's
// backlog so a deployment only has to supply the token.
func NewGitHubFeedbackClient() *GitHubFeedbackClient {
	return &GitHubFeedbackClient{
		Token:   os.Getenv("GITHUB_FEEDBACK_TOKEN"),
		Repo:    envOrDefault("GITHUB_FEEDBACK_REPO", "tesserix/Home-Chef-App"),
		BaseURL: envOrDefault("GITHUB_API_URL", "https://api.github.com"),
		HTTP:    &http.Client{Timeout: 15 * time.Second},
	}
}

func envOrDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func (c *GitHubFeedbackClient) Configured() bool {
	return strings.TrimSpace(c.Token) != ""
}

func (c *GitHubFeedbackClient) Submit(ctx context.Context, sub FeedbackSubmission) (FeedbackResult, error) {
	if !c.Configured() {
		return FeedbackResult{}, ErrFeedbackNotConfigured
	}

	payload := map[string]any{
		"title":  feedbackIssueTitle(sub),
		"body":   feedbackIssueBody(sub),
		"labels": feedbackIssueLabels(sub),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return FeedbackResult{}, err
	}

	url := fmt.Sprintf("%s/repos/%s/issues", strings.TrimSuffix(c.BaseURL, "/"), c.Repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return FeedbackResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return FeedbackResult{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return FeedbackResult{}, fmt.Errorf("feedback: github responded %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out struct {
		HTMLURL string `json:"html_url"`
		Number  int    `json:"number"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return FeedbackResult{}, err
	}
	return FeedbackResult{URL: out.HTMLURL, Number: out.Number}, nil
}

func feedbackIssueLabels(sub FeedbackSubmission) []string {
	label := FeedbackLabel
	if sub.Kind == FeedbackKindIdea {
		label = IdeaLabel
	}
	app := strings.TrimSpace(sub.App)
	if app == "" {
		app = "unknown"
	}
	return []string{label, "app: " + app}
}

func feedbackIssueTitle(sub FeedbackSubmission) string {
	prefix := "[Feedback] "
	if sub.Kind == FeedbackKindIdea {
		prefix = "[Idea] "
	}
	title := strings.TrimSpace(sub.Title)
	// Budget in bytes (GitHub's own limit) and cut on a rune boundary so a
	// truncated Hindi or emoji title never becomes mojibake.
	if budget := feedbackTitleMaxLen - len(prefix) - len("…"); len(title) > budget {
		cut := title[:budget]
		for len(cut) > 0 && !utf8.ValidString(cut) {
			cut = cut[:len(cut)-1]
		}
		title = strings.TrimSpace(cut) + "…"
	}
	return prefix + title
}

func feedbackIssueBody(sub FeedbackSubmission) string {
	message := strings.TrimSpace(sub.Message)
	if len(message) > feedbackMessageMaxLen {
		message = message[:feedbackMessageMaxLen]
	}
	var b strings.Builder
	b.WriteString(message)
	b.WriteString("\n\n---\n\n")
	b.WriteString("| | |\n|---|---|\n")
	fmt.Fprintf(&b, "| App | %s |\n", strOrDefault(sub.App, "unknown"))
	fmt.Fprintf(&b, "| Platform | %s |\n", strOrDefault(sub.Platform, "unknown"))
	fmt.Fprintf(&b, "| Version | %s |\n", strOrDefault(sub.AppVersion, "unknown"))
	fmt.Fprintf(&b, "| Reported by | user `%s` |\n", strOrDefault(sub.UserID, "anonymous"))
	fmt.Fprintf(&b, "| Received | %s |\n", time.Now().UTC().Format(time.RFC3339))
	b.WriteString("\nFiled from in-app feedback. Contact details are held in the app database, not here.\n")
	return b.String()
}
