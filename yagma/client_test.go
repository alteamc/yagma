package yagma

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
)

func TestClient_ProfileByUsername(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder(
		http.MethodGet, `=~^https://api\.mojang\.com/users/profiles/minecraft/((.+)(\?as=(\d+))?)?`,
		func(req *http.Request) (*http.Response, error) {
			username := httpmock.MustGetSubmatch(req, 1)
			switch {
			case username == "":
				return httpmock.NewJsonResponse(http.StatusNotFound, map[string]interface{}{
					"error":        "Not Found",
					"errorMessage": "The server has not found anything matching the request URI",
				})
			case len(username) > 25:
				return httpmock.NewJsonResponse(http.StatusBadRequest, map[string]interface{}{
					"error":        "BadRequestException",
					"errorMessage": fmt.Sprintf("%s is invalid", username),
				})
			case username == "aValidUser":
				res, err := httpmock.NewJsonResponse(http.StatusOK, map[string]interface{}{
					"name": "aValidUser",
					"id":   "02ebc15aa0ef4db1baa44422b61bc0ed",
				})
				if err != nil {
					return httpmock.NewBytesResponse(http.StatusInternalServerError, nil), nil
				}
				return res, nil
			case username == "missingUser":
				return httpmock.NewBytesResponse(http.StatusNoContent, nil), nil
			default:
				panic(fmt.Errorf("unexpected username %s", username))
			}
		},
	)

	y := New()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	u, err := y.ProfileByUsername(ctx, "aValidUser", time.Time{})
	if err != nil {
		t.Logf("unexpected error %s", err)
		t.Fail()
	} else {
		if u.Name != "aValidUser" {
			t.Logf("unexpected name %s", u.Name)
			t.Fail()
		}
		if u.ID.String() != "02ebc15a-a0ef-4db1-baa4-4422b61bc0ed" {
			t.Logf("unexpected uuid %s", u.ID)
			t.Fail()
		}
	}

	u, err = y.ProfileByUsername(ctx, "missingUser", time.Time{})
	if err == nil {
		t.Logf("unexpected nil error")
		t.Fail()
	} else {
		if !errors.Is(err, ErrNoSuchProfile) {
			t.Logf("unexpected %s error", reflect.TypeOf(err).Name())
			t.Fail()
		}
	}

	u, err = y.ProfileByUsername(ctx, "", time.Time{})
	if err == nil {
		t.Logf("unexpected nil error")
		t.Fail()
	} else {
		if err, ok := err.(*RequestError); !ok {
			t.Logf("unexpected %s error", reflect.TypeOf(err).String())
			t.Fail()
		} else {
			if err.Type != "Not Found" {
				t.Logf("unexpected error type %s", err.Type)
				t.Fail()
			}
			if err.Message != "The server has not found anything matching the request URI" {
				t.Logf("unexpeced error message %s", err.Message)
				t.Fail()
			}
		}
	}

	invalidName := strings.Repeat("0", 26)
	u, err = y.ProfileByUsername(ctx, invalidName, time.Time{})
	if err == nil {
		t.Logf("unexpected nil error")
		t.Fail()
	} else {
		if err, ok := err.(*RequestError); !ok {
			t.Logf("unexpected %s error", reflect.TypeOf(err).String())
			t.Fail()
		} else {
			if err.Type != "BadRequestException" {
				t.Logf("unexpected error type %s", err.Type)
				t.Fail()
			}
			if err.Message != fmt.Sprintf("%s is invalid", invalidName) {
				t.Logf("unexpeced error message %s", err.Message)
				t.Fail()
			}
		}
	}
}
