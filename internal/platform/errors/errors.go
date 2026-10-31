package errors

import (
	"fmt"
	"net/http"
)

// Standard Error Codes
const (
	CodeInvalidRequest           = "INVALID_REQUEST"
	CodeUnauthorized             = "UNAUTHORIZED"
	CodeForbidden                = "FORBIDDEN"
	CodeNotFound                 = "NOT_FOUND"
	CodeConflict                 = "CONFLICT"
	CodeUnprocessableEntity      = "UNPROCESSABLE_ENTITY"
	CodeIdempotencyConflict      = "IDEMPOTENCY_CONFLICT"
	CodeConnectorExternalBlocked = "CONNECTOR_EXTERNAL_BLOCKED"
	CodeInsufficientData         = "INSUFFICIENT_DATA"
	CodeTenantIsolationViolation = "TENANT_ISOLATION_VIOLATION"
	CodeInternalServerError      = "INTERNAL_SERVER_ERROR"
)

// ProblemDetails represents an RFC 9457 Problem Details object
type ProblemDetails struct {
	Type     string         `json:"type"`
	Title    string         `json:"title"`
	Status   int            `json:"status"`
	Detail   string         `json:"detail"`
	Instance string         `json:"instance,omitempty"`
	Code     string         `json:"code"`
	Errors   []FieldProblem `json:"errors,omitempty"`
}

// FieldProblem represents specific invalid field detail
type FieldProblem struct {
	Field   string `json:"field"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// AppError is the core application error with code and HTTP status
type AppError struct {
	Code       string
	MessageKo  string
	MessageEn  string
	HTTPStatus int
	Err        error
	FieldErrs  []FieldProblem
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.MessageKo, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.MessageKo)
}

func (e *AppError) ToProblemDetails(instance string) ProblemDetails {
	return ProblemDetails{
		Type:     fmt.Sprintf("https://api.merchant-os.bookingnetwork.internal/errors/%s", e.Code),
		Title:    e.MessageKo,
		Status:   e.HTTPStatus,
		Detail:   e.MessageKo,
		Instance: instance,
		Code:     e.Code,
		Errors:   e.FieldErrs,
	}
}

// Factory helper constructors
func NewBadRequest(messageKo string, fieldErrs ...FieldProblem) *AppError {
	return &AppError{
		Code:       CodeInvalidRequest,
		MessageKo:  messageKo,
		HTTPStatus: http.StatusBadRequest,
		FieldErrs:  fieldErrs,
	}
}

func NewUnauthorized(messageKo string) *AppError {
	return &AppError{
		Code:       CodeUnauthorized,
		MessageKo:  messageKo,
		HTTPStatus: http.StatusUnauthorized,
	}
}

func NewForbidden(messageKo string) *AppError {
	return &AppError{
		Code:       CodeForbidden,
		MessageKo:  messageKo,
		HTTPStatus: http.StatusForbidden,
	}
}

func NewNotFound(resource string) *AppError {
	return &AppError{
		Code:       CodeNotFound,
		MessageKo:  fmt.Sprintf("요청하신 %s을(를) 찾을 수 없습니다.", resource),
		HTTPStatus: http.StatusNotFound,
	}
}

func NewConflict(messageKo string) *AppError {
	return &AppError{
		Code:       CodeConflict,
		MessageKo:  messageKo,
		HTTPStatus: http.StatusConflict,
	}
}

func NewUnprocessable(messageKo string) *AppError {
	return &AppError{
		Code:       CodeUnprocessableEntity,
		MessageKo:  messageKo,
		HTTPStatus: http.StatusUnprocessableEntity,
	}
}

func NewBlockedConnector(providerName string, fallbackGuide string) *AppError {
	return &AppError{
		Code:       CodeConnectorExternalBlocked,
		MessageKo:  fmt.Sprintf("[%s] 연동은 정식 제휴/인허가 준비 중입니다. %s", providerName, fallbackGuide),
		HTTPStatus: http.StatusUnprocessableEntity,
	}
}

func NewInternal(err error) *AppError {
	return &AppError{
		Code:       CodeInternalServerError,
		MessageKo:  "서버 내부 처리 중 일시적인 오류가 발생했습니다. 잠시 후 다시 시도해 주세요.",
		HTTPStatus: http.StatusInternalServerError,
		Err:        err,
	}
}

