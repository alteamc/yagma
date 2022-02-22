package yagma

import (
	"bytes"
	"context"
	cryptoRandom "crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	mathRandom "math/rand"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jarcoal/httpmock"
)

// Mock user data repository

const mockUserCount = 10000

type mockUser struct {
	profile *Profile
	names   []*NameHistoryRecord
}

type mockNameHistory nameHistoryRecordJSONMappingArray

type mockUserRepo struct {
	idList            []uuid.UUID
	usersByUUID       map[uuid.UUID]*mockUser
	usersByName       map[string]*mockUser
	nameHistoryByUUID map[uuid.UUID]mockNameHistory
}

func newMockUserRepo() *mockUserRepo {
	r := &mockUserRepo{
		idList:            make([]uuid.UUID, mockUserCount),
		usersByUUID:       make(map[uuid.UUID]*mockUser, mockUserCount),
		usersByName:       make(map[string]*mockUser, mockUserCount),
		nameHistoryByUUID: make(map[uuid.UUID]mockNameHistory, mockUserCount),
	}

	for i := 0; i < mockUserCount; i++ {
		u := r.NewRandomUser()
		r.idList[i] = u.profile.ID
		r.usersByUUID[u.profile.ID] = u
		r.usersByName[strings.ToLower(u.profile.Name)] = u
	}

	return r
}

func (r *mockUserRepo) NewRandomUser() *mockUser {
	var id uuid.UUID

	id, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}
	name := randomString(8 + mathRandom.Intn(16))

	u := &mockUser{
		profile: &Profile{
			ID:     id,
			Name:   name,
			Legacy: mathRandom.Intn(2) == 0,
			Demo:   mathRandom.Intn(2) == 0,
		},
	}

	u.profile.Properties = newProperties(u)
	u.names = newNameHistory()
	return u
}

func newNameHistory() []*NameHistoryRecord {
	n := 3 + mathRandom.Intn(5)
	names := make([]*NameHistoryRecord, 0, 8)
	now := time.UnixMilli(time.Now().UnixMilli() - mathRandom.Int63n(1_000_000)*1000)
	for i := 1; i < n; i++ {
		now = time.UnixMilli(now.UnixMilli() - mathRandom.Int63n(100_000)*1000)
		names = append(names, &NameHistoryRecord{
			Name:      randomString(3 + mathRandom.Intn(22)),
			ChangedAt: now,
		})
	}
	names = append(names, &NameHistoryRecord{Name: randomString(3 + mathRandom.Intn(22))})
	return names
}

func newProperties(u *mockUser) []*ProfileProperty {
	m := &profileTexturesJSONMapping{
		ProfileID:   u.profile.ID,
		ProfileName: u.profile.Name,
		Timestamp:   time.Now().UnixMilli(),
	}

	if mathRandom.Intn(2) == 0 {
		m.Textures.Skin.URL = "https://textures.minecraft.net/texture/" + randomString(64)
		if mathRandom.Intn(2) == 0 {
			m.Textures.Skin.Metadata.Model = "slim"
		}
	}
	if mathRandom.Intn(2) == 0 {
		m.Textures.Cape.URL = "https://textures.minecraft.net/texture/" + randomString(64)
	}

	pp, err := m.Wrap().ProfileProperty()
	if err != nil {
		panic(err)
	}

	return []*ProfileProperty{pp}
}

func (r *mockUserRepo) RandomUser() *mockUser {
	id := r.idList[mathRandom.Intn(mockUserCount)]
	return r.usersByUUID[id]
}

func (r *mockUserRepo) ByName(name string) (*mockUser, bool) {
	u, exists := r.usersByName[strings.ToLower(name)]
	return u, exists
}

func (r *mockUserRepo) ByUUID(id uuid.UUID) (*mockUser, bool) {
	u, exists := r.usersByUUID[id]
	return u, exists
}

// Utility method definition

