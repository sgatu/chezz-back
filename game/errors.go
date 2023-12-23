package game

type CodedError interface {
	Code() string
}
type InvalidMoveError struct {
	code    string
	message string
}

func (e *InvalidMoveError) Error() string {
	return e.message
}
func (e *InvalidMoveError) Code() string {
	return e.code
}

type UnparseableMoveError struct{}

func (e *UnparseableMoveError) Error() string {
	return "Could not parse move."
}
func (e *UnparseableMoveError) Code() string {
	return "UNPARSEABLE_MOVE"
}
