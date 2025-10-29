// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package hrobot

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// encodeForm converts a map to url.Values.
func encodeForm(data map[string]string) url.Values {
	values := url.Values{}
	for k, v := range data {
		values.Set(k, v)
	}
	return values
}

// waitForCondition polls a condition function with exponential backoff until it returns true or context times out.
func waitForCondition(ctx context.Context, condition func() (bool, error)) error {
	const (
		minDelay   = 2 * time.Second
		maxDelay   = 30 * time.Second
		maxRetries = 30 // max ~15 minutes with exponential backoff
	)

	delay := minDelay
	for i := 0; i < maxRetries; i++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting cancelled: %w", ctx.Err())
		default:
		}

		// Check condition
		ready, err := condition()
		if err != nil {
			return fmt.Errorf("error checking condition: %w", err)
		}
		if ready {
			return nil
		}

		// Wait before retry
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("waiting cancelled: %w", ctx.Err())
		case <-timer.C:
		}

		// Exponential backoff with max delay
		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}

	return fmt.Errorf("timeout waiting for condition after %d retries", maxRetries)
}
