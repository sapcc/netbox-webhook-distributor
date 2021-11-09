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

package events

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sapcc/netbox-webhook-distributor/pkg/config"
)

type Consumer struct {
	js              nats.JetStreamContext
	name            string
	distributionURL string

	netboxWebhooks map[string][]string

	distributionSuccess prometheus.Counter
	distributionErrors  prometheus.Counter
}

func NewConsumer(d config.Distributor, nc *nats.Conn, ctx context.Context) (c *Consumer, err error) {
	js, err := nc.JetStream()
	if err != nil {
		return
	}
	c = &Consumer{
		name:            d.Name,
		distributionURL: d.URL,
		js:              js,
		distributionSuccess: prometheus.NewCounter(prometheus.CounterOpts{
			Subsystem:   "distribution",
			Name:        "success_total",
			Help:        "Total number of successfully distributed webhooks",
			ConstLabels: prometheus.Labels{"consumer": d.Name},
		}),
		distributionErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Subsystem:   "distribution",
			Name:        "errors_total",
			Help:        "Total number of successfully distributed webhooks",
			ConstLabels: prometheus.Labels{"consumer": d.Name},
		}),
		netboxWebhooks: d.NetboxWebhooks,
	}
	prometheus.Register(c.distributionSuccess)
	prometheus.Register(c.distributionErrors)
	return
}

func (c *Consumer) Subscribe(ctx context.Context) {
	for object, events := range c.netboxWebhooks {
		for _, e := range events {
			go c.subscribe(fmt.Sprintf("NETBOX.%s.%s", object, e), fmt.Sprintf("%s-%s-%s", c.name, object, e), ctx)
		}
	}
}

func (c *Consumer) subscribe(subj string, name string, ctx context.Context) {
	sub, err := c.js.PullSubscribe(subj, name, nats.PullMaxWaiting(128))
	if err != nil {
		log.Fatal(err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		msgs, err := sub.Fetch(10, nats.Context(ctx))
		if err != nil {
			continue
		}
		for _, msg := range msgs {
			msg.Ack()
			fmt.Println(string(msg.Data))
			if err = c.dispatch(msg.Data); err != nil {
				c.distributionErrors.Inc()
				log.Printf("Error dispatching event: %s ==> %s", msg.Subject, c.distributionURL)
				continue
			}
			c.distributionSuccess.Inc()
		}
	}
}

func (c *Consumer) dispatch(data []byte) (err error) {
	req, err := http.NewRequest("POST", c.distributionURL, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("msg distribution not successful: http status-code %d", resp.StatusCode)
	}
	return
}
