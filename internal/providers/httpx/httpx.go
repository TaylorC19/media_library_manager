package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"media_library_manager/internal/providers/providererrors"
)

type QueryValue = any

type GetJSONOptions struct {
	Query   map[string]QueryValue
	Headers map[string]string
	Timeout time.Duration
}

func GetJSON(ctx context.Context, client *http.Client, provider, operation, rawURL string, options GetJSONOptions, out any) error {
	requestURL, err := url.Parse(rawURL)
	if err != nil {
		return providererrors.New(providererrors.Options{
			Provider:  provider,
			Operation: operation,
			Code:      providererrors.CodeConfiguration,
			Message:   "Provider URL is invalid.",
			Cause:     err,
		})
	}

	query := requestURL.Query()
	for key, value := range options.Query {
		switch v := value.(type) {
		case nil:
			continue
		case string:
			if strings.TrimSpace(v) == "" {
				continue
			}
			query.Set(key, v)
		default:
			stringValue := strings.TrimSpace(toString(v))
			if stringValue == "" {
				continue
			}
			query.Set(key, stringValue)
		}
	}
	requestURL.RawQuery = query.Encode()

	timeout := options.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return providererrors.New(providererrors.Options{
			Provider:   provider,
			Operation:  operation,
			Code:       providererrors.CodeUnknown,
			Message:    "Provider request could not be created.",
			Cause:      err,
			RequestURL: safeURL(requestURL),
		})
	}
	req.Header.Set("Accept", "application/json")
	for key, value := range options.Headers {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		req.Header.Set(key, value)
	}

	httpClient := client
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return mapTransportError(provider, operation, safeURL(requestURL), err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return mapStatusError(provider, operation, safeURL(requestURL), res.StatusCode)
	}

	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		return providererrors.New(providererrors.Options{
			Provider:   provider,
			Operation:  operation,
			Code:       providererrors.CodeInvalidResponse,
			Message:    "Provider returned an invalid response.",
			Cause:      err,
			RequestURL: safeURL(requestURL),
		})
	}

	return nil
}

func mapStatusError(provider, operation, requestURL string, statusCode int) error {
	code := providererrors.CodeUnknown
	message := "Provider request failed."

	switch {
	case statusCode == http.StatusNotFound:
		code = providererrors.CodeNotFound
		message = "Provider record was not found."
	case statusCode == http.StatusTooManyRequests:
		code = providererrors.CodeRateLimited
		message = "Provider rate limit was reached."
	case statusCode == http.StatusUnauthorized:
		code = providererrors.CodeUnauthorized
		message = "Provider credentials were rejected."
	case statusCode == http.StatusForbidden:
		code = providererrors.CodeForbidden
		message = "Provider request was forbidden."
	case statusCode == http.StatusRequestTimeout:
		code = providererrors.CodeTimeout
		message = "Provider request timed out."
	case statusCode >= 500:
		code = providererrors.CodeUpstream
		message = "Provider service is temporarily unavailable."
	default:
		code = providererrors.CodeUnavailable
		message = "Provider request failed."
	}

	return providererrors.New(providererrors.Options{
		Provider:   provider,
		Operation:  operation,
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Retryable:  providererrors.IsRetryableCode(code),
		RequestURL: requestURL,
	})
}

func mapTransportError(provider, operation, requestURL string, err error) error {
	if errorsIsTimeout(err) {
		return providererrors.New(providererrors.Options{
			Provider:   provider,
			Operation:  operation,
			Code:       providererrors.CodeTimeout,
			Message:    "Provider request timed out.",
			Cause:      err,
			Retryable:  true,
			RequestURL: requestURL,
		})
	}

	return providererrors.New(providererrors.Options{
		Provider:   provider,
		Operation:  operation,
		Code:       providererrors.CodeNetwork,
		Message:    "Provider network request failed.",
		Cause:      err,
		Retryable:  true,
		RequestURL: requestURL,
	})
}

func errorsIsTimeout(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func safeURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	return u.Scheme + "://" + u.Host + u.Path
}

func toString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}
