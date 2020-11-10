// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package prom

import (
	"context"
	"net/http"
	"time"

	"github.com/m3db/m3/src/query/api/v1/handler/prometheus/native"
	"github.com/m3db/m3/src/query/api/v1/options"
	"github.com/m3db/m3/src/query/block"
	"github.com/m3db/m3/src/query/storage/prometheus"

	"github.com/prometheus/prometheus/promql"
	"github.com/prometheus/prometheus/promql/parser"
)

// NB: since Prometheus engine is not brought up in the usual fashion,
// default subquery evaluation interval is unset, causing div by 0 errors.
func init() {
	promql.SetDefaultEvaluationInterval(time.Minute)
}

// Options defines options for PromQL handler.
type Options struct {
	PromQLEngine *promql.Engine
}

// NewReadHandler creates a handler to handle PromQL requests.
func NewReadHandler(opts Options, hOpts options.HandlerOptions) http.Handler {
	var requestParser RequestParser = func(ctx context.Context, r *http.Request) (native.ParsedOptions, error) {
		return native.ParseRequest(ctx, r, false, hOpts)
	}
	return NewReadHandlerWithCustomParser(requestParser, opts, hOpts)
}

// NewReadHandlerWithCustomParser creates a handler to handle PromQL requests.
func NewReadHandlerWithCustomParser(parser RequestParser, opts Options, hOpts options.HandlerOptions) http.Handler {
	queryable := prometheus.NewPrometheusQueryable(
		prometheus.PrometheusOptions{
			Storage:           hOpts.Storage(),
			InstrumentOptions: hOpts.InstrumentOpts(),
		})
	return newReadHandler(parser, opts, hOpts, queryable)
}

/* ================ How it would look like in enterprise ================ */
type ReadOptions struct {
	PromQLEngine                  *promql.Engine
	AdjustStep                    time.Duration
	AdjustStepWhenStepDivisibleBy time.Duration
}

func NewEnterpriseReadHandler(readOpts ReadOptions, hOpts options.HandlerOptions) http.Handler {
	var requestParser RequestParser = func(ctx context.Context, r *http.Request) (native.ParsedOptions, error) {
		params, err := native.ParseRequest(ctx, r, false, hOpts)
		if err != nil {
			return params, err
		}

		whenStepDivisibleBy := readOpts.AdjustStepWhenStepDivisibleBy
		adjustStep := whenStepDivisibleBy == 0 ||
			(whenStepDivisibleBy != 0 && params.Params.Step%whenStepDivisibleBy == 0)
		if adjustStep {
			params.Params.Step += readOpts.AdjustStep
		}

		return params, nil
	}
	opts := Options{
		PromQLEngine: readOpts.PromQLEngine,
	}
	return NewReadHandlerWithCustomParser(requestParser, opts, hOpts)
}

// NewReadInstantHandler creates a handler to handle PromQL requests.
func NewReadInstantHandler(opts Options, hOpts options.HandlerOptions) http.Handler {
	queryable := prometheus.NewPrometheusQueryable(
		prometheus.PrometheusOptions{
			Storage:           hOpts.Storage(),
			InstrumentOptions: hOpts.InstrumentOpts(),
		})
	return newReadInstantHandler(opts, hOpts, queryable)
}

// ApplyRangeWarnings applies warnings encountered during execution.
func ApplyRangeWarnings(
	query string, meta *block.ResultMetadata,
) error {
	expr, err := parser.ParseExpr(query)
	if err != nil {
		return err
	}

	parser.Inspect(expr, func(node parser.Node, path []parser.Node) error {
		if n, ok := node.(*parser.MatrixSelector); ok {
			meta.VerifyTemporalRange(n.Range)
		}

		return nil
	})

	return nil
}
