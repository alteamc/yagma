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

const mockUserCount = 20000

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
	id, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}

	b := make([]byte, mathRandom.Int31n(6)+4)
	if _, err = cryptoRandom.Read(b); err != nil {
		panic(err)
	}

	return &mockUser{
		ID:         id,
		Name:       hex.EncodeToString(b),
		Legacy:     mathRandom.Intn(2) == 0,
		Demo:       mathRandom.Intn(2) == 0,
		Properties: nil,
	}
}

func (r *mockUserRepo) PickRandomUser() *mockUser {
	return r.usersByUUID[r.idList[mathRandom.Intn(mockUserCount)]]
}

func (r *mockUserRepo) FindByName(name string) (*mockUser, bool) {
	u, e := r.usersByName[strings.ToLower(name)]
	return u, e
}

// Utility method definition

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

func newJSONResponse(status int, data j) *http.Response {
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

func logfAndFail(t *testing.T, format string, v ...interface{}) {
	t.Logf(format, v...)
	t.Fail()
}

func errEqNil(t *testing.T, err error) bool {
	if err != nil {
		logfAndFail(t, "expected error to be nil, but got %v", err)
		return false
	}

	return true
}

func errNeqNil(t *testing.T, err error) bool {
	if err == nil {
		logfAndFail(t, "expected error to be not nil, but got nil")
		return false
	}

	return true
}

func as(t *testing.T, v interface{}, dt reflect.Type) bool {
	if !reflect.TypeOf(v).AssignableTo(dt) {
		logfAndFail(t, "expected %v to be of type %v, but got type %v", v, dt, reflect.TypeOf(v))
		return false
	}

	return true
}

func errIs(t *testing.T, err error, check error) bool {
	if !errors.Is(err, check) {
		logfAndFail(t, "expected %v error, but got %v", check, err)
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

func isZero(t *testing.T, v interface{}) bool {
	if !reflect.ValueOf(v).IsZero() {
		logfAndFail(t, "expected %v, but got %v", reflect.Zero(reflect.TypeOf(v)), v)
		return false
	}

	return true
}

// Test utilities

func test(t *testing.T, desc string, fn func(t *testing.T)) {
	t.Logf("Testing %v", desc)
	fn(t)
}

// Test environment

var users = newMockUserRepo()

// Tests

func TestClient_ProfileByUsername(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	y := New()

	test(t, "random existing user", func(t *testing.T) {
		u := users.PickRandomUser()
		p, err := y.ProfileByUsername(ctx, u.Name, time.Time{})
		if errEqNil(t, err) {
			eq(t, u.ID, p.ID)
			eq(t, u.Name, p.Name)
			eq(t, u.Legacy, p.Legacy)
			eq(t, u.Demo, p.Demo)
		}
	})

	test(t, "random nonexistent user", func(t *testing.T) {
		u := users.NewRandomUser()
		p, err := y.ProfileByUsername(ctx, u.Name, time.Time{})
		isZero(t, p)
		if errNeqNil(t, err) {
			errIs(t, err, ErrNoSuchProfile)
		}
	})

	test(t, "empty username", func(t *testing.T) {
		p, err := y.ProfileByUsername(ctx, "", time.Time{})
		isZero(t, p)
		if errNeqNil(t, err) {
			as(t, err, reflect.TypeOf(&RequestError{}))
		}
	})

	test(t, "invalid username", func(t *testing.T) {
		n := strings.Repeat("0", 26)
		p, err := y.ProfileByUsername(ctx, n, time.Time{})
		isZero(t, p)
		if errNeqNil(t, err) {
			as(t, err, reflect.TypeOf(&RequestError{}))
		}
	})
}
