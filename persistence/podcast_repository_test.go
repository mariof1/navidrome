package persistence

import (
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PodcastRepository", func() {
	var repo model.PodcastRepository

	BeforeEach(func() {
		ctx := log.NewContext(GinkgoT().Context())
		ctx = request.WithUser(ctx, model.User{ID: "userid", UserName: "userid", IsAdmin: true})
		repo = NewPodcastRepository(ctx, GetDBXBuilder())
	})

	It("creates channels and episodes", func() {
		now := time.Now()
		channel := model.PodcastChannel{
			Title:           "Test Podcast",
			RSSURL:          "https://example.com/rss",
			SiteURL:         "https://example.com",
			Description:     "A test feed",
			ImageURL:        "https://example.com/image.jpg",
			UserID:          "userid",
			IsGlobal:        true,
			LastRefreshedAt: &now,
		}

		Expect(repo.CreateChannel(&channel)).To(Succeed())
		Expect(channel.ID).NotTo(BeEmpty())

		episodes := model.PodcastEpisodes{{
			GUID:        "guid-1",
			Title:       "Episode 1",
			Description: "First",
			AudioURL:    "https://example.com/audio.mp3",
			MimeType:    "audio/mpeg",
			Duration:    123,
			PublishedAt: now,
			ImageURL:    "https://example.com/audio.jpg",
		}}

		Expect(repo.SaveEpisodes(channel.ID, episodes)).To(Succeed())

		savedChannel, err := repo.GetChannel(channel.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(savedChannel.Title).To(Equal(channel.Title))
		Expect(savedChannel.IsGlobal).To(BeTrue())

		savedEpisodes, err := repo.ListEpisodes(channel.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(savedEpisodes).To(HaveLen(1))
		Expect(savedEpisodes[0].GUID).To(Equal("guid-1"))
	})

	It("saves long UTF-8 descriptions", func() {
		description := "Lemoniada i ciasto.\nDrugi wiersz z HTML <strong>znacznikami</strong> oraz polskie znaki: ąśćłóżź."
		channel := model.PodcastChannel{
			Title:       "Opis",
			RSSURL:      "https://example.com/utf8",
			Description: description,
			UserID:      "userid",
		}

		Expect(repo.CreateChannel(&channel)).To(Succeed())

		saved, err := repo.GetChannel(channel.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(saved.Description).To(Equal(description))
	})

	It("deletes channels and their episodes respecting foreign keys", func() {
		channel := model.PodcastChannel{
			Title:  "Deletable",
			RSSURL: "https://example.com/delete",
			UserID: "userid",
		}

		Expect(repo.CreateChannel(&channel)).To(Succeed())

		episodes := model.PodcastEpisodes{{
			GUID:        "guid-delete",
			Title:       "Episode to delete",
			AudioURL:    "https://example.com/audio.mp3",
			MimeType:    "audio/mpeg",
			Duration:    10,
			PublishedAt: time.Now(),
		}}

		Expect(repo.SaveEpisodes(channel.ID, episodes)).To(Succeed())

		Expect(repo.DeleteChannel(channel.ID)).To(Succeed())

		_, err := repo.GetChannel(channel.ID)
		Expect(err).To(MatchError(model.ErrNotFound))

		savedEpisodes, err := repo.ListEpisodes(channel.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(savedEpisodes).To(BeEmpty())
	})
})
