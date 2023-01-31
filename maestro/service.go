// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Adapted for Orb project, modifications licensed under MPL v. 2.0:
/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

package maestro

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-redis/redis/v8"
	maestroconfig "github.com/ns1labs/orb/maestro/config"
	"github.com/ns1labs/orb/maestro/kubecontrol"
	rediscons1 "github.com/ns1labs/orb/maestro/redis/consumer"
	"github.com/ns1labs/orb/pkg/config"
	sinkspb "github.com/ns1labs/orb/sinks/pb"
	"go.uber.org/zap"
)

var _ Service = (*maestroService)(nil)

type maestroService struct {
	serviceContext    context.Context
	serviceCancelFunc context.CancelFunc

	kubecontrol       kubecontrol.Service
	monitor           kubecontrol.MonitorService
	logger            *zap.Logger
	streamRedisClient *redis.Client
	sinkerRedisClient *redis.Client
	sinksClient       sinkspb.SinkServiceClient
	esCfg             config.EsConfig
	eventStore        rediscons1.Subscriber
	kafkaUrl          string
}

func NewMaestroService(logger *zap.Logger, streamRedisClient *redis.Client, sinkerRedisClient *redis.Client, sinksGrpcClient sinkspb.SinkServiceClient, esCfg config.EsConfig, otelCfg config.OtelConfig) Service {
	kubectr := kubecontrol.NewService(logger, streamRedisClient)
	eventStore := rediscons1.NewEventStore(streamRedisClient, otelCfg.KafkaUrl, kubectr, esCfg.Consumer, sinksGrpcClient, logger)
	monitor := kubecontrol.NewMonitorService(logger, &sinksGrpcClient, streamRedisClient, sinkerRedisClient, &kubectr)
	return &maestroService{
		logger:            logger,
		streamRedisClient: streamRedisClient,
		sinksClient:       sinksGrpcClient,
		kubecontrol:       kubectr,
		monitor:           monitor,
		eventStore:        eventStore,
		kafkaUrl:          otelCfg.KafkaUrl,
	}
}

// Start will load all sinks from DB using SinksGRPC,
//
//	then for each sink, will create DeploymentEntry in Redis
//	And for each sink with active state, deploy OtelCollector
func (svc *maestroService) Start(ctx context.Context, cancelFunction context.CancelFunc) error {

	loadCtx, loadCancelFunction := context.WithCancel(ctx)
	defer loadCancelFunction()
	svc.serviceContext = ctx
	svc.serviceCancelFunc = cancelFunction

	sinksRes, err := svc.sinksClient.RetrieveSinks(loadCtx, &sinkspb.SinksFilterReq{OtelEnabled: "enabled"})
	if err != nil {
		loadCancelFunction()
		return err
	}

	pods, err := svc.monitor.GetRunningPods(ctx)
	if err != nil {
		loadCancelFunction()
		return err
	}

	for _, sinkRes := range sinksRes.Sinks {
		sinkContext := context.WithValue(loadCtx, "sink-id", sinkRes.Id)
		var data maestroconfig.SinkData
		if err := json.Unmarshal(sinkRes.Config, &data); err != nil {
			svc.logger.Warn("failed to unmarshal sink, skipping", zap.String("sink-id", sinkRes.Id))
			continue
		}

		if val, _ := svc.eventStore.GetDeploymentEntryFromSinkId(ctx, sinkRes.Id); val != "" {
			svc.logger.Info("Skipping deploymentEntry because it is already created")
		} else {
			err := svc.eventStore.CreateDeploymentEntry(sinkContext, sinkRes.Id, data.Url, data.User, data.Password)
			if err != nil {
				svc.logger.Warn("failed to create deploymentEntry for sink, skipping", zap.String("sink-id", sinkRes.Id))
				continue
			}
			err = svc.updateSinkCache(ctx, data)
			if err != nil {
				svc.logger.Warn("failed to update cache for sink", zap.String("sink-id", sinkRes.Id))
				continue
			}
			svc.logger.Info("successfully created deploymentEntry for sink", zap.String("sink-id", sinkRes.Id), zap.String("state", sinkRes.State))
		}

		isDeployed := false
		if len(pods) > 0 {
			for _, pod := range pods {
				if strings.Contains(pod, sinkRes.Id) {
					isDeployed = true
					break
				}
			}
		}
		// if State is Active, deploy OtelCollector
		if sinkRes.State == "active" && !isDeployed {
			deploymentEntry, err := svc.eventStore.GetDeploymentEntryFromSinkId(sinkContext, sinkRes.Id)
			if err != nil {
				svc.logger.Warn("failed to fetch deploymentEntry for sink, skipping", zap.String("sink-id", sinkRes.Id), zap.Error(err))
				continue
			}
			err = svc.kubecontrol.CreateOtelCollector(sinkContext, sinkRes.OwnerID, sinkRes.Id, deploymentEntry)
			if err != nil {
				svc.logger.Warn("failed to deploy OtelCollector for sink, skipping", zap.String("sink-id", sinkRes.Id), zap.Error(err))
				continue
			}
			svc.logger.Info("successfully created otel collector for sink", zap.String("sink-id", sinkRes.Id))
		}
	}

	go svc.subscribeToSinksES(ctx)
	go svc.subscribeToSinkerES(ctx)

	monitorCtx := context.WithValue(ctx, "routine", "monitor")
	err = svc.monitor.Start(monitorCtx, cancelFunction)
	if err != nil {
		svc.logger.Error("error during monitor routine start", zap.Error(err))
		cancelFunction()
		return err
	}

	return nil
}

func (svc *maestroService) updateSinkCache(ctx context.Context, data maestroconfig.SinkData) (err error) {
	data.State = maestroconfig.Unknown
	keyPrefix := "sinker_key"
	skey := fmt.Sprintf("%s-%s:%s", keyPrefix, data.OwnerID, data.SinkID)
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if err = svc.sinkerRedisClient.Set(ctx, skey, bytes, 0).Err(); err != nil {
		return err
	}
	if err != nil {
		return err
	}
	return
}

func (svc *maestroService) subscribeToSinkerES(ctx context.Context) {
	if err := svc.eventStore.SubscribeSinker(ctx); err != nil {
		svc.logger.Error("Bootstrap service failed to subscribe to event sourcing sinker", zap.Error(err))
		return
	}
	svc.logger.Info("Subscribed to Redis Event Store for sinker")
}

func (svc *maestroService) subscribeToSinksES(ctx context.Context) {
	if err := svc.eventStore.SubscribeSinks(ctx); err != nil {
		svc.logger.Error("Bootstrap service failed to subscribe to event sourcing sinks", zap.Error(err))
		return
	}
	svc.logger.Info("Subscribed to Redis Event Store for sinks")
}
