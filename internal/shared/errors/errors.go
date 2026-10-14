// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package errors

import "fmt"

// Stage represents which processing stage produced the error.
type Stage string

const (
	StageInput    Stage = "input"
	StageAnalysis Stage = "analysis"
	StageOutput   Stage = "output"
	StageRuntime  Stage = "runtime"
	StageNetwork  Stage = "network"
	StageIndex    Stage = "index"
)

// Common error codes used across all modules.
const (
	ErrCodeNotImplemented = "ERR_NOT_IMPLEMENTED"
	ErrCodeInvalidInput   = "ERR_INVALID_INPUT"
	ErrCodeIOFailure      = "ERR_IO_FAILURE"
	ErrCodeTimeout        = "ERR_TIMEOUT"
	ErrCodePermission     = "ERR_PERMISSION"
	ErrCodeNotFound       = "ERR_NOT_FOUND"
	ErrCodeParseFailed    = "ERR_PARSE_FAILED"
)

// Error is the structured error type used across all modules.
type Error struct {
	Code    string `json:"code"`
	Stage   Stage  `json:"stage"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("[%s/%s] %s", e.Stage, e.Code, e.Message)
}

// New creates a new Error.
func New(code string, stage Stage, message string) *Error {
	return &Error{Code: code, Stage: stage, Message: message}
}

// Newf creates a new Error with a formatted message.
func Newf(code string, stage Stage, format string, args ...any) *Error {
	return &Error{Code: code, Stage: stage, Message: fmt.Sprintf(format, args...)}
}

// NotImplemented returns a standard "in development" error for stub adapters.
func NotImplemented(module string) *Error {
	return New(
		ErrCodeNotImplemented,
		StageRuntime,
		fmt.Sprintf("tend %s is in development and not yet available", module),
	)
}

// InvalidInput returns a standard invalid input error.
func InvalidInput(message string) *Error {
	return New(ErrCodeInvalidInput, StageInput, message)
}

// IOFailure returns a standard I/O error.
func IOFailure(message string) *Error {
	return New(ErrCodeIOFailure, StageInput, message)
}
