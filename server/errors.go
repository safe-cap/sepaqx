package server

type ErrorCode string

const (
	CodeInvalidJSON        ErrorCode = "invalid_json"
	CodeInvalidInput       ErrorCode = "invalid_input"
	CodeUnauthorized       ErrorCode = "unauthorized"
	CodeRateLimited        ErrorCode = "rate_limited"
	CodeMethodNotAllowed   ErrorCode = "method_not_allowed"
	CodePayloadBuildFailed ErrorCode = "payload_build_failed"
	CodeQREncodeFailed     ErrorCode = "qr_encode_failed"
)

func errorStatus(code ErrorCode) int {
	switch code {
	case CodeInvalidJSON, CodeInvalidInput:
		return 400
	case CodeUnauthorized:
		return 401
	case CodeRateLimited:
		return 429
	case CodeMethodNotAllowed:
		return 405
	case CodePayloadBuildFailed, CodeQREncodeFailed:
		return 500
	default:
		return 500
	}
}
