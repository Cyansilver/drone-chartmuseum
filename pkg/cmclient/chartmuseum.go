package cmclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"crypto/tls"
	"strings"
	"time"
)

const (
	userAgent = "drone-chartmuseum"
)

type (
	// A Client manages communication with the ChartMuseum API.
	Client struct {
		// Base URL for API requests. BaseURL should
		// always be specified with a trailing slash.
		BaseURL *url.URL

		//basic authentication username
		Username string

		//basic authentication password
		Password string

		// User agent used when communicating with the ChartMuseum API.
		UserAgent string

		httpClient *http.Client

		common service // Reuse a single struct instead of allocating one for each service on the heap.

		ChartService *ChartService
	}

	service struct {
		client *Client
	}

	// Response wraps http.Response and decodes ChartMuseum response
	Response struct {
		*http.Response

		Message string `json:"message,omitempty"`
		Error   string `json:"error,omitempty"`
		Saved   bool   `json:"saved,omitempty"`
		Deleted int    `json:"deleted,omitempty"`
	}
)

// NewClient returns a new ChartMuseum API client with provided base URL
// If trailing slash is missing from base URL, one is added automatically.
// If a nil httpClient is provided, http.DefaultClient will be used.
func NewClient(baseURL string, httpClient *http.Client, username string, password string, skipTlsVerify bool) (*Client, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("ChartMuseum API - base URL can not be blank")
	}
	baseEndpoint, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(baseEndpoint.Path, "/") {
		baseEndpoint.Path += "/"
	}
	if httpClient == nil {
		var tr = &http.Transport{
            TLSClientConfig: &tls.Config{InsecureSkipVerify: skipTlsVerify},
		}
	    httpClient= &http.Client{
	      Timeout: time.Second * 5,
	      Transport: tr,
	    }
	}

	c := &Client{httpClient: httpClient, BaseURL: baseEndpoint, UserAgent: userAgent}
	c.BaseURL = baseEndpoint
	c.Username = username
	c.Password = password
	c.common.client = c
	c.ChartService = (*ChartService)(&c.common)

	return c, nil
}

// NewUploadRequest creates an upload request. A relative URL can be provided in
// urlStr, in which case it is resolved relative to the UploadURL of the Client.
// Relative URLs should always be specified without a preceding slash.
func (c *Client) NewUploadRequest(urlStr string, reader io.Reader, size int64, mediaType string) (*http.Request, error) {
	if !strings.HasSuffix(c.BaseURL.Path, "/") {
		return nil, fmt.Errorf("base URL must have a trailing slash, but %q does not", c.BaseURL)
	}
	u, err := c.BaseURL.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", u.String(), reader)
	if err != nil {
		return nil, err
	}
	req.ContentLength = size

	req.Header.Set("Content-Type", mediaType)
	req.Header.Set("User-Agent", c.UserAgent)
	if len(c.Username) > 0 && len(c.Password) > 0 {
		req.SetBasicAuth(c.Username, c.Password)
	}

	return req, nil
}

// Do sends an API request and returns the API response.
//
// The provided ctx must be non-nil. If it is canceled or times out,
// ctx.Err() will be returned.
func (c *Client) Do(ctx context.Context, req *http.Request) (*Response, error) {
	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// If we got an error, and the context has been canceled,
		// the context's error is probably more useful.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		return nil, err
	}
	return parseResponse(resp)
}

func parseResponse(r *http.Response) (*Response, error) {
	response := &Response{Response: r}
	data, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err == nil && data != nil {
		json.Unmarshal(data, response)
	}
	if c := r.StatusCode; 200 <= c && c <= 299 {
		return response, nil
	}
	return response, fmt.Errorf(response.Error)
}
