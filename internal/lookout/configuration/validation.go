package configuration

import "github.com/go-playground/validator/v10"

func (c LookoutConfig) Validate() error {
	validate := validator.New()
	return validate.Struct(c)
}
