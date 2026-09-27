package appsecrets

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

var ErrNotFound = errors.New("app secret not found")
var secretName = regexp.MustCompile(`^prod-homechef-[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Client struct {
	address, role, tokenFile string
	http                     *http.Client
	mu                       sync.Mutex
	token                    string
	expires                  time.Time
}

func New(address, role, tokenFile string) (*Client, error) {
	u, err := url.Parse(address)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || role == "" || tokenFile == "" {
		return nil, errors.New("invalid OpenBao client configuration")
	}
	return &Client{address: strings.TrimRight(address, "/"), role: role, tokenFile: tokenFile, http: &http.Client{
		Timeout:       10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("OpenBao redirects refused") },
	}}, nil
}

func FromEnvironment() (*Client, error) {
	return fromEnvironment("APP_SECRET_STORE", "OPENBAO_ROLE")
}

func PIIFromEnvironment() (*Client, error) {
	return fromEnvironment("PII_SECRET_STORE", "OPENBAO_PII_ROLE")
}

func fromEnvironment(selector, roleVariable string) (*Client, error) {
	switch os.Getenv(selector) {
	case "", "gcp":
		return nil, nil
	case "openbao":
		return New(os.Getenv("OPENBAO_ADDR"), os.Getenv(roleVariable), "/var/run/secrets/kubernetes.io/serviceaccount/token")
	default:
		return nil, errors.New("unknown secret store selector")
	}
}

func secretPath(name string) (string, error) {
	if !secretName.MatchString(name) {
		return "", errors.New("secret outside production fe3dr scope")
	}
	return "homechef/homechef-api/fe3dr-" + strings.TrimPrefix(name, "prod-homechef-"), nil
}

func (c *Client) request(ctx context.Context, method, path, token string, body any, output any) error {
	var encoded []byte
	if body != nil {
		var err error
		encoded, err = json.Marshal(body)
		if err != nil {
			return errors.New("encode OpenBao request")
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.address+"/v1/"+path, bytes.NewReader(encoded))
	if err != nil {
		return errors.New("build OpenBao request")
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-Vault-Token", token)
	}
	response, err := c.http.Do(req)
	if err != nil {
		return errors.New("OpenBao request unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("OpenBao request status %d", response.StatusCode)
	}
	if output != nil {
		if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(output); err != nil {
			return errors.New("decode OpenBao response")
		}
	}
	return nil
}

func (c *Client) authenticate(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.expires) {
		return c.token, nil
	}
	jwt, err := os.ReadFile(c.tokenFile)
	if err != nil {
		return "", errors.New("read Kubernetes identity token")
	}
	var response struct {
		Auth struct {
			Token string `json:"client_token"`
			TTL   int    `json:"lease_duration"`
		} `json:"auth"`
	}
	if err := c.request(ctx, http.MethodPost, "auth/kubernetes/login", "", map[string]string{"role": c.role, "jwt": strings.TrimSpace(string(jwt))}, &response); err != nil {
		return "", err
	}
	if response.Auth.Token == "" || response.Auth.TTL <= 0 {
		return "", errors.New("invalid OpenBao login response")
	}
	c.token = response.Auth.Token
	c.expires = time.Now().Add(time.Duration(response.Auth.TTL) * time.Second * 4 / 5)
	return c.token, nil
}

func (c *Client) Read(ctx context.Context, name string) (string, error) {
	path, err := secretPath(name)
	if err != nil {
		return "", err
	}
	token, err := c.authenticate(ctx)
	if err != nil {
		return "", err
	}
	var response struct {
		Data struct {
			Data map[string]string `json:"data"`
		} `json:"data"`
	}
	if err := c.request(ctx, http.MethodGet, "kv/data/"+path, token, nil, &response); err != nil {
		return "", err
	}
	value, ok := response.Data.Data["value"]
	if !ok {
		return "", errors.New("OpenBao secret missing value property")
	}
	return value, nil
}

func (c *Client) Write(ctx context.Context, name, value string) error {
	path, err := secretPath(name)
	if err != nil {
		return err
	}
	if value == "" {
		return nil
	}
	token, err := c.authenticate(ctx)
	if err != nil {
		return err
	}
	return c.request(ctx, http.MethodPost, "kv/data/"+path, token, map[string]any{"data": map[string]string{"value": value}}, nil)
}

func (c *Client) Delete(ctx context.Context, name string) error {
	path, err := secretPath(name)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(name, "prod-homechef-vendor-payment-") && !strings.HasPrefix(name, "prod-homechef-driver-payment-") {
		return errors.New("deletion limited to payment-owner secrets")
	}
	token, err := c.authenticate(ctx)
	if err != nil {
		return err
	}
	err = c.request(ctx, http.MethodDelete, "kv/metadata/"+path, token, nil, nil)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}
