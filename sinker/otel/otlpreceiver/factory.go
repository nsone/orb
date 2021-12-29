// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otlpreceiver // import "go.opentelemetry.io/collector/receiver/otlpreceiver"

import (
	"context"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/ns1labs/orb/sinker/otel/otlpreceiver/internal/sharedcomponent"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
)

const (
	typeStr = "otlp"

	defaultGRPCEndpoint = "0.0.0.0:4317"
	defaultHTTPEndpoint = "0.0.0.0:4318"
	legacyHTTPEndpoint  = "0.0.0.0:55681"
)

// NewFactory creates a new OTLP receiver factory.
func NewFactory() component.ReceiverFactory {
	receiver := receiverhelper.WithMetrics(CreateMetricsReceiver)
	return receiverhelper.NewFactory(
		typeStr,
		CreateDefaultConfig,
		receiver)
}

// createDefaultCreateSettings
func CreateDefaultCreateSetting(logger *zap.Logger) component.ReceiverCreateSettings {
	return component.ReceiverCreateSettings{
		TelemetrySettings: component.TelemetrySettings{
			Logger:         logger,
			TracerProvider: trace.NewNoopTracerProvider(),
			MeterProvider:  global.GetMeterProvider(),
		},
		BuildInfo: component.NewDefaultBuildInfo(),
	}
}

// createDefaultConfig creates the default configuration for receiver.
func CreateDefaultConfig() config.Receiver {
	return &Config{
		ReceiverSettings: config.NewReceiverSettings(config.NewComponentID(typeStr)),
		Protocols: Protocols{
			GRPC: &configgrpc.GRPCServerSettings{
				NetAddr: confignet.NetAddr{
					Endpoint:  defaultGRPCEndpoint,
					Transport: "tcp",
				},
				// We almost write 0 bytes, so no need to tune WriteBufferSize.
				ReadBufferSize: 512 * 1024,
			},
			HTTP: &confighttp.HTTPServerSettings{
				Endpoint: defaultHTTPEndpoint,
			},
		},
	}
}

// CreateMetricsReceiver creates a metrics receiver based on provided config.
func CreateMetricsReceiver(
	_ context.Context,
	set component.ReceiverCreateSettings,
	cfg config.Receiver,
	consumer consumer.Metrics,
) (component.MetricsReceiver, error) {
	r := receivers.GetOrAdd(cfg, func() component.Component {
		return newOtlpReceiver(cfg.(*Config), set)
	})

	if err := r.Unwrap().(*otlpReceiver).registerMetricsConsumer(consumer); err != nil {
		return nil, err
	}
	return r, nil
}

// This is the map of already created OTLP receivers for particular configurations.
// We maintain this map because the Factory is asked trace and metric receivers separately
// when it gets CreateTracesReceiver() and CreateMetricsReceiver() but they must not
// create separate objects, they must use one otlpReceiver object per configuration.
// When the receiver is shutdown it should be removed from this map so the same configuration
// can be recreated successfully.
var receivers = sharedcomponent.NewSharedComponents()
