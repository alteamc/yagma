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
	HTTPError   = errors.New("failed to process HTTP request")
	JSONError   = errors.New("failed to parse JSON response")
	StatusError = errors.New("unknown status code")
)

type RequestError struct {
	Type    string `json:"error"`
	Message string `json:"errorMessage"`
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Utility methods

func parseRequestError(res *http.Response) error {
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("%w: %s", HTTPError, err)
	}

	reqErr := &RequestError{}
	if err = json.Unmarshal(data, reqErr); err != nil {
		return fmt.Errorf("%w: %s", JSONError, err)
	}

	return reqErr
}

func readResBody(res *http.Response) ([]byte, error) {
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", HTTPError, err)
	}
	return data, nil
}

func parseRes(res *http.Response, dest interface{}) error {
	data, err := readResBody(res)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("%w: %s", JSONError, err)
	}
	return nil
}

// Profile by username

var ProfileNotFound = errors.New("user not found")

func (c *Client) ProfileByUsername(ctx context.Context, username string, timestamp time.Time) (*Profile, error) {
	v := make(url.Values)
	if !timestamp.IsZero() {
		v.Add("at", strconv.FormatInt(timestamp.UnixMilli(), 10))
	}

	res, err := c.sendHTTPReq(ctx, http.MethodGet, c.baseURL.API("/users/profiles/minecraft/"+username, v), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", HTTPError, err)
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
		return nil, fmt.Errorf("%w: %s", ProfileNotFound, username)
	case http.StatusBadRequest:
		return nil, parseRequestError(res)
	default:
		return nil, fmt.Errorf("%w: %s", StatusError, res.Status)
	}
}

// Profile by username (bulk)

func (c *Client) ProfileByUsernameBulk(ctx context.Context, usernames []string) ([]*Profile, error) {
	bodyBytes, err := json.Marshal(usernames)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", HTTPError, err)
	}

	res, err := c.sendHTTPReq(ctx, http.MethodPost, c.baseURL.API("/profiles/minecraft", nil), nil, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", HTTPError, err)
	}
	defer func() { _ = res.Body.Close() }()

	switch res.StatusCode {
	case http.StatusOK:
		profiles := make([]*Profile, 0, len(usernames))
		if err = parseRes(res, profiles); err != nil {
			return nil, err
		}
		return profiles, nil
	case http.StatusBadRequest:
		return nil, parseRequestError(res)
	default:
		return nil, fmt.Errorf("%w: %s", StatusError, res.Status)
	}
}

// Name history by UUID

func (c *Client) NameHistoryByUUID(ctx context.Context, uuid uuid.UUID) ([]*NameHistoryRecord, error) {
	res, err := c.sendHTTPReq(ctx, http.MethodGet, c.baseURL.API("/user/profiles/"+uuid.String()+"/names", nil), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", HTTPError, err)
	}
	defer func() { _ = res.Body.Close() }()

	switch res.StatusCode {
	case http.StatusOK:
		records := make(nameHistoryRecordJSONMappingArray, 0, 8)
		if err = parseRes(res, records); err != nil {
			return nil, err
		}
		return records.Wrap(), nil
	case http.StatusNoContent:
		return nil, fmt.Errorf("%w: %s", ProfileNotFound, uuid)
	case http.StatusBadRequest:
		return nil, parseRequestError(res)
	default:
		return nil, fmt.Errorf("%w: %s", StatusError, res.Status)
	}
}

// Profile with skin/cape by UUID

func (c *Client) ProfileByUUID(ctx context.Context, uuid uuid.UUID) (*Profile, error) {
	res, err := c.sendHTTPReq(ctx, http.MethodGet, c.baseURL.SessionServer("/session/minecraft/profile/"+strings.Replace(uuid.String(), "-", "", -1), nil), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", HTTPError, err)
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
		return nil, ProfileNotFound
	case http.StatusBadRequest:
		return nil, parseRequestError(res)
	default:
		return nil, fmt.Errorf("%w: %s", StatusError, res.Status)
	}
}

// Blocked servers

func (c *Client) BlockedServerHashes(ctx context.Context) ([]string, error) {
	res, err := c.sendHTTPReq(ctx, http.MethodGet, c.baseURL.SessionServer("/blockedservers", nil), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", HTTPError, err)
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
		return nil, fmt.Errorf("%w: %s", StatusError, res.Status)
	}
}

// Statistics

func (c *Client) Statistics(ctx context.Context, keys []MetricKey) (*Statistics, error) {
	header := make(http.Header)
	header.Add("Content-Type", "application/json")

	bodyBytes, err := json.Marshal(map[string][]MetricKey{"metricKeys": keys})
	if err != nil {
		return nil, fmt.Errorf("%w: %s", HTTPError, err)
	}

	res, err := c.sendHTTPReq(ctx, http.MethodPost, c.baseURL.API("/orders/statistics", nil), header, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", HTTPError, err)
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
		return nil, fmt.Errorf("%w: %s", StatusError, res.Status)
	}
}