func randomString(length int) string {
	var buf []byte
	if length%2 != 0 {
		buf = make([]byte, (length/2)+1)
	} else {
		buf = make([]byte, length/2)
	}

	_, err := cryptoRandom.Read(buf)
	if err != nil {
		panic(err)
	}

	if length%2 != 0 {
		return hex.EncodeToString(buf)[:length]
	}

	return hex.EncodeToString(buf)
}

func (t *Time) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(time.Time(*t).UnixMilli(), 10)), nil
}

func (u *UUID) MarshalJSON() ([]byte, error) {
	return []byte(`"` + strings.ReplaceAll(uuid.UUID(*u).String(), "-", "") + `"`), nil
}

func (t *ProfileTextures) Unwrap() *profileTexturesJSONMapping {
	m := &profileTexturesJSONMapping{
		ProfileID:   t.ProfileID,
		ProfileName: t.ProfileName,
		Timestamp:   t.Timestamp.UnixMilli(),
		Textures: struct {
			Skin struct {
				URL      string `json:"url"`
				Metadata struct {
					Model string `json:"model"`
				} `json:"metadata"`
			} `json:"SKIN"`
			Cape struct {
				URL string `json:"url"`
			} `json:"CAPE"`
		}{
			Skin: struct {
				URL      string `json:"url"`
				Metadata struct {
					Model string `json:"model"`
				} `json:"metadata"`
			}{
				URL: t.Skin,
				Metadata: struct {
					Model string `json:"model"`
				}{}, // init this further on
			},
			Cape: struct {
				URL string `json:"url"`
			}{URL: t.Cape},
		},
	}

	if t.SkinModel == SkinModelAlex {
		m.Textures.Skin.Metadata.Model = "slim"
	}

	return m
}

func (t *ProfileTextures) ProfileProperty() (*ProfileProperty, error) {
	jsonBytes, err := json.Marshal(t.Unwrap())
	if err != nil {
		return nil, err
	}

	b64bytes := &bytes.Buffer{}
	_, err = base64.NewEncoder(base64.StdEncoding, b64bytes).Write(jsonBytes)
	if err != nil {
		panic(err)
	}

	return &ProfileProperty{
		Name:  "textures",
		Value: b64bytes.String(),
	}, nil
}

// Response templates

type j = map[string]interface{}

func newJSONResponse(status int, data interface{}) *http.Response {
	r, err := httpmock.NewJsonResponse(status, data)
	if err != nil {
		panic(err)
	}
	return r
}

func newNotFoundResponse() *http.Response {
	return newJSONResponse(http.StatusNotFound, j{
		"error":        "Not Found",
		"errorMessage": "The server has not found anything matching the request URI",
	})
}

func newBadRequestExceptionResponse(v string) *http.Response {
	return newJSONResponse(http.StatusBadRequest, j{
		"error":        "BadRequestException",
		"errorMessage": fmt.Sprintf("%s is invalid", v),
	})
}

func newNoContentResponse() *http.Response {
	return httpmock.NewBytesResponse(http.StatusNoContent, nil)
}

// Assertions

func as(t *testing.T, v interface{}, dt reflect.Type) bool {
	if !reflect.TypeOf(v).AssignableTo(dt) {
		logfAndFailNow(t, "expected %v to be of type %v, but got type %v", v, dt, reflect.TypeOf(v))
		return false
	}

	return true
}

func eq(t *testing.T, exp interface{}, act interface{}) bool {
	if exp != act {
		t.Logf("expected %v, got %v", exp, act)
		t.Fail()
		return false
	}

	return true
}

func errEqNil(t *testing.T, err error) bool {
	if err != nil {
		logfAndFailNow(t, "expected error to be nil, but got %v", err)
		return false
	}

	return true
}

func errNeqNil(t *testing.T, err error) bool {
	if err == nil {
		logfAndFailNow(t, "expected error to be not nil, but got nil")
		return false
	}

	return true
}

