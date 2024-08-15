/*
Given any type that implements the ErrorCustom interface, and ErrorCustomVariant
CustomError will automatically implement the necessary interfaces to simplify
the error handling process, while keeping enough information
*/
package errs

import (
	"errors"
	"reflect"
)

type ErrorCustomVariant interface {
	Error() string
	// Is(target ErrorCustomVariant) bool
}

type ErrorCustom interface {
	GetVariant() ErrorCustomVariant

	// --- Go classical errors compatibility
	Error() string
	Is(target error) bool
	// ---
}

type CustomError struct {
	variant ErrorCustomVariant
	cause   error
	message error
}

func NewCustomError(variant ErrorCustomVariant, cause, message string) CustomError {
	return CustomError{
		variant: variant,
		cause:   errors.New(cause),
		message: errors.New(message),
	}
}

func (ce CustomError) GetVariant() ErrorCustomVariant {
	return ce.variant
}

func (ce CustomError) Error() string {
	return errors.Join(ce.cause, ce.message).Error()
}

func (m CustomError) Is(target error) bool {
	val := reflect.TypeOf((*ErrorCustom)(nil)).Elem()

	if reflect.TypeOf(target).Implements(val) {
		targetCasted := target.(ErrorCustom)
		return targetCasted.GetVariant() == m.GetVariant()
	}
	if reflect.TypeOf(target) == reflect.TypeOf(m.variant) {
		return target.(ErrorCustomVariant) == m.GetVariant()
	}

	return false
}
