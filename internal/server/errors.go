package server

import (
	"net/http"
	"strings"

	"github.com/joomcode/errorx"
)

// ErrorType represents different types of errors
type ErrorType int

const (
	ErrorTypeNotFound ErrorType = iota
	ErrorTypeInternal
	ErrorTypeBadRequest
)

// handleHTTPError logs an error and sends an HTTP error response
func (s *Server) handleHTTPError(w http.ResponseWriter, err error, message string, statusCode int) {
	s.Logger.Error(message, "error", err)
	if err != nil {
		http.Error(w, message+": "+err.Error(), statusCode)
	} else {
		http.Error(w, message, statusCode)
	}
}

// handleError determines the appropriate HTTP status code and handles the error
func (s *Server) handleError(w http.ResponseWriter, err error, defaultMessage string) {
	if err == nil {
		return
	}

	// Determine error type and status code
	statusCode := s.getStatusCodeFromError(err)
	message := defaultMessage

	// For not found errors, use empty message to show just the error
	if statusCode == http.StatusNotFound {
		message = ""
	}

	s.handleHTTPError(w, err, message, statusCode)
}

// getStatusCodeFromError determines the appropriate HTTP status code from an error
func (s *Server) getStatusCodeFromError(err error) int {
	if strings.Contains(err.Error(), "not found") {
		return http.StatusNotFound
	}

	if errorx.IsOfType(err, errorx.InternalError) {
		return http.StatusInternalServerError
	}

	// Default to internal server error
	return http.StatusInternalServerError
}
