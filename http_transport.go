// Copyright 2018, OpenCensus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// https://github.com/census-instrumentation/opencensus-go/blob/master/plugin/ochttp/client.go
// https://github.com/census-instrumentation/opencensus-go/blob/master/plugin/ochttp/client_stats.go

package opencensus

import (
	"context"
	"io"
	"net/http"
	"net/http/httptrace"
	"strconv"
	"sync"
	"time"

	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/plugin/ochttp/propagation/b3"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"
	"go.opencensus.io/trace/propagation"
)

var defaultFormat propagation.HTTPFormat = &b3.HTTPFormat{}

// Transport is an http.RoundTripper that instruments all outgoing requests with
// OpenCensus stats and tracing.
//
// The zero value is intended to be a useful default, but for
// now it's recommended that you explicitly set Propagation, since the default
// for this may change.
type Transport struct {
	// Base may be set to wrap another http.RoundTripper that does the actual
	// requests. By default http.DefaultTransport is used.
	//
	// If base HTTP roundtripper implements CancelRequest,
	// the returned round tripper will be cancelable.
	Base http.RoundTripper

	// Propagation defines how traces are propagated. If unspecified, a default
	// (currently B3 format) will be used.
	Propagation propagation.HTTPFormat

	// StartOptions are applied to the span started by this Transport around each
	// request.
	//
	// StartOptions.SpanKind will always be set to trace.SpanKindClient
	// for spans started by this transport.
	StartOptions trace.StartOptions

	// GetStartOptions allows to set start options per request. If set,
	// StartOptions is going to be ignored.
	GetStartOptions func(*http.Request) trace.StartOptions

	// NameFromRequest holds the function to use for generating the span name
	// from the information found in the outgoing HTTP Request. By default the
	// name equals the URL Path.
	FormatSpanName func(*http.Request) string

	// NewClientTrace may be set to a function allowing the current *trace.Span
	// to be annotated with HTTP request event information emitted by the
	// httptrace package.
	NewClientTrace func(*http.Request, *trace.Span) *httptrace.ClientTrace

	// TODO: Implement tag propagation for HTTP.

	// Tag Mutator
	tags []tagGenerator
}

type tagGenerator func(*http.Request) tag.Mutator

// RoundTrip implements http.RoundTripper, delegating to Base and recording stats and traces for the request.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt := t.base()
	if isHealthEndpoint(req.URL.Path) {
		return rt.RoundTrip(req)
	}
	// TODO: remove excessive nesting of http.RoundTrippers here.
	format := t.Propagation
	if format == nil {
		format = defaultFormat
	}
	spanNameFormatter := t.FormatSpanName
	if spanNameFormatter == nil {
		spanNameFormatter = SpanNameFromURL
	}

	startOpts := t.StartOptions
	if t.GetStartOptions != nil {
		startOpts = t.GetStartOptions(req)
	}

	rt = &traceTransport{
		base:   rt,
		format: format,
		startOptions: trace.StartOptions{
			Sampler:  startOpts.Sampler,
			SpanKind: trace.SpanKindClient,
		},
		formatSpanName: spanNameFormatter,
		newClientTrace: t.NewClientTrace,
	}
	rt = statsTransport{base: rt, tags: t.tags}
	return rt.RoundTrip(req)
}

func (t *Transport) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}
	return http.DefaultTransport
}

// CancelRequest cancels an in-flight request by closing its connection.
func (t *Transport) CancelRequest(req *http.Request) {
	type canceler interface {
		CancelRequest(*http.Request)
	}
	if cr, ok := t.base().(canceler); ok {
		cr.CancelRequest(req)
	}
}

type traceTransport struct {
	base           http.RoundTripper
	startOptions   trace.StartOptions
	format         propagation.HTTPFormat
	formatSpanName func(*http.Request) string
	newClientTrace func(*http.Request, *trace.Span) *httptrace.ClientTrace
}

