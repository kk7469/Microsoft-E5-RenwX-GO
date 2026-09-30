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

func resolveTenant(acc *model.Account) string {
	t := strings.TrimSpace(acc.Tenant)
	if t != "" && !strings.EqualFold(t, "common") && !strings.EqualFold(t, "consumers") {
		return t
	}
	if i := strings.LastIndex(acc.UPN, "@"); i >= 0 {
		domain := strings.TrimSpace(acc.UPN[i+1:])
		if domain != "" {
			return domain
		}
	}
	return "organizations"
}

func (c *Client) EnsureToken(acc *model.Account) error {
	if acc.AccessToken != "" && time.Now().Add(2*time.Minute).Before(acc.TokenExpiry) {
		return nil
	}
	tenant := resolveTenant(acc)
	form := url.Values{}
	form.Set("client_id", acc.ClientID)
	form.Set("scope", "https://graph.microsoft.com/.default")
	if acc.Mode == model.ModeLogin {
		if acc.RefreshToken != "" {
			form.Set("grant_type", "refresh_token")
			form.Set("refresh_token", acc.RefreshToken)
			form.Set("scope", "https://graph.microsoft.com/.default offline_access")
		} else {
			form.Set("grant_type", "password")
			form.Set("username", acc.UPN)
			form.Set("password", acc.Secret)
			form.Set("scope", "https://graph.microsoft.com/.default offline_access")
		}
	} else {
		form.Set("grant_type", "client_credentials")
		form.Set("client_secret", acc.Secret)
	}

	resp, err := c.http.PostForm(fmt.Sprintf(tokenURL, tenant), form)
	if err != nil {
		return fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var tok struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Error        string `json:"error"`
		Desc         string `json:"error_description"`
	}
	_ = json.Unmarshal(body, &tok)
	if resp.StatusCode >= 300 || tok.AccessToken == "" {
		if acc.Mode == model.ModeLogin && acc.RefreshToken != "" && form.Get("grant_type") == "refresh_token" {
			acc.RefreshToken = ""
			return c.EnsureToken(acc)
		}
		if resp.StatusCode >= 300 {
			return fmt.Errorf("token http %d: %s", resp.StatusCode, truncate(string(body), 400))
		}
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
	Skipped  bool
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
				"subject": fmt.Sprintf("E5 RenewX GO keepalive %s", time.Now().Format(time.RFC3339)),
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
	skipped := !ok && isPermissionSkip(resp.StatusCode, string(raw))
	return CallResult{
		Status:   resp.StatusCode,
		OK:       ok,
		Skipped:  skipped,
		Message:  msg,
		Duration: time.Since(start),
		Path:     path,
		Method:   method,
	}
}

func isPermissionSkip(status int, body string) bool {
	if status != 401 && status != 403 {
		return false
	}
	s := strings.ToLower(body)
	keys := []string{
		"authentication_msgraphpermissionmissing",
		"authorization_requestdenied",
		"erroraccessdenied",
		"insufficient privileges",
		"does not have required microsoft graph permission",
		"missing required permissions",
	}
	for _, k := range keys {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
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
	words := []string{"北国风光，千里冰封万里雪飘","望长城内外，惟余莽莽","大河上下，顿失滔滔","山舞银蛇，原驰蜡象","欲与天公试比高","须晴日，看红装素裹，分外妖娆","江山如此多娇，引无数英雄竞折腰","惜秦皇汉武，略输文采","唐宗宋祖，稍逊风骚","一代天骄成吉思汗，只识弯弓射大雕","俱往矣，数风流人物，还看今朝"}
	sentence := words[rand.Intn(len(words))]
	return fmt.Sprintf("%s\n%s-%04d", sentence, time.Now().Format(time.RFC3339), rand.Intn(10000))
}
