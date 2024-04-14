package errors

type CodedError interface {
	Code() string
}
type InvalidMoveError struct {
	ErrCode string
	Message string
}

func (e *InvalidMoveError) Error() string {
	return e.Message
}

func (e *InvalidMoveError) Code() string {
	return e.ErrCode
}

type UnparseableMoveError struct{}

func (e *UnparseableMoveError) Error() string {
	return "Could not parse move."
}

func (e *UnparseableMoveError) Code() string {
	return "UNPARSEABLE_MOVE"
}
