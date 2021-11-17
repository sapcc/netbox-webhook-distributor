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
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/sapcc/netbox-webhook-distributor/pkg/events"
	"github.com/sapcc/netbox-webhook-distributor/pkg/webhooks"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatal(err)
	}
	p, err := events.NewPublisher(nc)
	if err != nil {
		log.Fatal(err)
	}
	wh := webhooks.NewNetbox(p)

	srv := &http.Server{
		Addr: "0.0.0.0:80",
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		// https://operations.global.cloud.sap/docs/support/playbook/kubernetes/idle_http_keep_alive_timeout.html
		ReadTimeout: time.Second * 61,
		IdleTimeout: time.Second * 61,
		Handler:     wh.Router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c

	cancel()
	srv.Shutdown(ctx)
	log.Println("shutting down")
	os.Exit(0)
}
