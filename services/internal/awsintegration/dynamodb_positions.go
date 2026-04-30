package awsintegration

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"

	"airpath/services/internal/application"
	"airpath/services/internal/domain"
)

const (
	maxDynamoDBTransactionExtraItems      = 99
	positionWriteTokenAttribute           = "positionWriteToken"
	committedPositionWriteTokenAttribute  = "committedPositionWriteToken"
	committedPositionWriteTokensAttribute = "committedPositionWriteTokens"
)

func (r *DynamoDBRepository) PutPosition(ctx context.Context, position domain.FlightPosition) error {
	return r.client.PutItem(ctx, r.tables.FlightPositions, positionItem(position))
}

func (r *DynamoDBRepository) AppendPosition(ctx context.Context, position domain.FlightPosition) error {
	return r.PutPosition(ctx, position)
}

func (r *DynamoDBRepository) AppendPositionIfLeaseHeld(ctx context.Context, position domain.FlightPosition, owner string, leaseUntil string) (bool, error) {
	return r.client.PutItemIfOtherItemLeaseOwner(ctx, r.tables.FlightPositions, positionItem(position), r.tables.Flights, map[string]any{
		"flightId": position.FlightID,
	}, owner, leaseUntil, r.nowISO())
}

func (r *DynamoDBRepository) UpdateFetchedFlightWithPosition(ctx context.Context, flight domain.Flight, position domain.FlightPosition, owner string, leaseUntil string, now string) (bool, error) {
	flight.UpdatedAt = now
	item, err := r.flightItemPreservingCommittedPositionWriteToken(ctx, flight)
	if err != nil {
		return false, err
	}
	return r.client.PutItemsIfLeaseOwner(ctx, r.tables.Flights, item, r.tables.FlightPositions, positionItem(position), owner, leaseUntil, now)
}

func (r *DynamoDBRepository) UpdateFetchedFlightWithTrackPositions(ctx context.Context, flight domain.Flight, positions []domain.FlightPosition, owner string, leaseUntil string, now string) (bool, error) {
	flight.UpdatedAt = now
	positionItems := uniquePositionItems(positions)
	if len(positionItems) <= maxDynamoDBTransactionExtraItems {
		item, err := r.flightItemPreservingCommittedPositionWriteToken(ctx, flight)
		if err != nil {
			return false, err
		}
		return r.client.PutItemCollectionIfLeaseOwner(ctx, r.tables.Flights, item, r.tables.FlightPositions, positionItems, owner, leaseUntil, now)
	}

	// DynamoDB transactions can write at most 100 items. Larger tracks are staged
	// with a token on each position, then made visible only after the flight row
	// commits the same token under the lease condition.
	writeToken, err := newPositionWriteToken(owner, leaseUntil)
	if err != nil {
		return false, err
	}
	written := make([]largeTrackPositionWrite, 0, len(positionItems))
	conditionKey := map[string]any{"flightId": flight.FlightID}
	for _, positionItem := range positionItems {
		positionItem[positionWriteTokenAttribute] = writeToken
		previous, existed, err := r.getPositionItem(ctx, positionItem)
		if err != nil {
			r.rollbackPositionItemsBestEffort(ctx, written)
			return false, err
		}
		updated, err := r.client.PutItemIfOtherItemLeaseOwner(ctx, r.tables.FlightPositions, positionItem, r.tables.Flights, conditionKey, owner, leaseUntil, now)
		if err != nil || !updated {
			r.rollbackPositionItemsBestEffort(ctx, written)
			return updated, err
		}
		written = append(written, largeTrackPositionWrite{item: positionItem, previous: previous, existed: existed})
	}

	updated, err := r.putFlightWithCommittedPositionWriteTokenIfLeaseOwner(ctx, flight, owner, leaseUntil, now, writeToken)
	if err != nil || !updated {
		r.rollbackPositionItemsBestEffort(ctx, written)
		return updated, err
	}
	cleanupComplete, err := r.clearPositionWriteTokensAfterCommit(ctx, written)
	if err != nil {
		return false, err
	}
	if cleanupComplete {
		tokenCleanupComplete, err := r.clearCommittedPositionWriteTokenIfLeaseOwner(ctx, flight, owner, leaseUntil, now, writeToken)
		if err != nil {
			return false, err
		}
		if !tokenCleanupComplete {
			return false, nil
		}
	}
	return updated, nil
}

func (r *DynamoDBRepository) putFlightWithCommittedPositionWriteTokenIfLeaseOwner(ctx context.Context, flight domain.Flight, owner string, leaseUntil string, now string, writeToken string) (bool, error) {
	item := flightItem(flight)
	tokens, err := r.committedPositionWriteTokens(ctx, flight.FlightID)
	if err != nil {
		return false, err
	}
	tokens = appendUniqueString(tokens, writeToken)
	applyCommittedPositionWriteTokens(item, tokens)
	return r.client.PutItemIfLeaseOwner(ctx, r.tables.Flights, item, owner, leaseUntil, now)
}

