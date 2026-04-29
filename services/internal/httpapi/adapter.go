package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"airpath/services/internal/application"
	"airpath/services/internal/domain"

	"github.com/aws/aws-lambda-go/events"
)

type UseCases interface {
	SearchFlights(context.Context, application.SearchFlightsInput) (application.FlightSearchResponse, error)
	GetFlightDetail(context.Context, application.FlightDetailInput) (application.FlightDetailResponse, error)
	GetFlightMapData(context.Context, application.FlightMapDataInput) (application.FlightMapDataResponse, error)
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
	switch request.RawPath {
	case "/v1/flights/search":
		ident := request.QueryStringParameters["ident"]
		if ident == "" {
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
	case "/v1/usage/status":
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
		apiErr := application.MapApplicationError(application.ErrNotFound, request.RequestContext.RequestID)
		return jsonResponse(http.StatusNotFound, map[string]any{"error": apiErr})
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
	case suffix == "map-data" && request.RequestContext.HTTP.Method == http.MethodGet:
		response, err := a.app.GetFlightMapData(ctx, application.FlightMapDataInput{
			FlightID:  domain.FlightID(flightID),
			CheckedAt: checkedAt,
		})
		if err != nil {
			apiErr := application.MapApplicationError(err, request.RequestContext.RequestID)
			return jsonResponse(statusFor(apiErr.Code), map[string]any{"error": apiErr})
		}
		return jsonResponse(http.StatusOK, response)
	case suffix == "refresh" && request.RequestContext.HTTP.Method == http.MethodPost:
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
		apiErr := application.MapApplicationError(application.ErrNotFound, request.RequestContext.RequestID)
		return jsonResponse(http.StatusNotFound, map[string]any{"error": apiErr})
	}
}

type refreshRequest struct {
	TaskTypes    []application.FetchTaskType `json:"taskTypes"`
	ClientReason application.FetchReason     `json:"clientReason"`
}

func parseRefreshRequest(body string) (refreshRequest, error) {
	var request refreshRequest
	if err := json.Unmarshal([]byte(body), &request); err != nil {
		return refreshRequest{}, err
	}
	if len(request.TaskTypes) == 0 || request.ClientReason == "" {
		return refreshRequest{}, application.ErrValidation
	}
	return request, nil
}

func splitFlightResource(path string) (flightID string, suffix string, ok bool) {
	rest := strings.TrimPrefix(path, "/v1/flights/")
	if rest == path || rest == "" {
		return "", "", false
	}
	parts := strings.Split(rest, "/")
	if len(parts) == 1 {
		return parts[0], "", true
	}
	if len(parts) == 2 {
		return parts[0], parts[1], true
	}
	return "", "", false
}

func nowUTC() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

func jsonResponse(statusCode int, body any) (events.APIGatewayV2HTTPResponse, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	return events.APIGatewayV2HTTPResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"content-type": "application/json",
		},
		Body: string(payload),
	}, nil
}

func statusFor(code application.ApiErrorCode) int {
	switch code {
	case application.ApiErrorFlightAwareBudgetExceeded, application.ApiErrorFlightAwareFetchDisabled:
		return http.StatusServiceUnavailable
	case application.ApiErrorFlightAwareRateLimited:
		return http.StatusTooManyRequests
	case application.ApiErrorStaleCacheUnavailable:
		return http.StatusNotFound
	default:
		return http.StatusBadGateway
	}
}
