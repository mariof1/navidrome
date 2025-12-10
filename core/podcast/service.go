package podcast

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type Service struct {
	repo            model.PodcastRepository
	refreshInterval time.Duration
	httpClient      *http.Client
}

type FeedErrorCode string

const (
	ErrInvalidURL  FeedErrorCode = "invalid_url"
	ErrFetchFailed FeedErrorCode = "fetch_failed"
	ErrInvalidFeed FeedErrorCode = "invalid_feed"
)

type FeedError struct {
	code FeedErrorCode
	err  error
}

func newFeedError(code FeedErrorCode, err error) *FeedError {
	return &FeedError{code: code, err: err}
}

func (e *FeedError) Error() string {
	return e.err.Error()
}

func (e *FeedError) Unwrap() error {
	return e.err
}

func (e *FeedError) Code() FeedErrorCode {
	return e.code
}

func (e *FeedError) Message() string {
	switch e.code {
	case ErrInvalidURL:
		return "invalid rss url"
	case ErrInvalidFeed:
		return "invalid rss feed"
	default:
		return "could not fetch rss feed"
	}
}

func NewService(repo model.PodcastRepository) *Service {
	return &Service{repo: repo, refreshInterval: time.Hour, httpClient: http.DefaultClient}
}

func (s *Service) AddChannel(ctx context.Context, url string, owner *model.User, isGlobal bool) (*model.PodcastChannel, error) {
	if url == "" {
		return nil, errors.New("rss url required")
	}
	feed, err := s.fetchFeed(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetch feed %q: %w", url, err)
	}
	now := time.Now()
	channel := &model.PodcastChannel{
		Title:           feed.Title,
		RSSURL:          url,
		SiteURL:         feed.Link,
		Description:     feed.Description,
		ImageURL:        feed.ImageURL,
		UserID:          owner.ID,
		IsGlobal:        isGlobal,
		LastRefreshedAt: &now,
	}
	if err := s.repo.CreateChannel(channel); err != nil {
		return nil, fmt.Errorf("create channel %q: %w", url, err)
	}
	episodes := s.mapEpisodes(channel, feed)
	if err := s.repo.SaveEpisodes(channel.ID, episodes); err != nil {
		return nil, fmt.Errorf("save episodes for %q: %w", url, err)
	}
	channel.Episodes = episodes
	return channel, nil
}

func (s *Service) UpdateChannel(ctx context.Context, channelID string, url string, isGlobal bool) (*model.PodcastChannel, error) {
	channel, err := s.repo.GetChannel(channelID)
	if err != nil {
		return nil, err
	}

	url = strings.TrimSpace(url)
	if url == "" {
		url = channel.RSSURL
	}

	channel.IsGlobal = isGlobal

	var episodes model.PodcastEpisodes
	if url != channel.RSSURL {
		feed, err := s.fetchFeed(ctx, url)
		if err != nil {
			return nil, fmt.Errorf("fetch feed %q: %w", url, err)
		}
		now := time.Now()
		channel.LastRefreshedAt = &now
		channel.LastError = ""
		channel.RSSURL = url
		channel.Title = feed.Title
		channel.Description = feed.Description
		channel.SiteURL = feed.Link
		if feed.ImageURL != "" {
			channel.ImageURL = feed.ImageURL
		}
		episodes = s.mapEpisodes(channel, feed)
	}

	if err := s.repo.UpdateChannel(channel); err != nil {
		return nil, err
	}

	if episodes != nil {
		if err := s.repo.SaveEpisodes(channel.ID, episodes); err != nil {
			return nil, fmt.Errorf("save episodes for %q: %w", url, err)
		}
		channel.Episodes = episodes
	}

	return channel, nil
}

func (s *Service) RefreshChannel(ctx context.Context, channelID string) error {
	channel, err := s.repo.GetChannel(channelID)
	if err != nil {
		return err
	}
	feed, err := s.fetchFeed(ctx, channel.RSSURL)
	if err != nil {
		channel.LastError = err.Error()
		return s.repo.UpdateChannel(channel)
	}
	now := time.Now()
	channel.LastRefreshedAt = &now
	channel.LastError = ""
	channel.Title = feed.Title
	channel.Description = feed.Description
	channel.SiteURL = feed.Link
	if feed.ImageURL != "" {
		channel.ImageURL = feed.ImageURL
	}
	if err := s.repo.UpdateChannel(channel); err != nil {
		return err
	}
	episodes := s.mapEpisodes(channel, feed)
	return s.repo.SaveEpisodes(channel.ID, episodes)
}

