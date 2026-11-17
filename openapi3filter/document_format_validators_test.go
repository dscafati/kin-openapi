package openapi3filter

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

const documentFormatValidatorsSpec = `
openapi: 3.0.0
info:
  title: document format validators
  version: "1"
paths:
  /x:
    post:
      operationId: post
      parameters:
        - in: query
          name: p
          required: true
          schema: {type: string, format: uri}
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [u]
              properties:
                u: {type: string, format: uri}
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                required: [u]
                properties:
                  u: {type: string, format: uri}
`

// validURI passes document format validation; anything else is rejected.
const validURI = "https://ok"

func rejectUnlessValidURI(value string) error {
	if value == validURI {
		return nil
	}
	return errors.New("rejected")
}

func newDocumentFormatValidatorsRouter(t *testing.T, validator openapi3.StringFormatValidator) routers.Router {
	t.Helper()

	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData([]byte(documentFormatValidatorsSpec))
	require.NoError(t, err)
	require.NoError(t, doc.Validate(loader.Context))

	doc.SetStringFormatValidator("uri", validator)

	router, err := gorillamux.NewRouter(doc)
	require.NoError(t, err)
	return router
}

func newPostXRequest(t *testing.T) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"u":"anything"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func newValidPostXRequest(t *testing.T) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"u":"`+validURI+`"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestDocumentFormatValidatorsApplyToRequestBodyByDefault(t *testing.T) {
	t.Parallel()

	router := newDocumentFormatValidatorsRouter(t, openapi3.NewCallbackValidator(rejectUnlessValidURI))

	req := newPostXRequest(t)
	req.URL.RawQuery = "p=" + validURI

	route, pathParams, err := router.FindRoute(req)
	require.NoError(t, err)

	// No SchemaValidationOptions: the document's registered format validator must still apply.
	err = ValidateRequest(t.Context(), &RequestValidationInput{
		Request:    req,
		PathParams: pathParams,
		Route:      route,
		Options:    &Options{},
	})
	require.ErrorContains(t, err, `string doesn't match the format "uri": rejected`)
}

func TestDocumentFormatValidatorsApplyToParametersByDefault(t *testing.T) {
	t.Parallel()

	router := newDocumentFormatValidatorsRouter(t, openapi3.NewCallbackValidator(rejectUnlessValidURI))

	req := newValidPostXRequest(t)
	req.URL.RawQuery = "p=anything"

	route, pathParams, err := router.FindRoute(req)
	require.NoError(t, err)

	// The body passes; only the query parameter violates the document's format validator.
	err = ValidateRequest(t.Context(), &RequestValidationInput{
		Request:    req,
		PathParams: pathParams,
		Route:      route,
		Options:    &Options{},
	})
	require.ErrorContains(t, err, `string doesn't match the format "uri": rejected`)

	req = newValidPostXRequest(t)
	req.URL.RawQuery = "p=" + validURI

	route, pathParams, err = router.FindRoute(req)
	require.NoError(t, err)

	err = ValidateRequest(t.Context(), &RequestValidationInput{
		Request:    req,
		PathParams: pathParams,
		Route:      route,
		Options:    &Options{},
	})
	require.NoError(t, err)
}

func TestDocumentFormatValidatorsApplyToResponseByDefault(t *testing.T) {
	t.Parallel()

	router := newDocumentFormatValidatorsRouter(t, openapi3.NewCallbackValidator(rejectUnlessValidURI))

	req := newPostXRequest(t)

	route, pathParams, err := router.FindRoute(req)
	require.NoError(t, err)

	err = ValidateResponse(t.Context(), &ResponseValidationInput{
		RequestValidationInput: &RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
			Options:    &Options{},
		},
		Status: http.StatusOK,
		Header: http.Header{"Content-Type": []string{"application/json"}},
		Body:   io.NopCloser(strings.NewReader(`{"u":"anything"}`)),
	})
	require.ErrorContains(t, err, `string doesn't match the format "uri": rejected`)
}

func TestDocumentFormatValidatorsCallerOptionsTakePrecedence(t *testing.T) {
	t.Parallel()

	router := newDocumentFormatValidatorsRouter(t, openapi3.NewCallbackValidator(rejectUnlessValidURI))

	req := newValidPostXRequest(t)
	req.URL.RawQuery = "p=" + validURI

	route, pathParams, err := router.FindRoute(req)
	require.NoError(t, err)

	err = ValidateRequest(t.Context(), &RequestValidationInput{
		Request:    req,
		PathParams: pathParams,
		Route:      route,
		Options: &Options{
			SchemaValidationOptions: []openapi3.SchemaValidationOption{
				openapi3.WithStringFormatValidators(map[string]openapi3.StringFormatValidator{
					"uri": openapi3.NewCallbackValidator(func(string) error {
						return errors.New("explicit")
					}),
				}),
			},
		},
	})
	require.ErrorContains(t, err, `string doesn't match the format "uri": explicit`)
}

func TestDocumentFormatValidatorsCallerOptionsDoNotMutateDocument(t *testing.T) {
	t.Parallel()

	router := newDocumentFormatValidatorsRouter(t, openapi3.NewCallbackValidator(rejectUnlessValidURI))

	// A singular caller option for the same format must not write into the document's map.
	req := newValidPostXRequest(t)
	req.URL.RawQuery = "p=" + validURI

	route, pathParams, err := router.FindRoute(req)
	require.NoError(t, err)

	err = ValidateRequest(t.Context(), &RequestValidationInput{
		Request:    req,
		PathParams: pathParams,
		Route:      route,
		Options: &Options{
			SchemaValidationOptions: []openapi3.SchemaValidationOption{
				openapi3.WithStringFormatValidator("uri", openapi3.NewCallbackValidator(func(string) error {
					return errors.New("override")
				})),
			},
		},
	})
	require.ErrorContains(t, err, `string doesn't match the format "uri": override`)

	// A later validation with default options must still see the document's own validator,
	// unmodified by the previous call.
	req = newValidPostXRequest(t)
	req.URL.RawQuery = "p=anything"

	route, pathParams, err = router.FindRoute(req)
	require.NoError(t, err)

	err = ValidateRequest(t.Context(), &RequestValidationInput{
		Request:    req,
		PathParams: pathParams,
		Route:      route,
		Options:    &Options{},
	})
	require.ErrorContains(t, err, `string doesn't match the format "uri": rejected`)
}
