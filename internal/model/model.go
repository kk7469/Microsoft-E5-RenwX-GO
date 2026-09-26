package model

import "time"

type AuthMode string

const (
	ModeLogin AuthMode = "login"
	ModeApp   AuthMode = "app"
)

type AccountStatus string

const (
	StatusRunning AccountStatus = "running"
	StatusPaused  AccountStatus = "paused"
	StatusError   AccountStatus = "error"
)

type APIDef struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	Permission  string   `json:"permission"`
	Modes       []string `json:"modes"`
	RandomBody  bool     `json:"randomBody"`
	Recommended bool     `json:"recommended"`
	Description string   `json:"description"`
}

type Account struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	UPN             string        `json:"upn"`
	ClientID        string        `json:"clientId"`
	Secret          string        `json:"secret"`
	Tenant          string        `json:"tenant"`
	Mode            AuthMode      `json:"mode"`
	APIList         []string      `json:"apiList"`
	Status          AccountStatus `json:"status"`
	NotifyEmail     string        `json:"notifyEmail"`
	CreatedAt       time.Time     `json:"createdAt"`
	UpdatedAt       time.Time     `json:"updatedAt"`
	LastRunAt       *time.Time    `json:"lastRunAt,omitempty"`
	NextRunAt       *time.Time    `json:"nextRunAt,omitempty"`
	PausedAt        *time.Time    `json:"pausedAt,omitempty"`
	LastError       string        `json:"lastError,omitempty"`
	SuccessCount    int           `json:"successCount"`
	FailCount       int           `json:"failCount"`
	ConsecutiveFail int           `json:"consecutiveFail"`
	AccessToken     string        `json:"accessToken,omitempty"`
	RefreshToken    string        `json:"refreshToken,omitempty"`
	TokenExpiry     time.Time     `json:"tokenExpiry,omitempty"`
}

type CallLog struct {
	ID        string    `json:"id"`
	AccountID string    `json:"accountId"`
	APIID     string    `json:"apiId"`
	APIName   string    `json:"apiName"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Status    int       `json:"status"`
	OK        bool      `json:"ok"`
	Skipped   bool      `json:"skipped"`
	Message   string    `json:"message"`
	Duration  int64     `json:"durationMs"`
	CreatedAt time.Time `json:"createdAt"`
}

type Settings struct {
	AdminPassword      string `json:"adminPassword"`
	NotifyEmail        string `json:"notifyEmail"`
	SMTPHost           string `json:"smtpHost"`
	SMTPPort           int    `json:"smtpPort"`
	SMTPUser           string `json:"smtpUser"`
	SMTPPassword       string `json:"smtpPassword"`
	SMTPFrom           string `json:"smtpFrom"`
	MinIntervalSec     int    `json:"minIntervalSec"`
	MaxIntervalSec     int    `json:"maxIntervalSec"`
	MaxAPIsPerRound    int    `json:"maxApisPerRound"`
	FailPauseThreshold int    `json:"failPauseThreshold"`
	AutoResumeHours    int    `json:"autoResumeHours"`
	PardonIntervalDays int    `json:"pardonIntervalDays"`
	DailyReportHour    int    `json:"dailyReportHour"`
	ICPText            string `json:"icpText"`
	ICPLink            string `json:"icpLink"`
	Notice             string `json:"notice"`
	SiteName           string `json:"siteName"`
}

type Snapshot struct {
	Settings   Settings  `json:"settings"`
	Accounts   []Account `json:"accounts"`
	Logs       []CallLog `json:"logs"`
	LastPardon time.Time `json:"lastPardon"`
	LastReport time.Time `json:"lastReport"`
}