func errIs(t *testing.T, err error, check error) bool {
	if !errors.Is(err, check) {
		logfAndFailNow(t, "expected %v error, but got %v", check, err)
		return false
	}

	return true
}

func isZero(t *testing.T, v interface{}) bool {
	if !reflect.ValueOf(v).IsZero() {
		logfAndFailNow(t, "expected %v, but got %v", reflect.Zero(reflect.TypeOf(v)), v)
		return false
	}

	return true
}

func isNotZero(t *testing.T, v interface{}) bool {
	if reflect.ValueOf(v).IsZero() {
		logfAndFailNow(t, "expected %v, but got %v", reflect.Zero(reflect.TypeOf(v)), v)
		return false
	}

	return true
}

func isNotNil(t *testing.T, v interface{}) bool {
	if v == nil {
		logfAndFailNow(t, "expected %v, but got nil", v)
		return false
	}

	return true
}

func saContains(t *testing.T, arr []string, v string) bool {
	for _, it := range arr {
		if it == v {
			return true
		}
	}

	logfAndFailNow(t, "expected %v to contain %v, but it does not", arr, v)
	return false
}

// Test utilities

func logfAndFailNow(t *testing.T, format string, v ...interface{}) {
	t.Logf(format, v...)
	t.FailNow()
}

// Test environment

const (
	ctxTimeout = 10 * time.Second
	iterations = 10000
)

var (
	client = New()
	users  = newMockUserRepo()
)

func TestMain(m *testing.M) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	registerProfileByUsernameResponder()
	registerProfileByUsernameBulkResponder()
	registerNameHistoryByUUIDResponder()

	m.Run()
}

// Tests

func registerProfileByUsernameResponder() {
	httpmock.RegisterResponder(
		http.MethodGet, `=~^https://api\.mojang\.com/users/profiles/minecraft/(?:(.*)(?:at=(.*))?)?`,
		func(r *http.Request) (*http.Response, error) {
			name := httpmock.MustGetSubmatch(r, 1)
			switch {
			case len(name) == 0:
				return newNotFoundResponse(), nil
			case len(name) > 25:
				return newBadRequestExceptionResponse(name), nil
			default:
				user, found := users.ByName(name)
				if !found {
					return newNoContentResponse(), nil
				}

				data := j{
					"id":   strings.ReplaceAll(user.profile.ID.String(), "-", ""),
					"name": user.profile.Name,
				}
				if user.profile.Legacy {
					data["legacy"] = true
				}
				if user.profile.Demo {
					data["demo"] = true
				}

				return newJSONResponse(http.StatusOK, data), nil
			}
		},
	)
}

func TestClient_ProfileByUsername(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	for i := 0; i < iterations; i++ {
		u := users.RandomUser()
		p, err := client.ProfileByUsername(ctx, u.profile.Name, time.Time{})

		isNotNil(t, p)
		if errEqNil(t, err) {
			eq(t, u.profile.ID, p.ID)
			eq(t, u.profile.Name, p.Name)
			eq(t, u.profile.Legacy, p.Legacy)
			eq(t, u.profile.Demo, p.Demo)
		}
	}
}

func TestClient_ProfileByUsername2(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	for i := 0; i < iterations; i++ {
		u := users.NewRandomUser()
		p, err := client.ProfileByUsername(ctx, u.profile.Name, time.Time{})

		isZero(t, p)
		if errNeqNil(t, err) {
			errIs(t, err, ErrNoSuchProfile)
		}
	}
}

func TestClient_ProfileByUsername3(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	p, err := client.ProfileByUsername(ctx, "", time.Time{})

	isZero(t, p)
	if errNeqNil(t, err) {
		as(t, err, reflect.TypeOf(&RequestError{}))
	}
}

func TestClient_ProfileByUsername4(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	n := strings.Repeat("0", 26)
	p, err := client.ProfileByUsername(ctx, n, time.Time{})

	isZero(t, p)
	if errNeqNil(t, err) {
		as(t, err, reflect.TypeOf(&RequestError{}))
	}
}

