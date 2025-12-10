package model

import "time"

// PodcastChannel represents a podcast feed configured in Navidrome.
type PodcastChannel struct {
	ID              string     `structs:"id" json:"id"`
	Title           string     `structs:"title" json:"title"`
	RSSURL          string     `structs:"rss_url" json:"rssUrl"`
	SiteURL         string     `structs:"site_url" json:"siteUrl"`
	Description     string     `structs:"description" json:"description"`
	ImageURL        string     `structs:"image_url" json:"imageUrl"`
	UserID          string     `structs:"user_id" json:"userId"`
	IsGlobal        bool       `structs:"is_global" json:"isGlobal"`
	CreatedAt       time.Time  `structs:"created_at" json:"createdAt"`
	UpdatedAt       time.Time  `structs:"updated_at" json:"updatedAt"`
	LastRefreshedAt *time.Time `structs:"last_refreshed_at" json:"lastRefreshedAt"`
	LastError       string     `structs:"last_error" json:"lastError"`

	Episodes PodcastEpisodes `structs:"-" json:"episodes,omitempty"`
}

// PodcastEpisode represents an episode belonging to a podcast channel.
type PodcastEpisode struct {
	ID          string    `structs:"id" json:"id"`
	ChannelID   string    `structs:"channel_id" json:"channelId"`
	GUID        string    `structs:"guid" json:"guid"`
	Title       string    `structs:"title" json:"title"`
	Description string    `structs:"description" json:"description"`
	AudioURL    string    `structs:"audio_url" json:"audioUrl"`
	MimeType    string    `structs:"mime_type" json:"mimeType"`
	Duration    int64     `structs:"duration" json:"duration"`
	PublishedAt time.Time `structs:"published_at" json:"publishedAt"`
	ImageURL    string    `structs:"image_url" json:"imageUrl"`
	CreatedAt   time.Time `structs:"created_at" json:"createdAt"`
	UpdatedAt   time.Time `structs:"updated_at" json:"updatedAt"`
}

// PodcastEpisodes helper slice.
type PodcastEpisodes []PodcastEpisode

// PodcastChannels helper slice.
type PodcastChannels []PodcastChannel

type PodcastRepository interface {
	CreateChannel(channel *PodcastChannel) error
	UpdateChannel(channel *PodcastChannel) error
	DeleteChannel(id string) error
	GetChannel(id string) (*PodcastChannel, error)
	ListVisible(userID string, includeGlobal bool) (PodcastChannels, error)
	SaveEpisodes(channelID string, episodes PodcastEpisodes) error
	ListEpisodes(channelID string) (PodcastEpisodes, error)
	GetEpisode(id string) (*PodcastEpisode, error)
}
