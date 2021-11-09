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
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	streamName     = "NETBOX"
	streamSubjects = "NETBOX.*.*"
)

type Publisher struct {
	js nats.JetStreamContext
}

func NewPublisher(nc *nats.Conn) (p *Publisher, err error) {
	js, err := nc.JetStream()
	if err != nil {
		return
	}
	p = &Publisher{
		js: js,
	}

	if err = p.createStream(); err != nil {
		return
	}
	return
}

func (p *Publisher) Publish(subj string, data []byte) (err error) {
	_, err = p.js.Publish(subj, data)
	return
}

func (p *Publisher) createStream() (err error) {
	stream, _ := p.js.StreamInfo(streamName)
	if stream == nil {
		log.Printf("creating stream %q and subjects %q", streamName, streamSubjects)
		_, err = p.js.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: []string{streamSubjects},
			MaxAge:   24 * time.Hour,
		})
		if err != nil {
			return
		}
	}
	return
}
