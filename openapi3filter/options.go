package openapi3filter

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
)

// Options used by ValidateRequest and ValidateResponse
type Options struct {
	// Set ExcludeRequestBody so ValidateRequest skips request body validation
	ExcludeRequestBody bool

	// Set ExcludeRequestQueryParams so ValidateRequest skips request query params validation
	ExcludeRequestQueryParams bool

	// Set ExcludeResponseBody so ValidateResponse skips response body validation
	ExcludeResponseBody bool

	// Set ExcludeReadOnlyValidations so ValidateRequest skips read-only validations
	ExcludeReadOnlyValidations bool

	// Set ExcludeWriteOnlyValidations so ValidateResponse skips write-only validations
	ExcludeWriteOnlyValidations bool

	// Set IncludeResponseStatus so ValidateResponse fails on response
	// status not defined in OpenAPI spec
	IncludeResponseStatus bool

	MultiError bool

	// Set RegexCompiler to override the regex implementation
	RegexCompiler openapi3.RegexCompilerFunc

	// Set RejectWhenRequestBodyNotSpecified so ValidateRequest fails when request body is present but not defined in the specification
	RejectWhenRequestBodyNotSpecified bool

	// A document with security schemes defined will not pass validation
	// unless an AuthenticationFunc is defined.
	// See NoopAuthenticationFunc
	AuthenticationFunc AuthenticationFunc

	// Indicates whether default values are set in the
	// request. If true, then they are not set
	SkipSettingDefaults bool

	// Set KeepValidatedValue so ValidateRequest gives req.Body a ValidatedValue() any
	// method returning the validated request body value, defaults included
	KeepValidatedValue bool

	customSchemaErrorFunc CustomSchemaErrorFunc

	// Additional schema validation options to pass through to schema validation.
	// The route document's own options (e.g. its registered format validators) are applied
	// automatically, before the options given here: a With*FormatValidators map replaces the
	// document's validators for that value type, a singular With*FormatValidator adds to them.
	SchemaValidationOptions []openapi3.SchemaValidationOption
}

// schemaValidationOptions returns the schema validation options to apply for a route:
// the document's own options (e.g. its registered format validators), followed by any
// caller-provided options so the caller's take precedence.
func (o *Options) schemaValidationOptions(route *routers.Route) []openapi3.SchemaValidationOption {
	if route == nil || route.Spec == nil {
		return o.SchemaValidationOptions
	}
	return append(route.Spec.GetSchemaValidationOptions(), o.SchemaValidationOptions...)
}

// CustomSchemaErrorFunc allows for custom the schema error message.
type CustomSchemaErrorFunc func(err *openapi3.SchemaError) string

// WithCustomSchemaErrorFunc sets a function to override the schema error message.
// If the passed function returns an empty string, it returns to the previous Error() implementation.
func (o *Options) WithCustomSchemaErrorFunc(f CustomSchemaErrorFunc) {
	o.customSchemaErrorFunc = f
}