func (r *DynamoDBRepository) clearPositionWriteTokensAfterCommit(ctx context.Context, written []largeTrackPositionWrite) (bool, error) {
	cleanupComplete := true
	for _, write := range written {
		item := cloneItem(write.item)
		delete(item, positionWriteTokenAttribute)
		// Cleanup is conditional on the staged body and token. If another worker has
		// already replaced the row, leave that newer row untouched and let the flight
		// token keep any remaining staged rows visible.
		updated, err := r.client.PutItemIfCurrentAttributesEqual(ctx, r.tables.FlightPositions, item, rollbackExpectedAttributes(write.item))
		if err != nil {
			return false, err
		}
		if !updated {
			cleanupComplete = false
		}
	}
	return cleanupComplete, nil
}

func (r *DynamoDBRepository) clearCommittedPositionWriteTokenIfLeaseOwner(ctx context.Context, flight domain.Flight, owner string, leaseUntil string, now string, cleanedToken string) (bool, error) {
	item := flightItem(flight)
	tokens, err := r.committedPositionWriteTokens(ctx, flight.FlightID)
	if err != nil {
		return false, err
	}
	tokens, err = r.remainingCommittedPositionWriteTokens(ctx, flight.FlightID, removeString(tokens, cleanedToken))
	if err != nil {
		return false, err
	}
	applyCommittedPositionWriteTokens(item, tokens)
	return r.client.PutItemIfLeaseOwner(ctx, r.tables.Flights, item, owner, leaseUntil, now)
}

func (r *DynamoDBRepository) remainingCommittedPositionWriteTokens(ctx context.Context, flightID domain.FlightID, tokens []string) ([]string, error) {
	if len(tokens) == 0 {
		return nil, nil
	}
	tokenSet := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		tokenSet[token] = struct{}{}
	}
	items, err := r.client.QueryRange(ctx, r.tables.FlightPositions, "", "flightId", string(flightID), "timestamp", nil, 0, false)
	if err != nil {
		return nil, err
	}
	remaining := []string{}
	for _, item := range items {
		token := stringValue(item[positionWriteTokenAttribute])
		if _, ok := tokenSet[token]; ok {
			remaining = appendUniqueString(remaining, token)
		}
	}
	return remaining, nil
}

type largeTrackPositionWrite struct {
	item     map[string]any
	previous map[string]any
	existed  bool
}

func (r *DynamoDBRepository) getPositionItem(ctx context.Context, item map[string]any) (map[string]any, bool, error) {
	timestamp := stringValue(item["timestamp"])
	if timestamp == "" {
		return nil, false, application.ErrValidation
	}
	items, err := r.client.QueryRange(ctx, r.tables.FlightPositions, "", "flightId", stringValue(item["flightId"]), "timestamp", &timestamp, 1, false)
	if err != nil {
		return nil, false, err
	}
	if len(items) == 0 || stringValue(items[0]["timestamp"]) != timestamp {
		return nil, false, nil
	}
	return items[0], true, nil
}

func uniquePositionItems(positions []domain.FlightPosition) []map[string]any {
	positionItems := make([]map[string]any, 0, len(positions))
	seen := map[string]int{}
	for _, position := range positions {
		item := positionItem(position)
		key := itemKey(item)
		if index, ok := seen[key]; ok {
			positionItems[index] = item
			continue
		}
		seen[key] = len(positionItems)
		positionItems = append(positionItems, item)
	}
	return positionItems
}

func (r *DynamoDBRepository) GetLatestPosition(ctx context.Context, flightID domain.FlightID) (*domain.FlightPosition, application.CacheMetadata, error) {
	items, err := r.client.QueryRange(ctx, r.tables.FlightPositions, "", "flightId", string(flightID), "timestamp", nil, 0, true)
	if err != nil {
		return nil, application.CacheMetadata{}, err
	}
	items, err = r.visiblePositionItems(ctx, flightID, items, 1)
	if err != nil {
		return nil, application.CacheMetadata{}, err
	}
	if len(items) == 0 {
		return nil, application.CacheMetadata{Freshness: application.CacheFreshnessMiss, Source: application.CacheSourceCache}, nil
	}
	position, err := positionFromItem(items[0])
	if err != nil {
		return nil, application.CacheMetadata{}, err
	}
	return &position, cacheFor(true), nil
}

func positionItem(position domain.FlightPosition) map[string]any {
	item := map[string]any{
		"flightId":  position.FlightID,
		"timestamp": position.Timestamp,
		"body":      mustJSON(position),
	}
	if position.TTL != nil {
		item["ttl"] = *position.TTL
	}
	return item
}

