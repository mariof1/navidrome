package nativeapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type recordingAlbumRepo struct {
	*tests.MockAlbumRepo
	Calls []model.QueryOptions
}

func (r *recordingAlbumRepo) GetAll(qo ...model.QueryOptions) (model.Albums, error) {
	if len(qo) > 0 {
		r.Calls = append(r.Calls, qo[0])
	}
	return r.MockAlbumRepo.GetAll(qo...)
}

var _ = Describe("Recommendations API", func() {
	var (
		router   http.Handler
		ds       *tests.MockDataStore
		userRepo *tests.MockedUserRepo
		alRepo   *recordingAlbumRepo
		user     model.User
		token    string
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())

		alRepo = &recordingAlbumRepo{MockAlbumRepo: tests.CreateMockAlbumRepo()}
		userRepo = tests.CreateMockUserRepo()

		ds = &tests.MockDataStore{
			MockedAlbum:    alRepo,
			MockedUser:     userRepo,
			MockedProperty: &tests.MockedPropertyRepo{},
		}

		auth.Init(ds)

		user = model.User{ID: "user-1", UserName: "u", Name: "User", IsAdmin: false, NewPassword: "pw"}
		Expect(userRepo.Put(&user)).To(Succeed())

		var err error
		token, err = auth.CreateToken(&user)
		Expect(err).ToNot(HaveOccurred())

		albums := model.Albums{
			{ID: "album-1", Name: "A1", AlbumArtist: "AA", AlbumArtistID: "artist-1", LibraryID: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{ID: "album-2", Name: "A2", AlbumArtist: "BB", AlbumArtistID: "artist-2", LibraryID: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		}
		alRepo.SetData(albums)

		nativeRouter := New(ds, nil, nil, nil, nil, core.NewMockLibraryService(), nil)
		router = server.JWTVerifier(nativeRouter)
	})

	createReq := func(authenticated bool) *http.Request {
		req := httptest.NewRequest("GET", "/recommendations/home?limit=7&seed=s", nil)
		req = req.WithContext(context.TODO())
		if authenticated {
			req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+token)
		}
		return req
	}

	It("requires authentication", func() {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, createReq(false))
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
	})

	It("returns home sections and queries albums", func() {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, createReq(true))
		Expect(w.Code).To(Equal(http.StatusOK))

		var resp homeRecommendationsResponse
		Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
		Expect(resp.Sections).To(HaveLen(12))
		Expect(resp.Sections[0].ID).To(Equal("dailyMix1"))
		Expect(resp.Sections[0].Resource).To(Equal("album"))
		Expect(resp.Sections[0].Items).ToNot(BeEmpty())

		// Called once per section
		Expect(alRepo.Calls).To(HaveLen(12))

		countSort := func(sort string) int {
			count := 0
			for _, c := range alRepo.Calls {
				if c.Sort == sort {
					count++
				}
			}
			return count
		}

		randomSeeds := []string{}
		for _, c := range alRepo.Calls {
			Expect(c.Max).To(Equal(7))
			if c.Sort == "random" {
				randomSeeds = append(randomSeeds, c.Seed)
			}
		}

		// The endpoint should query all baseline sections + the new mixes.
		Expect(countSort("play_date")).To(Equal(1))
		Expect(countSort("starred_at")).To(Equal(1))
		Expect(countSort("recently_added")).To(Equal(2))
		Expect(countSort("play_count")).To(Equal(3))
		Expect(countSort("random")).To(Equal(5))
		Expect(randomSeeds).To(ConsistOf("s", "s-dm1", "s-dm2", "s-dm3", "s-inspired"))
	})
})
