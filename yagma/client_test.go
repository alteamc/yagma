package yagma

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
		http.MethodGet, `=~^https://api\.mojang\.com/users/profiles/minecraft/((.+)(\?at=(\d+))?)?`,
		func(req *http.Request) (*http.Response, error) {
			username := httpmock.MustGetSubmatch(req, 1)
			switch {
			case username == "":
				res, err := httpmock.NewJsonResponse(http.StatusNotFound, map[string]interface{}{
					"error":        "Not Found",
					"errorMessage": "The server has not found anything matching the request URI",
				})
				if err != nil {
					panic(err)
				}
				return res, nil
			case len(username) > 25:
				res, err := httpmock.NewJsonResponse(http.StatusBadRequest, map[string]interface{}{
					"error":        "BadRequestException",
					"errorMessage": fmt.Sprintf("%s is invalid", username),
				})
				if err != nil {
					panic(err)
				}
				return res, nil
			case username == "aValidUser":
				res, err := httpmock.NewJsonResponse(http.StatusOK, map[string]interface{}{
					"name": "aValidUser",
					"id":   "02ebc15aa0ef4db1baa44422b61bc0ed",
				})
				if err != nil {
					panic(err)
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

	_, err = y.ProfileByUsername(ctx, "missingUser", time.Time{})
	if err == nil {
		t.Logf("unexpected nil error")
		t.Fail()
	} else {
		if !errors.Is(err, ErrNoSuchProfile) {
			t.Logf("unexpected %s error", reflect.TypeOf(err).Name())
			t.Fail()
		}
	}

	_, err = y.ProfileByUsername(ctx, "", time.Time{})
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
	_, err = y.ProfileByUsername(ctx, invalidName, time.Time{})
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

func TestClient_ProfileByUsernameBulk(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder(
		http.MethodPost, `=~^https://api\.mojang\.com/profiles/minecraft`,
		func(req *http.Request) (*http.Response, error) {
			data, err := io.ReadAll(req.Body)
			if err != nil {
				panic(err)
			}

			names := make([]string, 0, 10)
			if err = json.Unmarshal(data, &names); err != nil {
				panic(err)
			}

			if len(names) == 0 {
				res, err := httpmock.NewJsonResponse(http.StatusBadRequest, map[string]interface{}{
					"error":        "IllegalArgumentException",
					"errorMessage": "profileNames is marked non-null but is null",
				})
				if err != nil {
					panic(err)
				}
				return res, nil
			} else if len(names) > 10 {
				res, err := httpmock.NewJsonResponse(http.StatusBadRequest, map[string]interface{}{
					"error":        "IllegalArgumentException",
					"errorMessage": "Not more that 10 profile name per call is allowed.",
				})
				if err != nil {
					panic(err)
				}
				return res, nil
			}

			if len(names) == 2 {
				if names[0] == "validUserFoo" && names[1] == "validUserBar" {
					res, err := httpmock.NewJsonResponse(http.StatusOK, []map[string]interface{}{
						{
							"name": "validUserFoo",
							"id":   "02ebc15aa0ef4db1baa44422b61bc0ed",
						},
						{
							"name": "validUserBar",
							"id":   "1c05c268d9894ba0b144ea326065c9ef",
						},
					})
					if err != nil {
						panic(err)
					}
					return res, nil
				} else if names[0] == "validUser" && names[1] == "invalidUser" {
					res, err := httpmock.NewJsonResponse(http.StatusOK, []map[string]interface{}{
						{
							"name": "validUser",
							"id":   "02ebc15aa0ef4db1baa44422b61bc0ed",
						},
					})
					if err != nil {
						panic(err)
					}
					return res, nil
				} else if names[0] == "invalidUserFoo" && names[1] == "invalidUserBar" {
					res, err := httpmock.NewJsonResponse(http.StatusOK, []map[string]interface{}{})
					if err != nil {
						panic(err)
					}
					return res, nil
				}
			}

			if len(names) == 1 && len(names[0]) > 25 {
				return httpmock.NewJsonResponse(http.StatusBadRequest, map[string]interface{}{
					"error":        "BadRequestException",
					"errorMessage": fmt.Sprintf("%s is invalid", names[0]),
				})
			}

			return nil, nil
		},
	)

	y := New()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	u, err := y.ProfileByUsernameBulk(ctx, []string{"validUserFoo", "validUserBar"})
	if err != nil {
		t.Logf("unexpected error %s", err)
		t.Fail()
	} else {
		if len(u) != 2 {
			t.Logf("unexpected user count %d", len(u))
			t.Fail()
		} else {
			if u[0].Name != "validUserFoo" {
				t.Logf("unexpected name %s", u[0].Name)
				t.Fail()
			}
			if u[0].ID.String() != "02ebc15a-a0ef-4db1-baa4-4422b61bc0ed" {
				t.Logf("unexpected uuid %s", u[0].ID)
				t.Fail()
			}

			if u[1].Name != "validUserBar" {
				t.Logf("unexpected name %s", u[1].Name)
				t.Fail()
			}
			if u[1].ID.String() != "1c05c268-d989-4ba0-b144-ea326065c9ef" {
				t.Logf("unexpected uuid %s", u[1].ID)
				t.Fail()
			}
		}
	}

	u, err = y.ProfileByUsernameBulk(ctx, []string{"validUser", "invalidUser"})
	if err != nil {
		t.Logf("unexpected error %s", err)
		t.Fail()
	} else {
		if len(u) != 1 {
			t.Logf("unexpected user count %d", len(u))
			t.Fail()
		} else {
			if u[0].Name != "validUser" {
				t.Logf("unexpected name %s", u[0].Name)
				t.Fail()
			}
			if u[0].ID.String() != "02ebc15a-a0ef-4db1-baa4-4422b61bc0ed" {
				t.Logf("unexpected uuid %s", u[0].ID)
				t.Fail()
			}
		}
	}

	u, err = y.ProfileByUsernameBulk(ctx, []string{"invalidUserFoo", "invalidUserBar"})
	if err != nil {
		t.Logf("unexpected error %s", err)
		t.Fail()
	} else {
		if len(u) != 0 {
			t.Logf("unexpected user count %d", len(u))
			t.Fail()
		}
	}

	_, err = y.ProfileByUsernameBulk(ctx, []string{})
	if err == nil {
		t.Logf("unexpected nil error")
		t.Fail()
	} else {
		if err, ok := err.(*RequestError); !ok {
			t.Logf("unexpected %s error", reflect.TypeOf(err).String())
			t.Fail()
		} else {
			if err.Type != "IllegalArgumentException" {
				t.Logf("unexpected error type %s", err.Type)
				t.Fail()
			}
			if err.Message != "profileNames is marked non-null but is null" {
				t.Logf("unexpeced error message %s", err.Message)
				t.Fail()
			}
		}
	}

	_, err = y.ProfileByUsernameBulk(ctx, []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"})
	if err == nil {
		t.Logf("unexpected nil error")
		t.Fail()
	} else {
		if err, ok := err.(*RequestError); !ok {
			t.Logf("unexpected %s error", reflect.TypeOf(err).String())
			t.Fail()
		} else {
			if err.Type != "IllegalArgumentException" {
				t.Logf("unexpected error type %s", err.Type)
				t.Fail()
			}
			if err.Message != "Not more that 10 profile name per call is allowed." {
				t.Logf("unexpeced error message %s", err.Message)
				t.Fail()
			}
		}
	}

	invalidName := strings.Repeat("0", 26)
	_, err = y.ProfileByUsernameBulk(ctx, []string{invalidName})
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
