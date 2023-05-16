/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

package gnmic

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-cmd/cmd"
	dconf "github.com/orb-community/orb-gnmic/config"
	"github.com/orb-community/orb/agent/backend"
	"github.com/orb-community/orb/agent/config"
	"github.com/orb-community/orb/agent/policies"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

var _ backend.Backend = (*gnmicBackend)(nil)

const (
	DefaultBinary       = "/usr/local/bin/gnmic-agent"
	DefaultHost         = "localhost"
	DefaultPort         = "10337"
	ReadinessBackoff    = 10
	ReadinessTimeout    = 10
	ApplyPolicyTimeout  = 10
	RemovePolicyTimeout = 20
	VersionTimeout      = 2
	ScrapeTimeout       = 5
	TapsTimeout         = 5
)

type gnmicBackend struct {
	logger     *zap.Logger
	binary     string
	configFile string
	version    string
	proc       *cmd.Cmd
	statusChan <-chan cmd.Status
	startTime  time.Time
	cancelFunc context.CancelFunc
	ctx        context.Context

	// MQTT Config for OTEL MQTT Exporter
	mqttConfig config.MQTTConfig

	mqttClient  *mqtt.Client
	metricTopic string
	logTopic    string
	policyRepo  policies.PolicyRepo

	adminAPIHost     string
	adminAPIPort     string
	adminAPIProtocol string

	// added for Strings
	agentTags map[string]string

	// OpenTelemetry management
	otelMetricsReceiverHost string
	otelMetricsReceiverPort int
	otelLogsReceiverHost    string
	otelLogsReceiverPort    int

	// control scrap individually
	receiver   map[string]receiver.Metrics
	exporter   map[string]exporter.Metrics
	routineMap map[string]context.CancelFunc

	// Prometheus Receiver
	PromParams url.Values
}

func Register() bool {
	backend.Register("gnmic", &gnmicBackend{
		adminAPIProtocol: "http",
	})
	return true
}

func (d *gnmicBackend) getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func (d *gnmicBackend) addScraperProcess(ctx context.Context, cancel context.CancelFunc, policyID string, policyName string) {
	attributeCtx := context.WithValue(ctx, "policy_name", policyName)
	attributeCtx = context.WithValue(attributeCtx, "policy_id", policyID)
	d.scrapeOtlp(attributeCtx)
	d.routineMap[policyID] = cancel
}

func (d *gnmicBackend) killScraperProcess(policyID string) {
	cancel := d.routineMap[policyID]
	if cancel != nil {
		cancel()
		delete(d.routineMap, policyID)
	}
}

func (d *gnmicBackend) GetStartTime() time.Time {
	return d.startTime
}

func (d *gnmicBackend) GetCapabilities() (map[string]interface{}, error) {
	return make(map[string]interface{}), nil
}

func (d *gnmicBackend) GetRunningStatus() (backend.RunningStatus, string, error) {
	// first check process status
	runningStatus, errMsg, err := d.getProcRunningStatus()
	// if it's not running, we're done
	if runningStatus != backend.Running {
		return runningStatus, errMsg, err
	}
	// if it's running, check REST API availability too
	var dConf dconf.Status
	if err := d.request("status", &dConf, http.MethodGet, http.NoBody, "application/json", VersionTimeout); err != nil {
		return backend.BackendError, "process running, REST API unavailable", err
	}
	return runningStatus, "", nil
}

func (d *gnmicBackend) Version() (string, error) {
	var dConf dconf.Status
	if err := d.request("status", &dConf, http.MethodGet, http.NoBody, "application/json", VersionTimeout); err != nil {
		return "", err
	}
	return dConf.Version, nil
}

func (d *gnmicBackend) SetCommsClient(agentID string, client *mqtt.Client, baseTopic string) {
	d.mqttClient = client
	otlpTopic := strings.Replace(baseTopic, "?", "otlp", 1)
	d.metricTopic = fmt.Sprintf("%s/m/%c", otlpTopic, agentID[0])
	d.logTopic = fmt.Sprintf("%s/l/%c", otlpTopic, agentID[0])
}

func (d *gnmicBackend) Configure(logger *zap.Logger, repo policies.PolicyRepo, config map[string]string, otelConfig map[string]interface{}) error {
	d.logger = logger
	d.policyRepo = repo

	var err bool
	if d.binary, err = config["binary"]; !err {
		d.binary = DefaultBinary
	}
	if d.configFile, err = config["config_file"]; !err {
		d.configFile = ""
	}
	if d.adminAPIHost, err = config["api_host"]; !err {
		d.adminAPIHost = DefaultHost
	}
	if d.adminAPIPort, err = config["api_port"]; !err {
		d.adminAPIPort = DefaultPort
	}
	if agentTags, ok := otelConfig["agent_tags"]; ok {
		d.agentTags = agentTags.(map[string]string)
	}
	return nil
}

