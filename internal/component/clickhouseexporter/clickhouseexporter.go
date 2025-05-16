package clickhouseexporter

import (
	"context"
	"fmt"
	"time"

	"github.com/grafana/alloy/internal/component"
	"github.com/grafana/alloy/internal/component/otelcol"
	"github.com/grafana/alloy/internal/featuregate"
	// "github.com/grafana/alloy/syntax" // Not used directly in the provided design, can be removed if not needed
	otelcomponent "go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickhouseexporter"
	// "go.opentelemetry.io/collector/confmap" // Not used directly in the provided design, can be removed if not needed
)

// Static check that Component implements necessary interfaces.
var _ component.Component = (*Component)(nil)
var _ otelcol.Exporter = (*Component)(nil)

func init() {
	component.Register(component.Registration{
		Name: "otelcol.exporter.clickhouse",
		Args: Arguments{},
		Exports: otelcol.ConsumerExports{},
		Build: func(opts component.Options, args component.Arguments) (component.Component, error) {
			if !featuregate.IsEnabled(featuregate.CommunityComponents) {
				return nil, fmt.Errorf("otelcol.exporter.clickhouse component is experimental and must be enabled via the --feature.community-components.enabled flag")
			}
			return New(opts, args.(Arguments))
		},
	})
}

// Arguments defines the configuration for the otelcol.exporter.clickhouse component.
type Arguments struct {
	DSN     string `river:"dsn,attr"`
	Timeout string `river:"timeout,attr,optional"`

	Logs    LogsConfig    `river:"logs,block,optional"`
	Traces  TracesConfig  `river:"traces,block,optional"`
	Metrics MetricsConfig `river:"metrics,block,optional"`

	Retry      otelcol.RetryArguments       `river:"retry_on_failure,block,optional"`
	Queue      otelcol.QueueSettings        `river:"sending_queue,block,optional"`
	ClickHouse ClickHouseSpecificConfig `river:"clickhouse,block,optional"`

	ForwardTo []otelcol.Consumer `river:"forward_to,attr"`
}

// SetToDefault implements river.Defaulter.
func (args *Arguments) SetToDefault() {
	args.Retry = otelcol.DefaultRetryArguments()
	args.Queue = otelcol.DefaultQueueSettings()
	args.Logs = DefaultLogsConfig()
	args.Traces = DefaultTracesConfig()
	args.Metrics = DefaultMetricsConfig()
	args.ClickHouse = DefaultClickHouseSpecificConfig()
    // DSN is mandatory, no default.
    // Timeout has a default in the underlying exporter if not set or empty string.
}

// LogsConfig defines settings for log exporting.
type LogsConfig struct {
	Table   string `river:"table,attr,optional"`
	TTLDays int    `river:"ttl_days,attr,optional"`
}

// DefaultLogsConfig returns the default settings for log exporting.
func DefaultLogsConfig() LogsConfig {
	return LogsConfig{
		Table:   "otel_logs",
		TTLDays: 0,
	}
}

// TracesConfig defines settings for trace exporting.
type TracesConfig struct {
	Table   string `river:"table,attr,optional"`
	TTLDays int    `river:"ttl_days,attr,optional"`
}

// DefaultTracesConfig returns the default settings for trace exporting.
func DefaultTracesConfig() TracesConfig {
	return TracesConfig{
		Table:   "otel_traces",
		TTLDays: 0,
	}
}

// MetricsConfig defines settings for metric exporting.
type MetricsConfig struct {
	Table   string `river:"table,attr,optional"` // Base table name
	TTLDays int    `river:"ttl_days,attr,optional"`
}

// DefaultMetricsConfig returns the default settings for metric exporting.
func DefaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		Table:   "otel_metrics",
		TTLDays: 0,
	}
}

// ClickHouseSpecificConfig defines ClickHouse specific settings.
type ClickHouseSpecificConfig struct {
	ClusterName string `river:"cluster_name,attr,optional"`
	TableEngine string `river:"table_engine,attr,optional"`
}

// DefaultClickHouseSpecificConfig returns the default ClickHouse specific settings.
func DefaultClickHouseSpecificConfig() ClickHouseSpecificConfig {
	return ClickHouseSpecificConfig{}
}

// Component is the Alloy component for the ClickHouse exporter.
type Component struct {
	opts    component.Options
	args    Arguments

	exporter   otelcomponent.Exporter
	cancelFunc context.CancelFunc
}

