package errx

import (
	"fmt"

	"github.com/cloudwego/kitex/pkg/kerrors"

	"github.com/arklib/ark/debug"
)

const DefaultErrCode = 500
const DefaultErrMessage = "Internal Server Error"
const defaultStackSkip = 3

type AppError struct {
	fatal   bool
	code    int
	message string
	stack   string
	basic   error
}

// New vals: string | int | error
func New(vals ...any) *AppError {
	err := &AppError{code: DefaultErrCode, message: DefaultErrMessage}

	for _, val := range vals {
		switch v := val.(type) {
		case int:
			err.code = v
		case string:
			err.message = v
		case *kerrors.DetailedError:
			err.message = v.Unwrap().Error()
			err.basic = v
		case error:
			if len(vals) == 1 {
				err.message = v.Error()
			}
			err.basic = v
		}
	}
	return err
}

func (e *AppError) Fatal() *AppError {
	e.fatal = true
	e.stack = string(debug.Stack(defaultStackSkip))
	return e
}

func (e *AppError) IsFatal() bool {
	return e.fatal
}

func (e *AppError) Code() int {
	return e.code
}

func (e *AppError) WithCode(code int) *AppError {
	e.code = code
	return e
}

func (e *AppError) Error() string {
	return e.message
}

func (e *AppError) FullError() string {
	message := e.message
	if e.basic != nil {
		message = fmt.Sprintf("%s (%s)", message, e.basic.Error())
	}
	if e.stack != "" {
		message = fmt.Sprintf("%s\n%s", message, e.stack)
	}
	return message
}

func (e *AppError) IsBasic() bool {
	return e.basic != nil
}

func (e *AppError) Stack() string {
	return e.stack
}

func (e *AppError) UnWrap() error {
	return e.basic
}

func Wrap(err any) *AppError {
	if IsAppError(err) {
		return err.(*AppError)
	}
	return New(err)
}

func Sprintf(format string, v ...any) *AppError {
	return New(fmt.Sprintf(format, v...))
}

// ----------------------------------------------------------------
// Assert
// ----------------------------------------------------------------

func NewX(vals ...any) *AppError {
	return New(vals...).Fatal()
}

func Assert(e error, vals ...any) {
	if e == nil {
		return
	}
	err := New(append(vals, e)...)
	panic(err)
}

func AssertX(e error, vals ...any) {
	if e == nil {
		return
	}
	err := New(append(vals, e)...).Fatal()
	panic(err)
}

func Throw(vals ...any) {
	err := New(vals...)
	panic(err)
}

func ThrowX(vals ...any) {
	err := New(vals...).Fatal()
	panic(err)
}

func IsAppError(err any) bool {
	switch err.(type) {
	case *AppError:
		return true
	default:
		return false
	}
}
