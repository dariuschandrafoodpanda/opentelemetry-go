// Copyright The OpenTelemetry Authors
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

package otest // import "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/internal/otest"

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/metric/unit"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	cpb "go.opentelemetry.io/proto/otlp/common/v1"
	mpb "go.opentelemetry.io/proto/otlp/metrics/v1"
	rpb "go.opentelemetry.io/proto/otlp/resource/v1"
)

var (
	// Sat Jan 01 2000 00:00:00 GMT+0000.
	start = time.Date(2000, time.January, 01, 0, 0, 0, 0, time.FixedZone("GMT", 0))
	end   = start.Add(30 * time.Second)

	kvAlice = &cpb.KeyValue{Key: "user", Value: &cpb.AnyValue{
		Value: &cpb.AnyValue_StringValue{StringValue: "alice"},
	}}
	kvBob = &cpb.KeyValue{Key: "user", Value: &cpb.AnyValue{
		Value: &cpb.AnyValue_StringValue{StringValue: "bob"},
	}}
	kvSrvName = &cpb.KeyValue{Key: "service.name", Value: &cpb.AnyValue{
		Value: &cpb.AnyValue_StringValue{StringValue: "test server"},
	}}
	kvSrvVer = &cpb.KeyValue{Key: "service.version", Value: &cpb.AnyValue{
		Value: &cpb.AnyValue_StringValue{StringValue: "v0.1.0"},
	}}

	min, max, sum = 2.0, 4.0, 90.0
	hdp           = []*mpb.HistogramDataPoint{{
		Attributes:        []*cpb.KeyValue{kvAlice},
		StartTimeUnixNano: uint64(start.UnixNano()),
		TimeUnixNano:      uint64(end.UnixNano()),
		Count:             30,
		Sum:               &sum,
		ExplicitBounds:    []float64{1, 5},
		BucketCounts:      []uint64{0, 30, 0},
		Min:               &min,
		Max:               &max,
	}}

	hist = &mpb.Histogram{
		AggregationTemporality: mpb.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
		DataPoints:             hdp,
	}

	dPtsInt64 = []*mpb.NumberDataPoint{
		{
			Attributes:        []*cpb.KeyValue{kvAlice},
			StartTimeUnixNano: uint64(start.UnixNano()),
			TimeUnixNano:      uint64(end.UnixNano()),
			Value:             &mpb.NumberDataPoint_AsInt{AsInt: 1},
		},
		{
			Attributes:        []*cpb.KeyValue{kvBob},
			StartTimeUnixNano: uint64(start.UnixNano()),
			TimeUnixNano:      uint64(end.UnixNano()),
			Value:             &mpb.NumberDataPoint_AsInt{AsInt: 2},
		},
	}
	dPtsFloat64 = []*mpb.NumberDataPoint{
		{
			Attributes:        []*cpb.KeyValue{kvAlice},
			StartTimeUnixNano: uint64(start.UnixNano()),
			TimeUnixNano:      uint64(end.UnixNano()),
			Value:             &mpb.NumberDataPoint_AsDouble{AsDouble: 1.0},
		},
		{
			Attributes:        []*cpb.KeyValue{kvBob},
			StartTimeUnixNano: uint64(start.UnixNano()),
			TimeUnixNano:      uint64(end.UnixNano()),
			Value:             &mpb.NumberDataPoint_AsDouble{AsDouble: 2.0},
		},
	}

	sumInt64 = &mpb.Sum{
		AggregationTemporality: mpb.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
		IsMonotonic:            true,
		DataPoints:             dPtsInt64,
	}
	sumFloat64 = &mpb.Sum{
		AggregationTemporality: mpb.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
		IsMonotonic:            false,
		DataPoints:             dPtsFloat64,
	}

	gaugeInt64   = &mpb.Gauge{DataPoints: dPtsInt64}
	gaugeFloat64 = &mpb.Gauge{DataPoints: dPtsFloat64}

	metrics = []*mpb.Metric{
		{
			Name:        "int64-gauge",
			Description: "Gauge with int64 values",
			Unit:        string(unit.Dimensionless),
			Data:        &mpb.Metric_Gauge{Gauge: gaugeInt64},
		},
		{
			Name:        "float64-gauge",
			Description: "Gauge with float64 values",
			Unit:        string(unit.Dimensionless),
			Data:        &mpb.Metric_Gauge{Gauge: gaugeFloat64},
		},
		{
			Name:        "int64-sum",
			Description: "Sum with int64 values",
			Unit:        string(unit.Dimensionless),
			Data:        &mpb.Metric_Sum{Sum: sumInt64},
		},
		{
			Name:        "float64-sum",
			Description: "Sum with float64 values",
			Unit:        string(unit.Dimensionless),
			Data:        &mpb.Metric_Sum{Sum: sumFloat64},
		},
		{
			Name:        "histogram",
			Description: "Histogram",
			Unit:        string(unit.Dimensionless),
			Data:        &mpb.Metric_Histogram{Histogram: hist},
		},
	}

	scope = &cpb.InstrumentationScope{
		Name:    "test/code/path",
		Version: "v0.1.0",
	}
	scopeMetrics = []*mpb.ScopeMetrics{{
		Scope:     scope,
		Metrics:   metrics,
		SchemaUrl: semconv.SchemaURL,
	}}

	res = &rpb.Resource{
		Attributes: []*cpb.KeyValue{kvSrvName, kvSrvVer},
	}
	resourceMetrics = &mpb.ResourceMetrics{
		Resource:     res,
		ScopeMetrics: scopeMetrics,
		SchemaUrl:    semconv.SchemaURL,
	}
)

