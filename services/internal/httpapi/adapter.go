package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"airpath/services/internal/application"
	"airpath/services/internal/domain"

	"github.com/aws/aws-lambda-go/events"
)

const (
	allowOriginHeader  = "access-control-allow-origin"
	allowMethodsHeader = "access-control-allow-methods"
	allowHeadersHeader = "access-control-allow-headers"
	contentTypeHeader  = "content-type"
	utcTimestampLayout = "2006-01-02T15:04:05Z"
)

type UseCases interface {
	SearchFlights(context.Context, application.SearchFlightsInput) (application.FlightSearchResponse, error)
	GetFlightDetail(context.Context, application.FlightDetailInput) (application.FlightDetailResponse, error)
	GetFlightMapData(context.Context, application.FlightMapDataInput) (application.FlightMapDataResponse, error)
	GetFlightPositions(context.Context, application.FlightPositionsInput) (application.FlightPositionsResponse, error)
	RequestFlightRefresh(context.Context, application.FlightRefreshInput) (application.FlightRefreshResponse, error)
	GetUsageStatus(context.Context, application.UsageStatusInput) (application.UsageStatus, error)
}

type Adapter struct {
	app UseCases
}

func NewAdapter(app UseCases) *Adapter {
	return &Adapter{app: app}
}

func (a *Adapter) Handle(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	if request.RequestContext.HTTP.Method == http.MethodOptions {
		return preflightResponse(), nil
	}

	switch request.RawPath {
	case routeFlightSearch:
		if unsupportedReadMethod(request.RequestContext.HTTP.Method) {
			return notFoundResponse(request.RequestContext.RequestID)
		}
		ident := strings.TrimSpace(request.QueryStringParameters["ident"])
		if _, ok := request.QueryStringParameters["date"]; ok || len(ident) < 2 {
			return jsonResponse(http.StatusBadRequest, map[string]any{
				"error": application.MapApplicationError(application.ErrValidation, request.RequestContext.RequestID),
			})
		}
		response, err := a.app.SearchFlights(ctx, application.SearchFlightsInput{
			Ident:     ident,
			CheckedAt: nowUTC(),
		})
		if err != nil {
			apiErr := application.MapApplicationError(err, request.RequestContext.RequestID)
			return jsonResponse(statusFor(apiErr.Code), map[string]any{"error": apiErr})
		}
		return jsonResponse(http.StatusOK, response)
	case routeUsageStatus:
		if unsupportedReadMethod(request.RequestContext.HTTP.Method) {
			return notFoundResponse(request.RequestContext.RequestID)
		}
		response, err := a.app.GetUsageStatus(ctx, application.UsageStatusInput{
			CheckedAt: nowUTC(),
		})
		if err != nil {
			apiErr := application.MapApplicationError(err, request.RequestContext.RequestID)
			return jsonResponse(statusFor(apiErr.Code), map[string]any{"error": apiErr})
		}
		return jsonResponse(http.StatusOK, response)
	default:
		return a.handleFlightResource(ctx, request)
	}
}

