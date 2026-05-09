package domain

import (
	"github.com/google/uuid"
)

type MessageAnalytics struct {
	TotalSent      int             `json:"total_sent"`
	Delivered      int             `json:"delivered"`
	Read           int             `json:"read"`
	Failed         int             `json:"failed"`
	DeliveryRate   float64         `json:"delivery_rate"`
	ReadRate       float64         `json:"read_rate"`
	StatsByDay     []DayStats      `json:"stats_by_day"`
	StatsByChannel map[Channel]int `json:"stats_by_channel"`
}

type DayStats struct {
	Date string `json:"date"`
	Sent int    `json:"sent"`
	Read int    `json:"read"`
}

type EmailStats struct {
	TotalSent     int     `json:"total_sent"`
	Delivered     int     `json:"delivered"`
	DeliveryRate  float64 `json:"delivery_rate"` // delivered / sent * 100
	Opens         int     `json:"opens"`
	UniqueOpens   int     `json:"unique_opens"`
	OpenRate      float64 `json:"open_rate"` // unique_opens / delivered * 100
	Clicks        int     `json:"clicks"`
	UniqueClicks  int     `json:"unique_clicks"`
	ClickRate     float64 `json:"click_rate"` // unique_clicks / delivered * 100
	Bounces       int     `json:"bounces"`
	HardBounces   int     `json:"hard_bounces"`
	SoftBounces   int     `json:"soft_bounces"`
	BounceRate    float64 `json:"bounce_rate"` // bounces / sent * 100
	Complaints    int     `json:"complaints"`
	ComplaintRate float64 `json:"complaint_rate"` // complaints / delivered * 100
	Unsubscribes  int     `json:"unsubscribes"`
	UnsubRate     float64 `json:"unsub_rate"` // unsubscribes / delivered * 100
}

type EmailCampaignStats struct {
	CampaignID      uuid.UUID          `json:"campaign_id"`
	EmailStats                         // embedded
	TotalRecipients int                `json:"total_recipients"`
	Suppressed      int                `json:"suppressed"`   // blocked by suppression list
	Unsubscribed    int                `json:"unsubscribed"` // blocked by unsubscribe list
	TopLinks        []LinkStats        `json:"top_links"`
	OpensByHour     []HourlyEngagement `json:"opens_by_hour"`
	ClicksByHour    []HourlyEngagement `json:"clicks_by_hour"`
}

type LinkStats struct {
	URL          string `json:"url"`
	TotalClicks  int    `json:"total_clicks"`
	UniqueClicks int    `json:"unique_clicks"`
}

type HourlyEngagement struct {
	Hour  int `json:"hour"` // 0-23
	Count int `json:"count"`
}

type DomainReputation struct {
	BounceRate    float64 `json:"bounce_rate_30d"`    // last 30 days
	ComplaintRate float64 `json:"complaint_rate_30d"` // last 30 days
	HealthStatus  string  `json:"health_status"`      // "good", "warning", "critical"
	// "good": bounce < 2%, complaint < 0.1%
	// "warning": bounce 2-5% or complaint 0.1-0.3%
	// "critical": bounce > 5% or complaint > 0.3%
}

type BounceReport struct {
	HardBounces int                `json:"hard_bounces"`
	SoftBounces int                `json:"soft_bounces"`
	TopReasons  []BounceReasonStat `json:"top_reasons"`
	TopDomains  []BounceDomainStat `json:"top_domains"`
}

type BounceReasonStat struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

type BounceDomainStat struct {
	Domain      string  `json:"domain"`
	BounceRate  float64 `json:"bounce_rate"`
	TotalSent   int     `json:"total_sent"`
	TotalBounce int     `json:"total_bounce"`
}