// RoundTrip creates a trace.Span and inserts it into the outgoing request's headers.
// The created span can follow a parent span, if a parent is presented in
// the request's context.
func (t *traceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	name := t.formatSpanName(req)
	// TODO(jbd): Discuss whether we want to prefix
	// outgoing requests with Sent.
	ctx, span := trace.StartSpan(req.Context(), name,
		trace.WithSampler(t.startOptions.Sampler),
		trace.WithSpanKind(trace.SpanKindClient))

	if t.newClientTrace != nil {
		req = req.WithContext(httptrace.WithClientTrace(ctx, t.newClientTrace(req, span)))
	} else {
		req = req.WithContext(ctx)
	}

	if t.format != nil {
		// SpanContextToRequest will modify its Request argument, which is
		// contrary to the contract for http.RoundTripper, so we need to
		// pass it a copy of the Request.
		// However, the Request struct itself was already copied by
		// the WithContext calls above and so we just need to copy the header.
		header := make(http.Header)
		for k, v := range req.Header {
			header[k] = v
		}
		req.Header = header
		t.format.SpanContextToRequest(span.SpanContext(), req)
	}

	span.AddAttributes(RequestAttrs(req)...)
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		span.SetStatus(trace.Status{Code: trace.StatusCodeUnknown, Message: err.Error()})
		span.End()
		return resp, err
	}

	span.AddAttributes(ResponseAttrs(resp)...)
	span.SetStatus(TraceStatus(resp.StatusCode, resp.Status))

	// span.End() will be invoked after
	// a read from resp.Body returns io.EOF or when
	// resp.Body.Close() is invoked.
	bt := &bodyTracker{rc: resp.Body, span: span}
	resp.Body = wrappedBody(bt, resp.Body)
	return resp, err
}

// wrappedBody returns a wrapped version of the original
// Body and only implements the same combination of additional
// interfaces as the original.
func wrappedBody(wrapper io.ReadCloser, body io.ReadCloser) io.ReadCloser {
	wr, i0 := body.(io.Writer)
	switch {
	case !i0:
		return struct {
			io.ReadCloser
		}{wrapper}

	case i0:
		return struct {
			io.ReadCloser
			io.Writer
		}{wrapper, wr}
	default:
		return struct {
			io.ReadCloser
		}{wrapper}
	}
}

// bodyTracker wraps a response.Body and invokes
// trace.EndSpan on encountering io.EOF on reading
// the body of the original response.
type bodyTracker struct {
	rc   io.ReadCloser
	span *trace.Span
}

var _ io.ReadCloser = (*bodyTracker)(nil)

func (bt *bodyTracker) Read(b []byte) (int, error) {
	n, err := bt.rc.Read(b)

	switch err {
	case nil:
		return n, nil
	case io.EOF:
		bt.span.End()
	default:
		// For all other errors, set the span status
		bt.span.SetStatus(trace.Status{
			// Code 2 is the error code for Internal server error.
			Code:    2,
			Message: err.Error(),
		})
	}
	return n, err
}

func (bt *bodyTracker) Close() error {
	// Invoking endSpan on Close will help catch the cases
	// in which a read returned a non-nil error, we set the
	// span status but didn't end the span.
	bt.span.End()
	return bt.rc.Close()
}

// TraceStatus is a utility to convert the HTTP status code to a trace.Status that
// represents the outcome as closely as possible.
func TraceStatus(httpStatusCode int, statusLine string) trace.Status {
	var code int32
	if httpStatusCode < 200 || httpStatusCode >= 400 {
		code = trace.StatusCodeUnknown
	}
	switch httpStatusCode {
	case 499:
		code = trace.StatusCodeCancelled
	case http.StatusBadRequest:
		code = trace.StatusCodeInvalidArgument
	case http.StatusUnprocessableEntity:
		code = trace.StatusCodeInvalidArgument
	case http.StatusGatewayTimeout:
		code = trace.StatusCodeDeadlineExceeded
	case http.StatusNotFound:
		code = trace.StatusCodeNotFound
	case http.StatusForbidden:
		code = trace.StatusCodePermissionDenied
	case http.StatusUnauthorized: // 401 is actually unauthenticated.
		code = trace.StatusCodeUnauthenticated
	case http.StatusTooManyRequests:
		code = trace.StatusCodeResourceExhausted
	case http.StatusNotImplemented:
		code = trace.StatusCodeUnimplemented
	case http.StatusServiceUnavailable:
		code = trace.StatusCodeUnavailable
	case http.StatusOK:
		code = trace.StatusCodeOK
	}
	return trace.Status{Code: code, Message: codeToStr[code]}
}