func (r *DynamoDBRepository) rollbackPositionItemsBestEffort(ctx context.Context, items []largeTrackPositionWrite) {
	for index := len(items) - 1; index >= 0; index-- {
		item := items[index]
		expected := rollbackExpectedAttributes(item.item)
		if item.existed {
			_, _ = r.client.PutItemIfCurrentAttributesEqual(ctx, r.tables.FlightPositions, item.previous, expected)
			continue
		}
		_, _ = r.client.DeleteItemIfCurrentAttributesEqual(ctx, r.tables.FlightPositions, positionItemKey(item.item), expected)
	}
}

func rollbackExpectedAttributes(item map[string]any) map[string]any {
	expected := map[string]any{"body": item["body"]}
	if token := stringValue(item[positionWriteTokenAttribute]); token != "" {
		expected[positionWriteTokenAttribute] = token
	}
	return expected
}

func positionItemKey(item map[string]any) map[string]any {
	return map[string]any{
		"flightId":  item["flightId"],
		"timestamp": item["timestamp"],
	}
}

func newPositionWriteToken(owner string, leaseUntil string) (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return owner + "#" + leaseUntil + "#" + hex.EncodeToString(bytes), nil
}

func (r *DynamoDBRepository) ListPositions(ctx context.Context, flightID domain.FlightID, since *string, limit int) ([]domain.FlightPosition, application.CacheMetadata, error) {
	items, err := r.client.QueryRange(ctx, r.tables.FlightPositions, "", "flightId", string(flightID), "timestamp", since, 0, false)
	if err != nil {
		return nil, application.CacheMetadata{}, err
	}
	items, err = r.visiblePositionItems(ctx, flightID, items, limit)
	if err != nil {
		return nil, application.CacheMetadata{}, err
	}
	positions := make([]domain.FlightPosition, 0, len(items))
	for _, item := range items {
		position, err := positionFromItem(item)
		if err != nil {
			return nil, application.CacheMetadata{}, err
		}
		positions = append(positions, position)
	}
	return positions, cacheFor(len(positions) > 0), nil
}

func (r *DynamoDBRepository) visiblePositionItems(ctx context.Context, flightID domain.FlightID, items []map[string]any, limit int) ([]map[string]any, error) {
	visible := make([]map[string]any, 0, len(items))
	committedTokens := map[string]struct{}{}
	loadedCommittedToken := false
	now := r.now()
	for _, item := range items {
		if itemTTLExpired(item, now) {
			continue
		}
		writeToken := stringValue(item[positionWriteTokenAttribute])
		if writeToken != "" {
			// Tokened rows belong to a large-track staged write. They become readable
			// only after the matching token is committed on the flight row.
			if !loadedCommittedToken {
				tokens, err := r.committedPositionWriteTokens(ctx, flightID)
				if err != nil {
					return nil, err
				}
				for _, token := range tokens {
					committedTokens[token] = struct{}{}
				}
				loadedCommittedToken = true
			}
			if _, ok := committedTokens[writeToken]; !ok {
				continue
			}
		}
		visible = append(visible, item)
		if limit > 0 && len(visible) >= limit {
			break
		}
	}
	return visible, nil
}

func (r *DynamoDBRepository) committedPositionWriteTokens(ctx context.Context, flightID domain.FlightID) ([]string, error) {
	if r.tables.Flights == "" {
		return nil, nil
	}
	item, ok, err := r.client.GetItem(ctx, r.tables.Flights, "flightId", string(flightID))
	if err != nil || !ok {
		return nil, err
	}
	return committedPositionWriteTokensFromItem(item)
}

func committedPositionWriteTokensFromItem(item map[string]any) ([]string, error) {
	tokens := []string{}
	if token := stringValue(item[committedPositionWriteTokenAttribute]); token != "" {
		tokens = appendUniqueString(tokens, token)
	}
	if encodedTokens := stringValue(item[committedPositionWriteTokensAttribute]); encodedTokens != "" {
		var parsed []string
		if err := json.Unmarshal([]byte(encodedTokens), &parsed); err != nil {
			return nil, err
		}
		for _, token := range parsed {
			tokens = appendUniqueString(tokens, token)
		}
	}
	return tokens, nil
}

func applyCommittedPositionWriteTokens(item map[string]any, tokens []string) {
	if len(tokens) == 0 {
		return
	}
	item[committedPositionWriteTokenAttribute] = tokens[len(tokens)-1]
	item[committedPositionWriteTokensAttribute] = mustJSON(tokens)
}

func appendUniqueString(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func removeString(values []string, value string) []string {
	filtered := values[:0]
	for _, existing := range values {
		if existing != value {
			filtered = append(filtered, existing)
		}
	}
	return filtered
}

func positionFromItem(item map[string]any) (domain.FlightPosition, error) {
	var position domain.FlightPosition
	if err := json.Unmarshal([]byte(stringValue(item["body"])), &position); err != nil {
		return domain.FlightPosition{}, err
	}
	return position, nil
}