// New creates a new ClickHouseExporter component.
func New(opts component.Options, args Arguments) (*Component, error) {
	c := &Component{
		opts: opts,
	}

	if err := c.Update(args); err != nil {
		return nil, err
	}
	return c, nil
}

// Run starts the component.
func (c *Component) Run(ctx context.Context) error {
	runCtx, cancel := context.WithCancel(ctx)
	c.cancelFunc = cancel

	if err := c.exporter.Start(runCtx, c.opts.Host); err != nil {
		return fmt.Errorf("failed to start underlying clickhouse exporter: %w", err)
	}

	<-runCtx.Done()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := c.exporter.Shutdown(shutdownCtx); err != nil {
		c.opts.Logger.Error("failed to shutdown underlying clickhouse exporter", "err", err)
	}
	return nil
}

// Update reconfigures the component.
func (c *Component) Update(args component.Arguments) error {
	newArgs := args.(Arguments)
	c.args = newArgs

	factory := clickhouseexporter.NewFactory()
	cfg := factory.CreateDefaultConfig().(*clickhouseexporter.Config)

	cfg.DSN = newArgs.DSN
	if newArgs.Timeout != "" {
		timeout, err := time.ParseDuration(newArgs.Timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout duration: %w", err)
		}
		cfg.TimeoutSettings.Timeout = timeout
	}

	cfg.LogsTableName = newArgs.Logs.Table
	cfg.LogsTTLDays = newArgs.Logs.TTLDays
	cfg.TracesTableName = newArgs.Traces.Table
	cfg.TracesTTLDays = newArgs.Traces.TTLDays
	cfg.MetricsTableName = newArgs.Metrics.Table
	cfg.MetricsTTLDays = newArgs.Metrics.TTLDays

	cfg.RetrySettings = exporterhelper.RetrySettings{
		Enabled:             newArgs.Retry.Enabled,
		InitialInterval:     newArgs.Retry.InitialInterval,
		MaxInterval:         newArgs.Retry.MaxInterval,
		MaxElapsedTime:      newArgs.Retry.MaxElapsedTime,
	}
	cfg.QueueSettings = exporterhelper.QueueSettings{
		Enabled:      newArgs.Queue.Enabled,
		NumConsumers: newArgs.Queue.NumConsumers,
		QueueSize:    newArgs.Queue.QueueSize,
	}

	cfg.ClusterName = newArgs.ClickHouse.ClusterName
	cfg.TableEngine = newArgs.ClickHouse.TableEngine

	exporterSettings := exporter.CreateSettings{
		ID:                otelcomponent.NewIDWithName(factory.Type(), c.opts.ID),
		TelemetrySettings: c.opts.Telemetry.ToOtelComponentTelemetrySettings(),
		BuildInfo:         otelcomponent.BuildInfo{Command: "alloy", Version: "dev"}, // TODO: Get version from Alloy build info
	}

	var err error
	// The ClickHouse exporter factory creates an exporter that handles all signal types.
	// We can use any of the Create*Exporter methods.
	exp, err := factory.CreateTracesExporter(context.Background(), exporterSettings, cfg)
	if err != nil {
		return fmt.Errorf("failed to create clickhouse exporter: %w", err)
	}
	c.exporter = exp

	// Notify Alloy about the consumers for the forward_to argument.
	c.opts.Registerer.RegisterConsumers(newArgs.ForwardTo, c.exporter.(otelcol.LogsConsumer), c.exporter.(otelcol.TracesConsumer), c.exporter.(otelcol.MetricsConsumer))

	return nil
}

// Shutdown stops the component.
func (c *Component) Shutdown() error {
	if c.cancelFunc != nil {
		c.cancelFunc()
	}
	return nil
}

// Consumers returns the exporter itself as a consumer for all data types.
func (c *Component) Consumers() (otelcol.LogsConsumer, otelcol.TracesConsumer, otelcol.MetricsConsumer) {
	if c.exporter == nil {
		return nil, nil, nil
	}
	// The created exporter from clickhouseexporter.NewFactory() should implement these interfaces.
	// It's designed to be a multi-signal exporter.
	return c.exporter.(otelcol.LogsConsumer), c.exporter.(otelcol.TracesConsumer), c.exporter.(otelcol.MetricsConsumer)
}

