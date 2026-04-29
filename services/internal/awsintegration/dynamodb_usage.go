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
		"budgetScope": scope,
		"body":        mustJSON(status),
	})
}

func (r *DynamoDBRepository) PutMonthlyUsageStatus(ctx context.Context, scope application.UsageBudgetScope, status application.UsageStatus) error {
	status.Budget.Environment = scope.Environment
	status.Budget.Month = scope.Month
	return r.PutUsageStatus(ctx, scope.Key(), status)
}

func (r *DynamoDBRepository) GetUsageStatus(ctx context.Context) (application.UsageStatus, error) {
	if r.usageScope.Key() != "" {
		return r.GetMonthlyUsageStatus(ctx, r.usageScope)
	}
	items, err := r.client.QueryByPrefix(ctx, r.tables.UsageBudget, "budgetScope", "")
	if err != nil {
		return application.UsageStatus{}, err
	}
	if len(items) == 0 {
		return application.UsageStatus{}, application.ErrNotFound
	}
	var status application.UsageStatus
	if err := json.Unmarshal([]byte(stringValue(items[len(items)-1]["body"])), &status); err != nil {
		return application.UsageStatus{}, err
	}
	return application.NormalizeUsageStatus(status), nil
}

func (r *DynamoDBRepository) GetMonthlyUsageStatus(ctx context.Context, scope application.UsageBudgetScope) (application.UsageStatus, error) {
	item, ok, err := r.client.GetItem(ctx, r.tables.UsageBudget, "budgetScope", scope.Key())
	if err != nil {
		return application.UsageStatus{}, err
	}
	if !ok {
		return application.UsageStatus{}, application.ErrNotFound
	}
	var status application.UsageStatus
	if err := json.Unmarshal([]byte(stringValue(item["body"])), &status); err != nil {
		return application.UsageStatus{}, err
	}
	status.Budget.Environment = scope.Environment
	status.Budget.Month = scope.Month
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
	status, err := r.GetMonthlyUsageStatus(ctx, scope)
	if err != nil {
		if !errors.Is(err, application.ErrNotFound) {
			return err
		}
		status = application.UsageStatus{
			Budget: application.UsageBudgetStatus{
				Environment:       scope.Environment,
				Month:             scope.Month,
				Currency:          "USD",
				SoftStopThreshold: application.DefaultSoftStopThresholdUSD,
			},
			FetchingEnabled: true,
		}
	}
	status.Budget.EstimatedMonthToDateCost += record.EstimatedCostUSD
	return r.PutMonthlyUsageStatus(ctx, scope, status)
}

func (r *DynamoDBRepository) ReconcileAccountUsage(ctx context.Context, scope application.UsageBudgetScope, response flightaware.UsageResponse) error {
	if scope.Key() == "" {
		return application.ErrValidation
	}
	status, err := r.GetMonthlyUsageStatus(ctx, scope)
	if err != nil {
		if !errors.Is(err, application.ErrNotFound) {
			return err
		}
		status = application.UsageStatus{
			Budget: application.UsageBudgetStatus{
				Environment:       scope.Environment,
				Month:             scope.Month,
				Currency:          response.Currency,
				SoftStopThreshold: application.DefaultSoftStopThresholdUSD,
			},
			FetchingEnabled: true,
		}
	}
	if response.Currency != "" {
		status.Budget.Currency = response.Currency
	}
	if response.MonthToDate.EstimatedCostUSD > status.Budget.EstimatedMonthToDateCost {
		status.Budget.EstimatedMonthToDateCost = response.MonthToDate.EstimatedCostUSD
	}
	return r.PutMonthlyUsageStatus(ctx, scope, status)
}