func registerProfileByUsernameBulkResponder() {
	httpmock.RegisterResponder(
		http.MethodPost, `=~^https://api\.mojang\.com/profiles/minecraft`,
		func(r *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				panic(err)
			}

			var rn []string
			if err = json.Unmarshal(body, &rn); err != nil {
				panic(err)
			}

			p := make([]interface{}, 0, len(rn))
			for _, name := range rn {
				switch {
				case len(name) == 0, len(name) > 25:
					return newBadRequestExceptionResponse(name), nil
				default:
					user, found := users.ByName(name)
					if !found {
						continue
					}

					data := j{
						"id":   strings.ReplaceAll(user.profile.ID.String(), "-", ""),
						"name": user.profile.Name,
					}
					if user.profile.Legacy {
						data["legacy"] = true
					}
					if user.profile.Demo {
						data["demo"] = true
					}

					p = append(p, data)
				}
			}

			return newJSONResponse(http.StatusOK, p), nil
		},
	)
}

func TestClient_ProfileByUsernameBulk(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	for i := 0; i < iterations; i++ {
		n := int(mathRandom.Int31n(10))
		names := make([]string, n)
		for k := 0; k < n; k++ {
			u := users.RandomUser()
			names[k] = u.profile.Name
		}

		p, err := client.ProfileByUsernameBulk(ctx, names)
		if errEqNil(t, err) {
			eq(t, n, len(p))
			for _, it := range p {
				saContains(t, names, it.Name)
			}
		}
	}
}

func TestClient_ProfileByUsernameBulk2(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	for i := 0; i < iterations; i++ {
		ne := int(mathRandom.Int31n(5 + 1))
		nm := 10 - ne
		names := make([]string, ne+nm)
		exist := make([]string, ne)
		for k := 0; k < ne; k++ {
			u := users.RandomUser()
			name := u.profile.Name
			names[k] = name
			exist[k] = name
		}
		for k := ne; k < ne+nm; k++ {
			u := users.NewRandomUser()
			names[k] = u.profile.Name
		}

		p, err := client.ProfileByUsernameBulk(ctx, names)
		if errEqNil(t, err) {
			eq(t, ne, len(p))
			for _, it := range p {
				saContains(t, exist, it.Name)
			}
		}
	}
}

func TestClient_ProfileByUsernameBulk3(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	n := int(1 + mathRandom.Int31n(9))
	names := make([]string, n)
	for k := 0; k < n-1; k++ {
		u := users.RandomUser()
		names[k] = u.profile.Name
	}
	names[n-1] = strings.Repeat("0", 26)

	p, err := client.ProfileByUsernameBulk(ctx, names)
	isZero(t, p)
	if errNeqNil(t, err) {
		as(t, err, reflect.TypeOf(&RequestError{}))
	}
}

func registerNameHistoryByUUIDResponder() {
	httpmock.RegisterResponder(
		http.MethodGet, `=~^https://api\.mojang\.com/user/profiles/(.*)/names`,
		func(r *http.Request) (*http.Response, error) {
			uuidStr := httpmock.MustGetSubmatch(r, 1)
			id := uuid.MustParse(uuidStr)

			user, exists := users.ByUUID(id)
			if !exists {
				return newNoContentResponse(), nil
			}

			return newJSONResponse(http.StatusOK, user.names), nil
		},
	)
}

func TestClient_NameHistoryByUUID(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	for i := 0; i < iterations; i++ {
		u := users.RandomUser()
		hist, err := client.NameHistoryByUUID(ctx, u.profile.ID)
		errEqNil(t, err)
		isNotZero(t, hist)
	}
}

func TestClient_NameHistoryByUUID2(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	for i := 0; i < iterations; i++ {
		u := users.NewRandomUser()
		hist, err := client.NameHistoryByUUID(ctx, u.profile.ID)
		errNeqNil(t, err)
		isZero(t, hist)
	}
}
