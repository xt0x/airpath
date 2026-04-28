package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"airpath/services/internal/application"

	"github.com/aws/aws-lambda-go/events"
)

type SearchUseCase interface {
	SearchFlights(context.Context, application.SearchFlightsInput) (application.FlightSearchResponse, error)
}

type Adapter struct {
	app SearchUseCase
}

func NewAdapter(app SearchUseCase) *Adapter {
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
			CheckedAt: time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		})
		if err != nil {
			apiErr := application.MapApplicationError(err, request.RequestContext.RequestID)
			return jsonResponse(statusFor(apiErr.Code), map[string]any{"error": apiErr})
		}
		return jsonResponse(http.StatusOK, response)
	default:
		apiErr := application.MapApplicationError(application.ErrNotFound, request.RequestContext.RequestID)
		return jsonResponse(http.StatusNotFound, map[string]any{"error": apiErr})
	}
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
