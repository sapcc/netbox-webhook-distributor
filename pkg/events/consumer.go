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
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sapcc/netbox-webhook-distributor/pkg/config"
	"github.com/siddontang/go/log"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
)

var waitBackoff = wait.Backoff{
	Steps:    50,
	Duration: 50 * time.Millisecond,
	Factor:   1.1,
	Jitter:   0.1,
}

type DispatchError struct {
	StatusCode int
	Err        error
}

func (d *DispatchError) Error() string {
	return d.Err.Error()
}

type Consumer struct {
	js              nats.JetStreamContext
	name            string
	distributionURL string

	netboxWebhooks map[string][]string

	distributionSuccess prometheus.Counter
	distributionErrors  prometheus.Counter
}

func NewConsumer(d config.Distributor, nc *nats.Conn, ctx context.Context) (c *Consumer, err error) {
	log.Debugf("creating new consumer %s", d.Name)
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
	if err = prometheus.Register(c.distributionSuccess); err != nil {
		return
	}
	return c, prometheus.Register(c.distributionErrors)
}

func (c *Consumer) Subscribe(ctx context.Context) {
	for object := range c.netboxWebhooks {
		go c.subscribe(fmt.Sprintf("NETBOX.%s", object), fmt.Sprintf("%s-%s", c.name, object), object, ctx)
	}
}

func (c *Consumer) subscribe(subj, name, object string, ctx context.Context) {
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
		msgs, err := sub.Fetch(1, nats.Context(ctx))
		if err != nil {
			if errors.Is(err, nats.ErrTimeout) {
				return
			}
			continue
		}
	MESSAGES:
		for _, msg := range msgs {
			if err = msg.InProgress(nats.AckWait(6 * time.Second)); err != nil {
				log.Errorf("set msg inProgress error %s", err.Error())
				continue
			}
			wb := WebhookBody{}
			if err = json.Unmarshal(msg.Data, &wb); err != nil {
				log.Errorf("msg data unmarshal error %s", err.Error())
				c.ack(msg)
				continue
			}
			for _, e := range c.netboxWebhooks[object] {
				if e != wb.Event {
					continue
				}
				log.Debugf("dispatching: %s, %s", msg.Subject, c.distributionURL)
				resultErr := retry.OnError(waitBackoff, func(err error) bool {
					return isRetryError(err)
				}, func() error {
					meta, _ := msg.Metadata()
					if meta != nil {
						log.Debugf("retry dispatching: %s, time: %s to %s", msg.Subject, meta.Timestamp, c.distributionURL)
					}
					return c.dispatch(msg.Data)
				})
				if resultErr != nil {
					c.distributionErrors.Inc()
					log.Debugf("error dispatching event: %s ==> %s: error %s", msg.Subject, c.distributionURL, resultErr.Error())
					log.Errorf("done retrying to deliver event to %s. dropping event", c.name)
					c.ack(msg)
					continue MESSAGES
				}
			}
			c.ack(msg)
			c.distributionSuccess.Inc()
		}
	}
}

func (c *Consumer) dispatch(data []byte) (err error) {
	req, err := http.NewRequest("POST", c.distributionURL, bytes.NewBuffer(data))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return &DispatchError{
			StatusCode: resp.StatusCode,
		}
	}
	return
}

func (c *Consumer) ack(msg *nats.Msg) (err error) {
	if err = msg.AckSync(); err != nil {
		log.Errorf("ackSync error: %s", err)
	}
	return
}

// only retry on timeout or server (500 / 503) errors
func isRetryError(err error) bool {
	if err, ok := err.(net.Error); ok && err.Timeout() {
		return true
	}
	if err, ok := err.(*DispatchError); ok {
		if err.StatusCode == http.StatusInternalServerError ||
			err.StatusCode == http.StatusServiceUnavailable {
			return true
		}
	}
	return false
}
