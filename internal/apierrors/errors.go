package apierrors

import "net/http"

const (
	CodeModelMissing  = "MODEL_MISSING"
	CodeModelLoadFail = "MODEL_LOAD_FAILED"
	CodeInvalidLang   = "INVALID_LANGUAGE"
	CodeSynthTimeout  = "SYNTH_TIMEOUT"
	CodeBadRequest    = "BAD_REQUEST"
	CodeInternalError = "INTERNAL_ERROR"
	CodeUnauthorized  = "UNAUTHORIZED"
	CodeConflict      = "CONFLICT"
)

type APIError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
	Status    int    `json:"-"`
}

func (e APIError) Error() string {
	return e.Code + ": " + e.Message
}

func BadRequest(msg string) APIError {
	return APIError{Code: CodeBadRequest, Message: msg, Retryable: false, Status: http.StatusBadRequest}
}

func Unauthorized(msg string) APIError {
	return APIError{Code: CodeUnauthorized, Message: msg, Retryable: false, Status: http.StatusUnauthorized}
}

func InvalidLanguage(msg string) APIError {
	return APIError{Code: CodeInvalidLang, Message: msg, Retryable: false, Status: http.StatusBadRequest}
}

func ModelMissing(msg string) APIError {
	return APIError{Code: CodeModelMissing, Message: msg, Retryable: false, Status: http.StatusNotFound}
}

func ModelLoadFailed(msg string) APIError {
	return APIError{Code: CodeModelLoadFail, Message: msg, Retryable: true, Status: http.StatusInternalServerError}
}

func SynthTimeout(msg string) APIError {
	return APIError{Code: CodeSynthTimeout, Message: msg, Retryable: true, Status: http.StatusGatewayTimeout}
}

func Conflict(msg string) APIError {
	return APIError{Code: CodeConflict, Message: msg, Retryable: false, Status: http.StatusConflict}
}

func Internal(msg string) APIError {
	return APIError{Code: CodeInternalError, Message: msg, Retryable: true, Status: http.StatusInternalServerError}
}
