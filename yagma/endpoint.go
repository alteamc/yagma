package yagma

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// General errors

var (
	ErrUnsuccessfulTransmission = errors.New("an error occurred during HTTP request/response transmission")
	ErrInvalidJSON              = errors.New("unable to process JSON data")
	ErrUnknownStatusCode        = errors.New("unknown status code")
)

type RequestError struct {
	Type             string `json:"error"`
	Message          string `json:"errorMessage"`
	DeveloperMessage string `json:"developerMessage"`
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Utility methods

func parseRequestError(res *http.Response) error {
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrUnsuccessfulTransmission, err)
	}

	reqErr := &RequestError{}
	if err = json.Unmarshal(data, reqErr); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidJSON, err)
	}

	return reqErr
}

func readResBody(res *http.Response) ([]byte, error) {
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnsuccessfulTransmission, err)
	}
	return data, nil
}

func parseRes(res *http.Response, dest interface{}) error {
	data, err := readResBody(res)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidJSON, err)
	}
	return nil
}

// Endpoints

var ErrNoSuchProfile = errors.New("profile not found")

// ProfileByUsername performs a lookup of Profile for provided username at provided timestamp.
func (c *Client) ProfileByUsername(ctx context.Context, username string, timestamp time.Time) (*Profile, error) {
	v := make(url.Values)
	if !timestamp.IsZero() {
		v.Add("at", strconv.FormatInt(timestamp.UnixMilli(), 10))
	}

	res, err := c.sendHTTPReq(ctx, http.MethodGet, c.baseURL.API("/users/profiles/minecraft/"+username, v), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnsuccessfulTransmission, err)
	}
	defer func() { _ = res.Body.Close() }()

	switch res.StatusCode {
	case http.StatusOK:
		m := &profileJSONMapping{}
		if err = parseRes(res, m); err != nil {
			return nil, err
		}
		return m.Wrap(), nil
	case http.StatusNoContent:
		return nil, fmt.Errorf("%w: %s", ErrNoSuchProfile, username)
	case http.StatusBadRequest, http.StatusNotFound:
		return nil, parseRequestError(res)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownStatusCode, res.Status)
	}
}

// ProfileByUsernameBulk performs a bulk lookup of Profiles for provided array of usernames.
func (c *Client) ProfileByUsernameBulk(ctx context.Context, usernames []string) ([]*Profile, error) {
	bodyBytes, err := json.Marshal(usernames)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnsuccessfulTransmission, err)
	}

	res, err := c.sendHTTPReq(ctx, http.MethodPost, c.baseURL.API("/profiles/minecraft", nil), nil, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnsuccessfulTransmission, err)
	}
	defer func() { _ = res.Body.Close() }()

	switch res.StatusCode {
	case http.StatusOK:
		profiles := make([]*Profile, 0, len(usernames))
		if err = parseRes(res, &profiles); err != nil {
			return nil, err
		}
		return profiles, nil
	case http.StatusBadRequest:
		return nil, parseRequestError(res)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownStatusCode, res.Status)
	}
}

// NameHistoryByUUID performs a username history query for provided UUID.
func (c *Client) NameHistoryByUUID(ctx context.Context, uuid uuid.UUID) ([]*NameHistoryRecord, error) {
	res, err := c.sendHTTPReq(ctx, http.MethodGet, c.baseURL.API("/user/profiles/"+uuid.String()+"/names", nil), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnsuccessfulTransmission, err)
	}
	defer func() { _ = res.Body.Close() }()

	switch res.StatusCode {
	case http.StatusOK:
		records := make(nameHistoryRecordJSONMappingArray, 0, 8)
		if err = parseRes(res, &records); err != nil {
			return nil, err
		}
		return records.Wrap(), nil
	case http.StatusNoContent:
		return nil, fmt.Errorf("%w: %s", ErrNoSuchProfile, uuid)
	case http.StatusBadRequest:
		return nil, parseRequestError(res)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownStatusCode, res.Status)
	}
}

// ProfileByUUID performs a lookup of Profile for provided UUID.
func (c *Client) ProfileByUUID(ctx context.Context, uuid uuid.UUID) (*Profile, error) {
	res, err := c.sendHTTPReq(ctx, http.MethodGet, c.baseURL.SessionServer("/session/minecraft/profile/"+strings.Replace(uuid.String(), "-", "", -1), nil), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnsuccessfulTransmission, err)
	}
	defer func() { _ = res.Body.Close() }()

	switch res.StatusCode {
	case http.StatusOK:
		m := &profileJSONMapping{}
		if err = parseRes(res, m); err != nil {
			return nil, err
		}
		return m.Wrap(), nil
	case http.StatusNoContent:
		return nil, ErrNoSuchProfile
	case http.StatusBadRequest:
		return nil, parseRequestError(res)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownStatusCode, res.Status)
	}
}

// BlockedServerHashes performs a blocked server hash query.
func (c *Client) BlockedServerHashes(ctx context.Context) ([]string, error) {
	res, err := c.sendHTTPReq(ctx, http.MethodGet, c.baseURL.SessionServer("/blockedservers", nil), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnsuccessfulTransmission, err)
	}
	defer func() { _ = res.Body.Close() }()

	switch res.StatusCode {
	case http.StatusOK:
		data, err := readResBody(res)
		if err != nil {
			return nil, err
		}
		return strings.Split(string(data), "\n"), nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownStatusCode, res.Status)
	}
}

// Statistics performs Mojang sell statistics query for provided array of MetricKeys.
func (c *Client) Statistics(ctx context.Context, keys []MetricKey) (*Statistics, error) {
	header := make(http.Header)
	header.Add("Content-Type", "application/json")

	bodyBytes, err := json.Marshal(map[string][]MetricKey{"metricKeys": keys})
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnsuccessfulTransmission, err)
	}

	res, err := c.sendHTTPReq(ctx, http.MethodPost, c.baseURL.API("/orders/statistics", nil), header, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnsuccessfulTransmission, err)
	}
	defer func() { _ = res.Body.Close() }()

	switch res.StatusCode {
	case http.StatusOK:
		s := &Statistics{}
		if err = parseRes(res, s); err != nil {
			return nil, err
		}
		return s, nil
	case http.StatusBadRequest:
		return nil, parseRequestError(res)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownStatusCode, res.Status)
	}
}
