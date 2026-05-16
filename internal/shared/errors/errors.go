// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
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

// AnnaveError is the structured error type used across all modules.
type AnnaveError struct {
	Code    string `json:"code"`
	Stage   Stage  `json:"stage"`
	Message string `json:"message"`
}

func (e *AnnaveError) Error() string {
	return fmt.Sprintf("[%s/%s] %s", e.Stage, e.Code, e.Message)
}

// New creates a new AnnaveError.
func New(code string, stage Stage, message string) *AnnaveError {
	return &AnnaveError{Code: code, Stage: stage, Message: message}
}

// Newf creates a new AnnaveError with a formatted message.
func Newf(code string, stage Stage, format string, args ...any) *AnnaveError {
	return &AnnaveError{Code: code, Stage: stage, Message: fmt.Sprintf(format, args...)}
}

// NotImplemented returns a standard "in development" error for stub adapters.
func NotImplemented(module string) *AnnaveError {
	return New(
		ErrCodeNotImplemented,
		StageRuntime,
		fmt.Sprintf("annave %s is in development and not yet available", module),
	)
}

// InvalidInput returns a standard invalid input error.
func InvalidInput(message string) *AnnaveError {
	return New(ErrCodeInvalidInput, StageInput, message)
}

// IOFailure returns a standard I/O error.
func IOFailure(message string) *AnnaveError {
	return New(ErrCodeIOFailure, StageInput, message)
}