func (d *gnmicBackend) Start(ctx context.Context, cancelFunc context.CancelFunc) error {

	// this should record the start time whether it's successful or not
	// because it is used by the automatic restart system for last attempt
	d.startTime = time.Now()
	d.cancelFunc = cancelFunc
	d.ctx = ctx

	_, err := exec.LookPath(d.binary)
	if err != nil {
		d.logger.Error("gnmic-agent startup error: binary not found", zap.Error(err))
		return err
	}

	pvOptions := []string{
		"run",
		"-a",
		d.adminAPIHost,
		"-p",
		d.adminAPIPort,
	}

	// Metrics
	if d.otelMetricsReceiverPort == 0 {
		d.otelMetricsReceiverPort = 8080
	}

	if len(d.otelMetricsReceiverHost) == 0 {
		d.otelMetricsReceiverHost = DefaultHost
	}

	d.logger.Info("gnmic-agent startup", zap.Strings("arguments", pvOptions))

	d.proc = cmd.NewCmdOptions(cmd.Options{
		Buffered:  false,
		Streaming: true,
	}, d.binary, pvOptions...)
	d.statusChan = d.proc.Start()

	// log STDOUT and STDERR lines streaming from Cmd
	doneChan := make(chan struct{})
	go func() {
		defer func() {
			if doneChan != nil {
				close(doneChan)
			}
		}()
		for d.proc.Stdout != nil || d.proc.Stderr != nil {
			select {
			case line, open := <-d.proc.Stdout:
				if !open {
					d.proc.Stdout = nil
					continue
				}
				d.logger.Info("gnmic-agent stdout", zap.String("log", line))
			case line, open := <-d.proc.Stderr:
				if !open {
					d.proc.Stderr = nil
					continue
				}
				d.logger.Info("gnmic-agent stderr", zap.String("log", line))
			}
		}
	}()

	// wait for simple startup errors
	time.Sleep(time.Second)

	status := d.proc.Status()

	if status.Error != nil {
		d.logger.Error("gnmic-agent startup error", zap.Error(status.Error))
		return status.Error
	}

	if status.Complete {
		err = d.proc.Stop()
		if err != nil {
			d.logger.Error("proc.Stop error", zap.Error(err))
		}
		return errors.New("gnmic-agent startup error, check log")
	}

	d.logger.Info("gnmic-agent process started", zap.Int("pid", status.PID))

	var readinessError error
	for backoff := 0; backoff < ReadinessBackoff; backoff++ {
		var dConf dconf.Status
		readinessError = d.request("status", &dConf, http.MethodGet, http.NoBody, "application/json", ReadinessTimeout)
		if readinessError == nil {
			d.logger.Info("gnmic-agent readiness ok, got version ", zap.String("otelinf_agent_version", dConf.Version))
			break
		}
		backoffDuration := time.Duration(backoff) * time.Second
		d.logger.Info("gnmic-agent is not ready, trying again with backoff", zap.String("backoff backoffDuration", backoffDuration.String()))
		time.Sleep(backoffDuration)
	}

	if readinessError != nil {
		d.logger.Error("gnmic-agent error on readiness", zap.Error(readinessError))
		err = d.proc.Stop()
		if err != nil {
			d.logger.Error("proc.Stop error", zap.Error(err))
		}
		return readinessError
	}

	return nil
}

func (d *gnmicBackend) Stop(ctx context.Context) error {
	d.logger.Info("routine call to stop gnmic-agent", zap.Any("routine", ctx.Value("routine")))
	defer d.cancelFunc()
	err := d.proc.Stop()
	finalStatus := <-d.statusChan
	if err != nil {
		d.logger.Error("gnmic-agent shutdown error", zap.Error(err))
	}
	d.logger.Info("gnmic-agent process stopped", zap.Int("pid", finalStatus.PID), zap.Int("exit_code", finalStatus.Exit))
	return nil
}

func (d *gnmicBackend) FullReset(ctx context.Context) error {
	// force a stop, which stops scrape as well. if proc is dead, it no ops.
	if state, _, _ := d.getProcRunningStatus(); state == backend.Running {
		if err := d.Stop(ctx); err != nil {
			d.logger.Error("failed to stop backend on restart procedure", zap.Error(err))
			return err
		}
	}
	backendCtx, cancelFunc := context.WithCancel(context.WithValue(ctx, "routine", "gnmic-agent"))
	// start it
	if err := d.Start(backendCtx, cancelFunc); err != nil {
		d.logger.Error("failed to start backend on restart procedure", zap.Error(err))
		return err
	}
	return nil
}