var codeToStr = map[int32]string{
	trace.StatusCodeOK:                 `OK`,
	trace.StatusCodeCancelled:          `CANCELLED`,
	trace.StatusCodeUnknown:            `UNKNOWN`,
	trace.StatusCodeInvalidArgument:    `INVALID_ARGUMENT`,
	trace.StatusCodeDeadlineExceeded:   `DEADLINE_EXCEEDED`,
	trace.StatusCodeNotFound:           `NOT_FOUND`,
	trace.StatusCodeAlreadyExists:      `ALREADY_EXISTS`,
	trace.StatusCodePermissionDenied:   `PERMISSION_DENIED`,
	trace.StatusCodeResourceExhausted:  `RESOURCE_EXHAUSTED`,
	trace.StatusCodeFailedPrecondition: `FAILED_PRECONDITION`,
	trace.StatusCodeAborted:            `ABORTED`,
	trace.StatusCodeOutOfRange:         `OUT_OF_RANGE`,
	trace.StatusCodeUnimplemented:      `UNIMPLEMENTED`,
	trace.StatusCodeInternal:           `INTERNAL`,
	trace.StatusCodeUnavailable:        `UNAVAILABLE`,
	trace.StatusCodeDataLoss:           `DATA_LOSS`,
	trace.StatusCodeUnauthenticated:    `UNAUTHENTICATED`,
}

func isHealthEndpoint(path string) bool {
	if path == "/healthz" || path == "/_ah/health" {
		return true
	}
	return false
}

// statsTransport is an http.RoundTripper that collects stats for the outgoing requests.
type statsTransport struct {
	base http.RoundTripper

	// Tag Mutator
	tags []tagGenerator
}

// RoundTrip implements http.RoundTripper, delegating to Base and recording stats for the request.
func (t statsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tags := make([]tag.Mutator, len(t.tags))
	for i, tg := range t.tags {
		tags[i] = tg(req)
	}
	ctx, _ := tag.New(req.Context(), tags...)

	req = req.WithContext(ctx)
	track := &tracker{
		start: time.Now(),
		ctx:   ctx,
	}
	if req.Body == nil {
		// TODO: Handle cases where ContentLength is not set.
		track.reqSize = -1
	} else if req.ContentLength > 0 {
		track.reqSize = req.ContentLength
	}
	stats.Record(ctx, ochttp.ClientRequestCount.M(1))

	// Perform request.
	resp, err := t.base.RoundTrip(req)

	if err != nil {
		track.statusCode = http.StatusInternalServerError
		track.end()
	} else {
		track.statusCode = resp.StatusCode
		if req.Method != "HEAD" {
			track.respContentLength = resp.ContentLength
		}
		if resp.Body == nil {
			track.end()
		} else {
			track.body = resp.Body
			resp.Body = wrappedBody(track, resp.Body)
		}
	}
	return resp, err
}

// CancelRequest cancels an in-flight request by closing its connection.
func (t statsTransport) CancelRequest(req *http.Request) {
	type canceler interface {
		CancelRequest(*http.Request)
	}
	if cr, ok := t.base.(canceler); ok {
		cr.CancelRequest(req)
	}
}

type tracker struct {
	ctx               context.Context
	respSize          int64
	respContentLength int64
	reqSize           int64
	start             time.Time
	body              io.ReadCloser
	statusCode        int
	endOnce           sync.Once
}

var _ io.ReadCloser = (*tracker)(nil)

func (t *tracker) end() {
	t.endOnce.Do(func() {
		latencyMs := float64(time.Since(t.start)) / float64(time.Millisecond)
		respSize := t.respSize
		if t.respSize == 0 && t.respContentLength > 0 {
			respSize = t.respContentLength
		}
		m := []stats.Measurement{
			ochttp.ClientSentBytes.M(t.reqSize),
			ochttp.ClientReceivedBytes.M(respSize),
			ochttp.ClientRoundtripLatency.M(latencyMs),
			ochttp.ClientLatency.M(latencyMs),
			ochttp.ClientResponseBytes.M(t.respSize),
		}
		if t.reqSize >= 0 {
			m = append(m, ochttp.ClientRequestBytes.M(t.reqSize))
		}

		stats.RecordWithTags(t.ctx, []tag.Mutator{
			tag.Upsert(ochttp.StatusCode, strconv.Itoa(t.statusCode)),
			tag.Upsert(ochttp.KeyClientStatus, strconv.Itoa(t.statusCode)),
		}, m...)
	})
}

func (t *tracker) Read(b []byte) (int, error) {
	n, err := t.body.Read(b)
	t.respSize += int64(n)
	switch err {
	case nil:
		return n, nil
	case io.EOF:
		t.end()
	}
	return n, err
}

func (t *tracker) Close() error {
	// Invoking endSpan on Close will help catch the cases
	// in which a read returned a non-nil error, we set the
	// span status but didn't end the span.
	t.end()
	return t.body.Close()
}
