package handlers_messages

type UnknownSessionError struct {
	Message string `json:"message"`
	Code    int
}

func NewUnknownSessionError() UnknownSessionError {
	return UnknownSessionError{
		Message: "Unknown session",
		Code:    401,
	}
}
