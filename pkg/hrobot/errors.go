// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package hrobot

import (
	"encoding/json"
	"fmt"
)

// Error represents all possible errors from the hrobot library.
type Error struct {
	Kind    ErrorKind
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Kind, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Kind, e.Message)
}

func (e *Error) Unwrap() error {
	return e.Cause
}

// ErrorKind categorizes the error type.
type ErrorKind string

const (
	ErrKindAPI     ErrorKind = "API"
	ErrKindNetwork ErrorKind = "Network"
	ErrKindParse   ErrorKind = "Parse"
	ErrKindAuth    ErrorKind = "Auth"
)

// NewAPIError creates a new API error.
func NewAPIError(code ErrorCode, message string) *Error {
	return &Error{
		Kind:    ErrKindAPI,
		Message: fmt.Sprintf("[%s] %s", code, message),
	}
}

// NewNetworkError creates a new network error.
func NewNetworkError(message string, cause error) *Error {
	return &Error{
		Kind:    ErrKindNetwork,
		Message: message,
		Cause:   cause,
	}
}

// NewParseError creates a new parse error.
func NewParseError(message string, cause error) *Error {
	return &Error{
		Kind:    ErrKindParse,
		Message: message,
		Cause:   cause,
	}
}

// NewAuthError creates a new authentication error.
func NewAuthError(message string) *Error {
	return &Error{
		Kind:    ErrKindAuth,
		Message: message,
	}
}

// ErrorCode represents specific API error codes from Hetzner.
type ErrorCode string

const (
	// Common errors.
	ErrUnauthorized            ErrorCode = "UNAUTHORIZED"
	ErrInvalidInput            ErrorCode = "INVALID_INPUT"
	ErrInvalidInputServerIP    ErrorCode = "INVALID_INPUT_SERVER_IP"
	ErrInvalidInputIPAddress   ErrorCode = "INVALID_INPUT_IP_ADDRESS"
	ErrServerNotFound          ErrorCode = "SERVER_NOT_FOUND"
	ErrIPNotFound              ErrorCode = "IP_NOT_FOUND"
	ErrIPLocked                ErrorCode = "IP_LOCKED"
	ErrInsufficientPermissions ErrorCode = "INSUFFICIENT_PERMISSIONS"
	ErrRateLimitExceeded       ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrMaintenanceMode         ErrorCode = "MAINTENANCE_MODE"

	// Firewall errors.
	ErrFirewallInProcess         ErrorCode = "FIREWALL_IN_PROCESS"
	ErrFirewallAlreadyActive     ErrorCode = "FIREWALL_ALREADY_ACTIVE"
	ErrFirewallAlreadyDisabled   ErrorCode = "FIREWALL_ALREADY_DISABLED"
	ErrFirewallConfigInvalid     ErrorCode = "FIREWALL_CONFIG_INVALID"
	ErrFirewallRuleLimitExceeded ErrorCode = "FIREWALL_RULE_LIMIT_EXCEEDED"

	// Boot errors.
	ErrBootConfigNotFound  ErrorCode = "BOOT_CONFIG_NOT_FOUND"
	ErrBootAlreadyActive   ErrorCode = "BOOT_ALREADY_ACTIVE"
	ErrRescueNotActive     ErrorCode = "RESCUE_NOT_ACTIVE"
	ErrRescueAlreadyActive ErrorCode = "RESCUE_ALREADY_ACTIVE"

	// Reset errors.
	ErrResetNotAvailable ErrorCode = "RESET_NOT_AVAILABLE"
	ErrResetManualActive ErrorCode = "RESET_MANUAL_ACTIVE"

	// VNC errors.
	ErrVNCDisabled     ErrorCode = "VNC_DISABLED"
	ErrVNCNotAvailable ErrorCode = "VNC_NOT_AVAILABLE"

	// Reverse DNS errors.
	ErrReverseDNSNotFound ErrorCode = "RDNS_NOT_FOUND"
	ErrReverseDNSInvalid  ErrorCode = "RDNS_INVALID"

	// Unknown error.
	ErrUnknown ErrorCode = "UNKNOWN"
)

// APIErrorResponse represents the error response from Hetzner API.
type APIErrorResponse struct {
	Error APIErrorDetail `json:"error"`
}

// APIErrorDetail contains the error details.
type APIErrorDetail struct {
	Status  int       `json:"status"`
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

// UnmarshalJSON handles both known and unknown error codes.
func (d *APIErrorDetail) UnmarshalJSON(data []byte) error {
	type Alias APIErrorDetail
	aux := &struct {
		Code json.RawMessage `json:"code"`
		*Alias
	}{
		Alias: (*Alias)(d),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Try to unmarshal code as string
	var codeStr string
	if err := json.Unmarshal(aux.Code, &codeStr); err == nil {
		d.Code = ErrorCode(codeStr)
		return nil
	}

	// If not a string, set as unknown
	d.Code = ErrUnknown
	return nil
}

// IsAPIError checks if an error is an API error with a specific code.
func IsAPIError(err error, code ErrorCode) bool {
	if e, ok := err.(*Error); ok {
		if e.Kind != ErrKindAPI {
			return false
		}
		// Message is formatted as "[CODE] message"
		codeStr := fmt.Sprintf("[%s]", code)
		return len(e.Message) >= len(codeStr) && e.Message[:len(codeStr)] == codeStr
	}
	return false
}

// IsRateLimitError checks if the error is a rate limit error.
func IsRateLimitError(err error) bool {
	return IsAPIError(err, ErrRateLimitExceeded)
}

// IsNotFoundError checks if the error is a not found error.
func IsNotFoundError(err error) bool {
	return IsAPIError(err, ErrServerNotFound) || IsAPIError(err, ErrIPNotFound)
}

// IsFirewallInProcessError checks if the error is a firewall in process error.
func IsFirewallInProcessError(err error) bool {
	return IsAPIError(err, ErrFirewallInProcess)
}

// IsUnauthorizedError checks if the error is an unauthorized error.
func IsUnauthorizedError(err error) bool {
	return IsAPIError(err, ErrUnauthorized)
}

// IsFirewallRuleLimitExceededError checks if the error is a firewall rule limit exceeded error.
func IsFirewallRuleLimitExceededError(err error) bool {
	return IsAPIError(err, ErrFirewallRuleLimitExceeded)
}
