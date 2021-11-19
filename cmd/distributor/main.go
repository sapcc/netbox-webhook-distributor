/**
 * Copyright 2021 SAP SE
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sapcc/netbox-webhook-distributor/pkg/config"
	"github.com/sapcc/netbox-webhook-distributor/pkg/events"
	"github.com/siddontang/go/log"
)

var opts config.Options

func init() {
	flag.StringVar(&opts.ConfigFilePath, "CONFIG_FILE", "./etc/config.yaml", "Path to the config file")
	flag.IntVar(&opts.LogLevel, "LOG_LEVEL", 1, "Log level")
	flag.Parse()
}

func main() {
	log.SetLevel(opts.LogLevel)
	ctx, cancel := context.WithCancel(context.Background())
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}
	nc, err := nats.Connect(natsURL, nats.MaxReconnects(100))
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := config.GetConfig(opts)
	if err != nil {
		log.Fatal(err)
	}
	for _, d := range cfg.DistributorList {
		con, _ := events.NewConsumer(d, nc, ctx)
		con.Subscribe(ctx)
	}

	srv := &http.Server{
		Addr: "0.0.0.0:81",
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		// https://operations.global.cloud.sap/docs/support/playbook/kubernetes/idle_http_keep_alive_timeout.html
		ReadTimeout: time.Second * 61,
		IdleTimeout: time.Second * 61,
		Handler:     promhttp.Handler(),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Error(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c

	cancel()
	log.Info("shutting down")
	os.Exit(0)
}
