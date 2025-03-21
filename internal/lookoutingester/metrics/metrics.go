package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/armadaproject/armada/internal/common/ingest/metrics"
)

// Lookout ingester specific metrics
var avRowChangeTimeHist = promauto.NewHistogram(
	prometheus.HistogramOpts{
		Name:    metrics.ArmadaLookoutIngesterMetricsPrefix + "average_row_change_time",
		Help:    "Average time take in milliseconds to change one database row",
		Buckets: []float64{0.1, 0.2, 0.5, 1, 2, 3, 5, 7, 10, 15, 25, 50, 100, 1000},
	},
)

var avRowChangeTimeByOperationHist = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    metrics.ArmadaLookoutIngesterMetricsPrefix + "average_row_change_time_by_operation",
		Help:    "Average time take in milliseconds to change one database row",
		Buckets: []float64{0.1, 0.2, 0.5, 1, 2, 3, 5, 7, 10, 15, 25, 50, 100, 1000},
	},
	[]string{"table", "operation"},
)

var rowsChangedCounter = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: metrics.ArmadaLookoutIngesterMetricsPrefix + "rows_changed",
		Help: "Number of rows changed in the database",
	},
	[]string{"table", "operation"},
)

type Metrics struct {
	*metrics.Metrics
}

var m = &Metrics{
	metrics.NewMetrics(metrics.ArmadaLookoutIngesterMetricsPrefix),
}

func Get() *Metrics {
	return m
}

func (m *Metrics) RecordAvRowChangeTime(numRows int, duration time.Duration) {
	rowsPerMilli := float64(duration.Milliseconds()) / float64(numRows)
	avRowChangeTimeHist.Observe(rowsPerMilli)
}

func (m *Metrics) RecordAvRowChangeTimeByOperation(table string, operation metrics.DBOperation, numRows int, duration time.Duration) {
	rowsPerMilli := float64(duration.Milliseconds()) / float64(numRows)
	avRowChangeTimeByOperationHist.
		With(map[string]string{"table": table, "operation": string(operation)}).
		Observe(rowsPerMilli)
}

func (m *Metrics) RecordRowsChange(table string, operation metrics.DBOperation, numRows int) {
	rowsChangedCounter.
		With(map[string]string{"table": table, "operation": string(operation)}).
		Add(float64(numRows))
}
