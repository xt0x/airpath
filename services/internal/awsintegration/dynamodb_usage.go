package awsintegration

import (
	"context"
	"encoding/json"
	"errors"

	"airpath/services/internal/application"
	"airpath/services/internal/flightaware"
)

func (r *DynamoDBRepository) PutUsageStatus(ctx context.Context, scope string, status application.UsageStatus) error {
	status = application.NormalizeUsageStatus(status)
	return r.client.PutItem(ctx, r.tables.UsageBudget, map[string]any{
		"budgetScope":              scope,
		"body":                     mustJSON(status),
		"environment":              status.Budget.Environment,
		"month":                    status.Budget.Month,
		"currency":                 status.Budget.Currency,
		"estimatedMonthToDateCost": status.Budget.EstimatedMonthToDateCost,
		"softStopThreshold":        status.Budget.SoftStopThreshold,
		"fetchingEnabled":          status.FetchingEnabled,
	})
}

func (r *DynamoDBRepository) PutMonthlyUsageStatus(ctx context.Context, scope application.UsageBudgetScope, status application.UsageStatus) error {
	status.Budget.Environment = scope.Environment
	status.Budget.Month = scope.Month
	return r.PutUsageStatus(ctx, scope.Key(), status)
}

func (r *DynamoDBRepository) GetUsageStatus(ctx context.Context) (application.UsageStatus, error) {
	if r.usageScope.Key() == "" {
		return application.UsageStatus{}, application.ErrValidation
	}
	return r.GetMonthlyUsageStatus(ctx, r.usageScope)
}

func (r *DynamoDBRepository) GetMonthlyUsageStatus(ctx context.Context, scope application.UsageBudgetScope) (application.UsageStatus, error) {
	item, ok, err := r.client.GetItem(ctx, r.tables.UsageBudget, "budgetScope", scope.Key())
	if err != nil {
		return application.UsageStatus{}, err
	}
	if !ok {
		return application.UsageStatus{}, application.ErrNotFound
	}
	return usageStatusFromItem(scope, item)
}

func usageStatusFromItem(scope application.UsageBudgetScope, item map[string]any) (application.UsageStatus, error) {
	var status application.UsageStatus
	if body := stringValue(item["body"]); body != "" {
		if err := json.Unmarshal([]byte(body), &status); err != nil {
			return application.UsageStatus{}, err
		}
	}
	status.Budget.Environment = scope.Environment
	status.Budget.Month = scope.Month
	if currency := stringValue(item["currency"]); currency != "" {
		status.Budget.Currency = currency
	}
	if _, ok := item["estimatedMonthToDateCost"]; ok {
		status.Budget.EstimatedMonthToDateCost = numberValue(item["estimatedMonthToDateCost"])
	}
	if _, ok := item["softStopThreshold"]; ok {
		status.Budget.SoftStopThreshold = numberValue(item["softStopThreshold"])
	}
	if fetchingEnabled, ok := item["fetchingEnabled"].(bool); ok {
		status.FetchingEnabled = fetchingEnabled
	}
	return application.NormalizeUsageStatus(status), nil
}

func (r *DynamoDBRepository) FetchingAllowed(ctx context.Context) (bool, error) {
	status, err := r.GetUsageStatus(ctx)
	if err != nil {
		if err == application.ErrNotFound {
			return true, nil
		}
		return false, err
	}
	status = application.NormalizeUsageStatus(status)
	return status.FetchingEnabled && !status.Budget.Stopped, nil
}

func (r *DynamoDBRepository) RecordFlightAwareCall(ctx context.Context, record flightaware.UsageCallRecord) error {
	if record.Phase != flightaware.UsageRecordPhaseBefore {
		return nil
	}
	scope := r.usageScope
	if scope.Key() == "" {
		return application.ErrValidation
	}
	item, err := r.client.AddUsageEstimate(ctx, r.tables.UsageBudget, scope.Key(), map[string]any{
		"environment":       scope.Environment,
		"month":             scope.Month,
		"currency":          "USD",
		"softStopThreshold": application.DefaultSoftStopThresholdUSD,
		"fetchingEnabled":   true,
	}, record.EstimatedCostUSD)
	if err != nil {
		return err
	}
	status, err := usageStatusFromItem(scope, item)
	if err != nil {
		return err
	}
	if status.Budget.Stopped || !status.FetchingEnabled {
		return application.ErrBudgetExceeded
	}
	return nil
}

func (r *DynamoDBRepository) ReconcileAccountUsage(ctx context.Context, scope application.UsageBudgetScope, response flightaware.UsageResponse) error {
	if scope.Key() == "" {
		return application.ErrValidation
	}
	currency := response.Currency
	if currency == "" {
		currency = "USD"
	}
	_, _, err := r.client.PutUsageEstimateIfHigher(ctx, r.tables.UsageBudget, scope.Key(), map[string]any{
		"environment":       scope.Environment,
		"month":             scope.Month,
		"currency":          currency,
		"softStopThreshold": application.DefaultSoftStopThresholdUSD,
		"fetchingEnabled":   true,
	}, response.MonthToDate.EstimatedCostUSD)
	if err != nil {
		if errors.Is(err, application.ErrNotFound) {
			return nil
		}
		return err
	}
	return nil
}
