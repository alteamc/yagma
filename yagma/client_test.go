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
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jarcoal/httpmock"
)

// Mock user data repository

const mockUserCount = 2000

type mockUser Profile

type mockUserRepo struct {
	idList            []uuid.UUID
	usersByUUID       map[uuid.UUID]*mockUser
	usersByName       map[string]*mockUser
	nameHistoryByUUID map[uuid.UUID][]*NameHistoryRecord
}

func newMockUserRepo() *mockUserRepo {
	r := &mockUserRepo{
		idList:            make([]uuid.UUID, mockUserCount),
		usersByUUID:       make(map[uuid.UUID]*mockUser, mockUserCount),
		usersByName:       make(map[string]*mockUser, mockUserCount),
		nameHistoryByUUID: make(map[uuid.UUID][]*NameHistoryRecord, mockUserCount),
	}

	for i := 0; i < mockUserCount; i++ {
		u := r.NewRandomUser()
		r.idList[i] = u.ID
		r.usersByUUID[u.ID] = u
		r.usersByName[strings.ToLower(u.Name)] = u
	}

	return r
}

func (r *mockUserRepo) NewRandomUser() *mockUser {
	var id uuid.UUID

	id, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}

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
	r.nameHistoryByUUID[id] = names

	return &mockUser{
		ID:         id,
		Name:       randomString(8 + mathRandom.Intn(16)),
		Legacy:     mathRandom.Intn(2) == 0,
		Demo:       mathRandom.Intn(2) == 0,
		Properties: nil,
	}
}

func (r *mockUserRepo) PickRandomUser() *mockUser {
	return r.usersByUUID[r.idList[mathRandom.Intn(mockUserCount)]]
}

func (r *mockUserRepo) PickNameHistory(uuid uuid.UUID) ([]*NameHistoryRecord, bool) {
	hist, err := r.nameHistoryByUUID[uuid]
	return hist, err
}

func (r *mockUserRepo) FindByName(name string) (*mockUser, bool) {
	u, e := r.usersByName[strings.ToLower(name)]
	return u, e
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

const ctxTimeout = 10 * time.Second

var client = New()
var users = newMockUserRepo()

func TestMain(m *testing.M) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	registerProfileByUsernameResponder()
	registerProfileByUsernameBulkResponder()

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
				user, found := users.FindByName(name)
				if !found {
					return newNoContentResponse(), nil
				}

				data := j{
					"id":   strings.ReplaceAll(user.ID.String(), "-", ""),
					"name": user.Name,
				}
				if user.Legacy {
					data["legacy"] = true
				}
				if user.Demo {
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

	for i := 0; i < 100; i++ {
		u := users.PickRandomUser()
		p, err := client.ProfileByUsername(ctx, u.Name, time.Time{})

		isNotNil(t, p)
		if errEqNil(t, err) {
			eq(t, u.ID, p.ID)
			eq(t, u.Name, p.Name)
			eq(t, u.Legacy, p.Legacy)
			eq(t, u.Demo, p.Demo)
		}
	}
}

func TestClient_ProfileByUsername2(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	for i := 0; i < 100; i++ {
		u := users.NewRandomUser()
		p, err := client.ProfileByUsername(ctx, u.Name, time.Time{})

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
					user, found := users.FindByName(name)
					if !found {
						continue
					}

					data := j{
						"id":   strings.ReplaceAll(user.ID.String(), "-", ""),
						"name": user.Name,
					}
					if user.Legacy {
						data["legacy"] = true
					}
					if user.Demo {
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

	for i := 0; i < 100; i++ {
		n := int(mathRandom.Int31n(10))
		names := make([]string, n)
		for k := 0; k < n; k++ {
			names[k] = users.PickRandomUser().Name
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

	for i := 0; i < 100; i++ {
		ne := int(mathRandom.Int31n(5 + 1))
		nm := 10 - ne
		names := make([]string, ne+nm)
		exist := make([]string, ne)
		for k := 0; k < ne; k++ {
			name := users.PickRandomUser().Name
			names[k] = name
			exist[k] = name
		}
		for k := ne; k < ne+nm; k++ {
			names[k] = users.NewRandomUser().Name
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
		names[k] = users.PickRandomUser().Name
	}
	names[n-1] = strings.Repeat("0", 26)

	p, err := client.ProfileByUsernameBulk(ctx, names)
	isZero(t, p)
	if errNeqNil(t, err) {
		as(t, err, reflect.TypeOf(&RequestError{}))
	}
}
