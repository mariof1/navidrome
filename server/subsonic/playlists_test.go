package subsonic

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ core.Playlists = (*fakePlaylists)(nil)

var _ = Describe("UpdatePlaylist", func() {
	var router *Router
	var ds model.DataStore
	var playlists *fakePlaylists

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		playlists = &fakePlaylists{}
		router = New(ds, nil, nil, nil, nil, nil, nil, nil, playlists, nil, nil, nil, nil, nil)
	})

	It("clears the comment when parameter is empty", func() {
		r := newGetRequest("playlistId=123", "comment=")
		_, err := router.UpdatePlaylist(r)
		Expect(err).ToNot(HaveOccurred())
		Expect(playlists.lastPlaylistID).To(Equal("123"))
		Expect(playlists.lastComment).ToNot(BeNil())
		Expect(*playlists.lastComment).To(Equal(""))
	})

	It("leaves comment unchanged when parameter is missing", func() {
		r := newGetRequest("playlistId=123")
		_, err := router.UpdatePlaylist(r)
		Expect(err).ToNot(HaveOccurred())
		Expect(playlists.lastPlaylistID).To(Equal("123"))
		Expect(playlists.lastComment).To(BeNil())
	})

	It("sets public to true when parameter is 'true'", func() {
		r := newGetRequest("playlistId=123", "public=true")
		_, err := router.UpdatePlaylist(r)
		Expect(err).ToNot(HaveOccurred())
		Expect(playlists.lastPlaylistID).To(Equal("123"))
		Expect(playlists.lastPublic).ToNot(BeNil())
		Expect(*playlists.lastPublic).To(BeTrue())
	})

	It("sets public to false when parameter is 'false'", func() {
		r := newGetRequest("playlistId=123", "public=false")
		_, err := router.UpdatePlaylist(r)
		Expect(err).ToNot(HaveOccurred())
		Expect(playlists.lastPlaylistID).To(Equal("123"))
		Expect(playlists.lastPublic).ToNot(BeNil())
		Expect(*playlists.lastPublic).To(BeFalse())
	})

	It("leaves public unchanged when parameter is missing", func() {
		r := newGetRequest("playlistId=123")
		_, err := router.UpdatePlaylist(r)
		Expect(err).ToNot(HaveOccurred())
		Expect(playlists.lastPlaylistID).To(Equal("123"))
		Expect(playlists.lastPublic).To(BeNil())
	})

	It("rejects updates to auto Daily Mix playlists", func() {
		mockDS := ds.(*tests.MockDataStore)
		mockDS.MockedPlaylist = &tests.MockPlaylistRepo{Entity: &model.Playlist{ID: "123", Path: model.DailyMixPlaylistPath("user-1", 1)}}
		r := newGetRequest("playlistId=123", "name=Nope")
		resp, err := router.UpdatePlaylist(r)
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		var subErr subError
		Expect(errors.As(err, &subErr)).To(BeTrue())
		Expect(subErr.code).To(Equal(responses.ErrorAuthorizationFail))
		Expect(playlists.lastPlaylistID).To(Equal(""))
	})
})

type fakePlaylists struct {
	core.Playlists
	lastPlaylistID string
	lastName       *string
	lastComment    *string
	lastPublic     *bool
	lastAdd        []string
	lastRemove     []int
}

func (f *fakePlaylists) Update(ctx context.Context, playlistID string, name *string, comment *string, public *bool, idsToAdd []string, idxToRemove []int) error {
	f.lastPlaylistID = playlistID
	f.lastName = name
	f.lastComment = comment
	f.lastPublic = public
	f.lastAdd = idsToAdd
	f.lastRemove = idxToRemove
	return nil
}