func (a *Adapter) handleFlightResource(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	flightID, suffix, ok := splitFlightResource(request.RawPath)
	if !ok {
		return notFoundResponse(request.RequestContext.RequestID)
	}

	checkedAt := nowUTC()
	switch {
	case suffix == "" && request.RequestContext.HTTP.Method == http.MethodGet:
		response, err := a.app.GetFlightDetail(ctx, application.FlightDetailInput{
			FlightID:  domain.FlightID(flightID),
			CheckedAt: checkedAt,
		})
		if err != nil {
			apiErr := application.MapApplicationError(err, request.RequestContext.RequestID)
			return jsonResponse(statusFor(apiErr.Code), map[string]any{"error": apiErr})
		}
		return jsonResponse(http.StatusOK, response)
	case suffix == flightMapDataSuffix && request.RequestContext.HTTP.Method == http.MethodGet:
		if hasAnyQuery(request.QueryStringParameters, "include", "simplify") {
			apiErr := application.MapApplicationError(application.ErrValidation, request.RequestContext.RequestID)
			return jsonResponse(http.StatusBadRequest, map[string]any{"error": apiErr})
		}
		response, err := a.app.GetFlightMapData(ctx, application.FlightMapDataInput{
			FlightID:  domain.FlightID(flightID),
			CheckedAt: checkedAt,
		})
		if err != nil {
			apiErr := application.MapApplicationError(err, request.RequestContext.RequestID)
			return jsonResponse(statusFor(apiErr.Code), map[string]any{"error": apiErr})
		}
		return jsonResponse(http.StatusOK, response)
	case suffix == flightPositionsSuffix && request.RequestContext.HTTP.Method == http.MethodGet:
		input, err := parsePositionsQuery(request.QueryStringParameters)
		if err != nil {
			apiErr := application.MapApplicationError(application.ErrValidation, request.RequestContext.RequestID)
			return jsonResponse(http.StatusBadRequest, map[string]any{"error": apiErr})
		}
		input.FlightID = domain.FlightID(flightID)
		input.CheckedAt = checkedAt
		response, err := a.app.GetFlightPositions(ctx, input)
		if err != nil {
			apiErr := application.MapApplicationError(err, request.RequestContext.RequestID)
			return jsonResponse(statusFor(apiErr.Code), map[string]any{"error": apiErr})
		}
		return jsonResponse(http.StatusOK, response)
	case suffix == flightRefreshSuffix && request.RequestContext.HTTP.Method == http.MethodPost:
		input, err := parseRefreshRequest(request.Body)
		if err != nil {
			apiErr := application.MapApplicationError(application.ErrValidation, request.RequestContext.RequestID)
			return jsonResponse(http.StatusBadRequest, map[string]any{"error": apiErr})
		}
		response, err := a.app.RequestFlightRefresh(ctx, application.FlightRefreshInput{
			FlightID:     domain.FlightID(flightID),
			TaskTypes:    input.TaskTypes,
			ClientReason: input.ClientReason,
			RequestedAt:  checkedAt,
		})
		if err != nil {
			apiErr := application.MapApplicationError(err, request.RequestContext.RequestID)
			return jsonResponse(statusFor(apiErr.Code), map[string]any{"error": apiErr})
		}
		return jsonResponse(http.StatusAccepted, response)
	default:
		return notFoundResponse(request.RequestContext.RequestID)
	}
}

func parsePositionsQuery(query map[string]string) (application.FlightPositionsInput, error) {
	input := application.FlightPositionsInput{Limit: 200}
	if hasAnyQuery(query, "quality") {
		return application.FlightPositionsInput{}, application.ErrValidation
	}
	if rawSince, ok := query["since"]; ok {
		since := strings.TrimSpace(rawSince)
		if since == "" {
			return application.FlightPositionsInput{}, application.ErrValidation
		}
		parsedSince, err := time.Parse(time.RFC3339Nano, since)
		if err != nil {
			return application.FlightPositionsInput{}, application.ErrValidation
		}
		// Storage range keys use second-precision UTC strings, so normalize accepted
		// RFC3339 inputs before passing them into the application layer.
		normalizedSince := parsedSince.UTC().Format(utcTimestampLayout)
		input.Since = &normalizedSince
	}
	if rawLimit, ok := query["limit"]; ok {
		limitText := strings.TrimSpace(rawLimit)
		if limitText == "" {
			return application.FlightPositionsInput{}, application.ErrValidation
		}
		limit, err := strconv.Atoi(limitText)
		if err != nil || limit < 1 {
			return application.FlightPositionsInput{}, application.ErrValidation
		}
		if limit > 500 {
			limit = 500
		}
		input.Limit = limit
	}
	return input, nil
}

func hasAnyQuery(query map[string]string, names ...string) bool {
	for _, name := range names {
		if _, ok := query[name]; ok {
			return true
		}
	}
	return false
}

type refreshRequest struct {
	TaskTypes    []application.FetchTaskType `json:"taskTypes"`
	ClientReason application.FetchReason     `json:"clientReason"`
}