func (s *Service) ShouldRefresh(channel *model.PodcastChannel) bool {
	if channel.LastRefreshedAt == nil || channel.LastRefreshedAt.IsZero() {
		return true
	}
	return time.Since(*channel.LastRefreshedAt) > s.refreshInterval
}

func (s *Service) ListChannelsForUser(user *model.User) (model.PodcastChannels, error) {
	channels, err := s.repo.ListVisible(user.ID, true)
	if err != nil {
		return nil, err
	}
	for i := range channels {
		if s.ShouldRefresh(&channels[i]) {
			go func(ch model.PodcastChannel) {
				if err := s.RefreshChannel(context.Background(), ch.ID); err != nil {
					log.Error("podcast refresh failed", "id", ch.ID, "err", err)
				}
			}(channels[i])
		}
	}
	return channels, nil
}

func (s *Service) LoadChannelWithEpisodes(id string) (*model.PodcastChannel, error) {
	channel, err := s.repo.GetChannel(id)
	if err != nil {
		return nil, err
	}
	episodes, err := s.repo.ListEpisodes(id)
	if err != nil {
		return nil, err
	}
	channel.Episodes = episodes
	return channel, nil
}

type rssFeed struct {
	Title       string
	Link        string
	Description string
	ImageURL    string
	Items       []rssItem
}

type rssEnvelope struct {
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string         `xml:"title"`
	Link        string         `xml:"link"`
	Description string         `xml:"description"`
	Image       rssImage       `xml:"image"`
	ITunesImage rssITunesImage `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd image"`
	Items       []rssItem      `xml:"item"`
}

type rssImage struct {
	URL string `xml:"url"`
}

type rssITunesImage struct {
	Href string `xml:"href,attr"`
}

type rssItem struct {
	GUID        string         `xml:"guid"`
	Title       string         `xml:"title"`
	Link        string         `xml:"link"`
	Description string         `xml:"description"`
	Enclosure   rssEnclosure   `xml:"enclosure"`
	PubDate     string         `xml:"pubDate"`
	ITunes      rssITunesItem  `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd duration"`
	ITunesImage rssITunesImage `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd image"`
}

type rssITunesItem struct {
	Value string `xml:",chardata"`
}

type rssEnclosure struct {
	URL  string `xml:"url,attr"`
	Type string `xml:"type,attr"`
}

func (s *Service) fetchFeed(ctx context.Context, url string) (*rssFeed, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, newFeedError(ErrInvalidURL, err)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, newFeedError(ErrFetchFailed, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, newFeedError(ErrFetchFailed, fmt.Errorf("feed returned status %d", resp.StatusCode))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, newFeedError(ErrFetchFailed, err)
	}
	var env rssEnvelope
	if err := xml.Unmarshal(data, &env); err != nil {
		return nil, newFeedError(ErrInvalidFeed, err)
	}
	feed := &rssFeed{
		Title:       env.Channel.Title,
		Link:        env.Channel.Link,
		Description: env.Channel.Description,
		ImageURL:    env.Channel.Image.URL,
		Items:       env.Channel.Items,
	}
	if feed.ImageURL == "" && env.Channel.ITunesImage.Href != "" {
		feed.ImageURL = env.Channel.ITunesImage.Href
	}
	return feed, nil
}

func (s *Service) mapEpisodes(channel *model.PodcastChannel, feed *rssFeed) model.PodcastEpisodes {
	var episodes model.PodcastEpisodes
	for _, item := range feed.Items {
		ep := model.PodcastEpisode{
			GUID:        firstNonEmpty(item.GUID, item.Link, item.Title),
			Title:       item.Title,
			Description: item.Description,
			AudioURL:    item.Enclosure.URL,
			MimeType:    item.Enclosure.Type,
			ImageURL:    channel.ImageURL,
		}
		if item.ITunesImage.Href != "" {
			ep.ImageURL = item.ITunesImage.Href
		}
		if t, err := time.Parse(time.RFC1123Z, item.PubDate); err == nil {
			ep.PublishedAt = t
		}
		if item.ITunes.Value != "" {
			ep.Duration = parseDuration(item.ITunes.Value)
		}
		episodes = append(episodes, ep)
	}
	return episodes
}

func parseDuration(raw string) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	if strings.Contains(raw, ":") {
		parts := strings.Split(raw, ":")
		var seconds int64
		for _, p := range parts {
			v, _ := strconv.ParseInt(p, 10, 64)
			seconds = seconds*60 + v
		}
		return seconds
	}
	v, _ := strconv.ParseInt(raw, 10, 64)
	return v
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
