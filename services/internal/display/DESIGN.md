# Display Helper Design

The `display` package owns backend presentation helpers for nullable domain values. It converts structured missing-value reasons into display-ready labels and formatted text while keeping the core domain package presentation-neutral.

- Domain code stores missing values as stable `domain.MissingValueReason` codes.
- `ToNullableDisplayValue` returns an available typed value when a pointer is present, or a missing state containing the reason and localized label when the pointer is nil.
- Text, date-time, progress, and airport helpers provide formatted strings for backend responses that intentionally need display text.
- Zero values are treated as real values when a pointer is present; only nil pointers produce a missing display state.
- Airport display preserves the airport code and appends the name only when a name is available.
- Localized labels currently cover not-acquired, not-announced, not-applicable, unavailable, and an unknown-reason fallback.