// ClientFactory is a function that when called returns a
// otlpmetric.Client implementation that is connected to also returned
// Collector implementation. The Client is ready to upload metric data to the
// Collector which is ready to store that data.
type ClientFactory func() (otlpmetric.Client, Collector)

// RunClientTests runs a suite of Client integration tests. For example:
//
//	t.Run("Integration", RunClientTests(factory))
func RunClientTests(f ClientFactory) func(*testing.T) {
	return func(t *testing.T) {
		t.Run("ClientHonorsContextErrors", func(t *testing.T) {
			t.Run("Shutdown", testCtxErrs(func() func(context.Context) error {
				c, _ := f()
				return c.Shutdown
			}))

			t.Run("ForceFlush", testCtxErrs(func() func(context.Context) error {
				c, _ := f()
				return c.ForceFlush
			}))

			t.Run("UploadMetrics", testCtxErrs(func() func(context.Context) error {
				c, _ := f()
				return func(ctx context.Context) error {
					return c.UploadMetrics(ctx, nil)
				}
			}))
		})

		t.Run("ForceFlushFlushes", func(t *testing.T) {
			ctx := context.Background()
			client, collector := f()
			require.NoError(t, client.UploadMetrics(ctx, resourceMetrics))

			require.NoError(t, client.ForceFlush(ctx))
			rm := collector.Collect().Dump()
			// Data correctness is not important, just it was received.
			require.Greater(t, len(rm), 0, "no data uploaded")

			require.NoError(t, client.Shutdown(ctx))
			rm = collector.Collect().Dump()
			assert.Len(t, rm, 0, "client did not flush all data")
		})

		t.Run("UploadMetrics", func(t *testing.T) {
			ctx := context.Background()
			client, coll := f()

			require.NoError(t, client.UploadMetrics(ctx, resourceMetrics))
			require.NoError(t, client.Shutdown(ctx))
			got := coll.Collect().Dump()
			require.Len(t, got, 1, "upload of one ResourceMetrics")
			diff := cmp.Diff(got[0], resourceMetrics, cmp.Comparer(proto.Equal))
			if diff != "" {
				t.Fatalf("unexpected ResourceMetrics:\n%s", diff)
			}
		})
	}
}

func testCtxErrs(factory func() func(context.Context) error) func(t *testing.T) {
	return func(t *testing.T) {
		t.Helper()
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		t.Run("DeadlineExceeded", func(t *testing.T) {
			innerCtx, innerCancel := context.WithTimeout(ctx, time.Nanosecond)
			t.Cleanup(innerCancel)
			<-innerCtx.Done()

			f := factory()
			assert.ErrorIs(t, f(innerCtx), context.DeadlineExceeded)
		})

		t.Run("Canceled", func(t *testing.T) {
			innerCtx, innerCancel := context.WithCancel(ctx)
			innerCancel()

			f := factory()
			assert.ErrorIs(t, f(innerCtx), context.Canceled)
		})
	}
}
