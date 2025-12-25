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
			MockedAlbum: alRepo,
			MockedUser:  userRepo,
			MockedUserEvent: &tests.MockUserEventRepo{
				TopArtistIDs: []string{"artist-1", "artist-2", "artist-3", "artist-4", "artist-5", "artist-6"},
			},
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
		Expect(resp.Sections).ToNot(BeEmpty())
		// The endpoint should return all non-empty sections, in a stable order.
		// Daily mixes must be grouped together at the top.
		Expect(resp.Sections).To(HaveLen(16))
		Expect(resp.Sections[0].ID).To(Equal("dailyMix1"))
		Expect(resp.Sections[1].ID).To(Equal("dailyMix2"))
		Expect(resp.Sections[2].ID).To(Equal("dailyMix3"))
		Expect(resp.Sections[0].Resource).To(Equal("album"))
		Expect(resp.Sections[0].Items).ToNot(BeEmpty())

		// Called once per returned section
		Expect(alRepo.Calls).To(HaveLen(len(resp.Sections)))

		countSort := func(sort string) int {
			count := 0
			for _, c := range alRepo.Calls {
				if c.Sort == sort {
					count++
				}
			}
			return count
		}

		for _, c := range alRepo.Calls {
			Expect(c.Max).To(Equal(7))
		}
		Expect(countSort("random")).To(BeNumerically(">=", 1))
	})
})
