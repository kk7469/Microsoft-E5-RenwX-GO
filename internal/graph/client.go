package graph

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"e5renewx/internal/model"
)

const (
	tokenURL = "https://login.microsoftonline.com/%s/oauth2/v2.0/token"
	graphURL = "https://graph.microsoft.com/v1.0"
)

type Client struct {
	http *http.Client
}

func New() *Client {
	return &Client{http: &http.Client{Timeout: 45 * time.Second}}
}

func (c *Client) EnsureToken(acc *model.Account) error {
	if acc.AccessToken != "" && time.Now().Add(2*time.Minute).Before(acc.TokenExpiry) {
		return nil
	}
	tenant := acc.Tenant
	if tenant == "" {
		tenant = "common"
	}
	form := url.Values{}
	form.Set("client_id", acc.ClientID)
	form.Set("scope", "https://graph.microsoft.com/.default")
	if acc.Mode == model.ModeLogin {
		form.Set("grant_type", "password")
		form.Set("username", acc.UPN)
		form.Set("password", acc.Secret)
		form.Set("scope", "https://graph.microsoft.com/.default offline_access")
	} else {
		form.Set("grant_type", "client_credentials")
		form.Set("client_secret", acc.Secret)
		if tenant == "common" {
			tenant = "organizations"
		}
	}

	resp, err := c.http.PostForm(fmt.Sprintf(tokenURL, tenant), form)
	if err != nil {
		return fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("token http %d: %s", resp.StatusCode, truncate(string(body), 400))
	}
	var tok struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Error        string `json:"error"`
		Desc         string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &tok); err != nil {
		return err
	}
	if tok.AccessToken == "" {
		return fmt.Errorf("token error %s: %s", tok.Error, tok.Desc)
	}
	acc.AccessToken = tok.AccessToken
	if tok.RefreshToken != "" {
		acc.RefreshToken = tok.RefreshToken
	}
	if tok.ExpiresIn <= 0 {
		tok.ExpiresIn = 3600
	}
	acc.TokenExpiry = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	return nil
}

type CallResult struct {
	Status   int
	OK       bool
	Message  string
	Duration time.Duration
	Path     string
	Method   string
}

func (c *Client) Call(acc *model.Account, api model.APIDef) CallResult {
	start := time.Now()
	path := strings.ReplaceAll(api.Path, "{upn}", url.PathEscape(acc.UPN))
	name := randomName()
	path = strings.ReplaceAll(path, "{name}", name)
	method := api.Method
	var body io.Reader
	contentType := ""

	switch api.ID {
	case "sendmail":
		payload := map[string]any{
			"message": map[string]any{
				"subject": fmt.Sprintf("E5 Renew X keepalive %s", time.Now().Format(time.RFC3339)),
				"body": map[string]string{
					"contentType": "Text",
					"content":     randomText(),
				},
				"toRecipients": []map[string]any{
					{"emailAddress": map[string]string{"address": acc.UPN}},
				},
			},
			"saveToSentItems": false,
		}
		b, _ := json.Marshal(payload)
		body = bytes.NewReader(b)
		contentType = "application/json"
	case "drive-upload":
		body = strings.NewReader(randomText())
		contentType = "text/plain"
	}

	req, err := http.NewRequest(method, graphURL+path, body)
	if err != nil {
		return CallResult{OK: false, Message: err.Error(), Duration: time.Since(start), Path: path, Method: method}
	}
	req.Header.Set("Authorization", "Bearer "+acc.AccessToken)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return CallResult{OK: false, Message: err.Error(), Duration: time.Since(start), Path: path, Method: method}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	ok := resp.StatusCode >= 200 && resp.StatusCode < 300
	msg := http.StatusText(resp.StatusCode)
	if !ok {
		msg = truncate(string(raw), 300)
	}
	return CallResult{
		Status:   resp.StatusCode,
		OK:       ok,
		Message:  msg,
		Duration: time.Since(start),
		Path:     path,
		Method:   method,
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func randomName() string {
	return fmt.Sprintf("keep-%d-%04d", time.Now().Unix(), rand.Intn(10000))
}

func randomText() string {
	words := []string{"keepalive", "graph", "onedrive", "outlook", "teams", "sharepoint", "renew", "e5", "developer", "sandbox"}
	var b strings.Builder
	n := 8 + rand.Intn(12)
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(words[rand.Intn(len(words))])
	}
	b.WriteString("\n")
	b.WriteString(time.Now().Format(time.RFC3339Nano))
	return b.String()
}
