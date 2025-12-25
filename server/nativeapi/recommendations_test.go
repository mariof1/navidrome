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

type apiSection struct {
	ID       string                   `json:"id"`
	Resource string                   `json:"resource"`
	To       string                   `json:"to"`
	Items    []map[string]interface{} `json:"items"`
}

type apiResp struct {
	Sections []apiSection `json:"sections"`
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

		mfRepo := tests.CreateMockMediaFileRepo()
		ds = &tests.MockDataStore{
			MockedAlbum: alRepo,
			MockedMediaFile: mfRepo,
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

		mfs := model.MediaFiles{
			{ID: "song-1", Title: "S1", Album: "A1", AlbumID: "album-1", Artist: "AA", AlbumArtist: "AA", AlbumArtistID: "artist-1", LibraryID: 1, Missing: false, HasCoverArt: true, UpdatedAt: time.Now()},
			{ID: "song-2", Title: "S2", Album: "A1", AlbumID: "album-1", Artist: "AA", AlbumArtist: "AA", AlbumArtistID: "artist-1", LibraryID: 1, Missing: false, HasCoverArt: true, UpdatedAt: time.Now()},
			{ID: "song-3", Title: "S3", Album: "A2", AlbumID: "album-2", Artist: "BB", AlbumArtist: "BB", AlbumArtistID: "artist-2", LibraryID: 1, Missing: false, HasCoverArt: true, UpdatedAt: time.Now()},
		}
		mfRepo.SetData(mfs)

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

		var resp apiResp
		Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
		Expect(resp.Sections).ToNot(BeEmpty())
		// The endpoint should return all non-empty sections, in a stable order.
		// Daily mixes must be grouped together at the top. Some sections may be empty
		// depending on filtering and cross-section de-duplication.
		Expect(resp.Sections[0].ID).To(Equal("dailyMix1"))
		Expect(resp.Sections[0].Resource).To(Equal("song"))
		Expect(resp.Sections[0].Items).ToNot(BeEmpty())

		idx := func(id string) int {
			for i, s := range resp.Sections {
				if s.ID == id {
					return i
				}
			}
			return -1
		}

		// If present, mixes must appear in declared relative order.
		i1 := idx("dailyMix1")
		i2 := idx("dailyMix2")
		i3 := idx("dailyMix3")
		iInspired := idx("inspiredBy")
		Expect(i1).To(Equal(0))
		if i2 >= 0 {
			Expect(i2).To(BeNumerically(">", i1))
		}
		if i3 >= 0 {
			Expect(i3).To(BeNumerically(">", i1))
			if i2 >= 0 {
				Expect(i3).To(BeNumerically(">", i2))
			}
		}
		if iInspired >= 0 {
			Expect(iInspired).To(BeNumerically(">", i1))
			if i3 >= 0 {
				Expect(iInspired).To(BeNumerically(">", i3))
			}
		}

		// All mix sections (if any) must be grouped at the top.
		mixIDs := map[string]struct{}{"dailyMix1": {}, "dailyMix2": {}, "dailyMix3": {}, "inspiredBy": {}}
		seenNonMix := false
		for _, s := range resp.Sections {
			_, isMix := mixIDs[s.ID]
			if !isMix {
				seenNonMix = true
				continue
			}
			Expect(seenNonMix).To(BeFalse())
		}

		// The handler may execute a query and later drop a section if it becomes empty
		// after post-filtering (e.g., cross-section de-duplication).
		Expect(alRepo.Calls).ToNot(BeEmpty())
		Expect(len(alRepo.Calls)).To(BeNumerically(">=", len(resp.Sections)))

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
			Expect(c.Max).To(BeNumerically(">=", 7))
			Expect(c.Max).To(BeNumerically("<=", 200))
		}
		Expect(countSort("random")).To(BeNumerically(">=", 1))
	})
})