func parseRefreshRequest(body string) (refreshRequest, error) {
	var request refreshRequest
	decoder := json.NewDecoder(bytes.NewBufferString(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return refreshRequest{}, err
	}
	var trailing struct{}
	// Decode one extra value so concatenated JSON objects cannot be accepted as a
	// single refresh command.
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err != nil {
			return refreshRequest{}, err
		}
		return refreshRequest{}, application.ErrValidation
	}
	if len(request.TaskTypes) == 0 || request.ClientReason == "" {
		return refreshRequest{}, application.ErrValidation
	}
	if !application.IsValidFetchReason(request.ClientReason) {
		return refreshRequest{}, application.ErrValidation
	}
	if !isPublicRefreshReason(request.ClientReason) {
		return refreshRequest{}, application.ErrValidation
	}
	seenTaskTypes := make(map[application.FetchTaskType]struct{}, len(request.TaskTypes))
	for _, taskType := range request.TaskTypes {
		if !application.IsValidFetchTaskType(taskType) {
			return refreshRequest{}, application.ErrValidation
		}
		if taskType == application.FetchTaskSummary {
			return refreshRequest{}, application.ErrValidation
		}
		if _, ok := seenTaskTypes[taskType]; ok {
			return refreshRequest{}, application.ErrValidation
		}
		seenTaskTypes[taskType] = struct{}{}
	}
	return request, nil
}

func isPublicRefreshReason(reason application.FetchReason) bool {
	switch reason {
	case application.FetchReasonUserOpenedDetail, application.FetchReasonUserManualRefresh:
		return true
	default:
		return false
	}
}

func splitFlightResource(path string) (flightID string, suffix string, ok bool) {
	rest := strings.TrimPrefix(path, flightResourcePrefix)
	if rest == path || rest == "" {
		return "", "", false
	}
	parts := strings.Split(rest, "/")
	if len(parts) == 1 {
		if parts[0] == "" {
			return "", "", false
		}
		decodedFlightID, err := url.PathUnescape(parts[0])
		if err != nil || decodedFlightID == "" {
			return "", "", false
		}
		return decodedFlightID, "", true
	}
	if len(parts) == 2 {
		if parts[0] == "" || parts[1] == "" {
			return "", "", false
		}
		decodedFlightID, err := url.PathUnescape(parts[0])
		if err != nil || decodedFlightID == "" {
			return "", "", false
		}
		return decodedFlightID, parts[1], true
	}
	return "", "", false
}

func nowUTC() string {
	return time.Now().UTC().Format(utcTimestampLayout)
}

func jsonResponse(statusCode int, body any) (events.APIGatewayV2HTTPResponse, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	return events.APIGatewayV2HTTPResponse{
		StatusCode: statusCode,
		Headers:    responseHeaders("application/json"),
		Body:       string(payload),
	}, nil
}

func preflightResponse() events.APIGatewayV2HTTPResponse {
	return events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusNoContent,
		Headers:    responseHeaders(""),
	}
}

func responseHeaders(contentType string) map[string]string {
	headers := map[string]string{
		allowOriginHeader:  "*",
		allowMethodsHeader: strings.Join([]string{http.MethodGet, http.MethodPost, http.MethodOptions}, ","),
		allowHeadersHeader: "accept,content-type",
	}
	if contentType != "" {
		headers[contentTypeHeader] = contentType
	}
	return headers
}

func notFoundResponse(requestID string) (events.APIGatewayV2HTTPResponse, error) {
	apiErr := application.MapApplicationError(application.ErrNotFound, requestID)
	return jsonResponse(http.StatusNotFound, map[string]any{"error": apiErr})
}

func unsupportedReadMethod(method string) bool {
	return method != http.MethodGet
}

func statusFor(code application.ApiErrorCode) int {
	switch code {
	case application.ApiErrorFlightAwareBudgetExceeded, application.ApiErrorFlightAwareFetchDisabled:
		return http.StatusServiceUnavailable
	case application.ApiErrorFlightAwareRateLimited:
		return http.StatusTooManyRequests
	case application.ApiErrorStaleCacheUnavailable:
		return http.StatusNotFound
	case application.ApiErrorValidationFailed:
		return http.StatusBadRequest
	default:
		return http.StatusBadGateway
	}
}
