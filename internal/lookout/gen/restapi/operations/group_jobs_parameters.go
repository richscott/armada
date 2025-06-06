// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"io"
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/validate"
)

// NewGroupJobsParams creates a new GroupJobsParams object
//
// There are no default values defined in the spec.
func NewGroupJobsParams() GroupJobsParams {

	return GroupJobsParams{}
}

// GroupJobsParams contains all the bound params for the group jobs operation
// typically these are obtained from a http.Request
//
// swagger:parameters groupJobs
type GroupJobsParams struct {

	// HTTP Request Object
	HTTPRequest *http.Request `json:"-"`

	/*The backend to use for this request.
	  In: query
	*/
	Backend *string
	/*
	  Required: true
	  In: body
	*/
	GroupJobsRequest GroupJobsBody
}

// BindRequest both binds and validates a request, it assumes that complex things implement a Validatable(strfmt.Registry) error interface
// for simple values it will use straight method calls.
//
// To ensure default values, the struct must have been initialized with NewGroupJobsParams() beforehand.
func (o *GroupJobsParams) BindRequest(r *http.Request, route *middleware.MatchedRoute) error {
	var res []error

	o.HTTPRequest = r

	qs := runtime.Values(r.URL.Query())

	qBackend, qhkBackend, _ := qs.GetOK("backend")
	if err := o.bindBackend(qBackend, qhkBackend, route.Formats); err != nil {
		res = append(res, err)
	}

	if runtime.HasBody(r) {
		defer r.Body.Close()
		var body GroupJobsBody
		if err := route.Consumer.Consume(r.Body, &body); err != nil {
			if err == io.EOF {
				res = append(res, errors.Required("groupJobsRequest", "body", ""))
			} else {
				res = append(res, errors.NewParseError("groupJobsRequest", "body", "", err))
			}
		} else {
			// validate body object
			if err := body.Validate(route.Formats); err != nil {
				res = append(res, err)
			}

			ctx := validate.WithOperationRequest(r.Context())
			if err := body.ContextValidate(ctx, route.Formats); err != nil {
				res = append(res, err)
			}

			if len(res) == 0 {
				o.GroupJobsRequest = body
			}
		}
	} else {
		res = append(res, errors.Required("groupJobsRequest", "body", ""))
	}
	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

// bindBackend binds and validates parameter Backend from query.
func (o *GroupJobsParams) bindBackend(rawData []string, hasKey bool, formats strfmt.Registry) error {
	var raw string
	if len(rawData) > 0 {
		raw = rawData[len(rawData)-1]
	}

	// Required: false
	// AllowEmptyValue: false

	if raw == "" { // empty values pass all other validations
		return nil
	}
	o.Backend = &raw

	if err := o.validateBackend(formats); err != nil {
		return err
	}

	return nil
}

// validateBackend carries on validations for parameter Backend
func (o *GroupJobsParams) validateBackend(formats strfmt.Registry) error {

	if err := validate.EnumCase("backend", "query", *o.Backend, []interface{}{"jsonb"}, true); err != nil {
		return err
	}

	return nil
}
