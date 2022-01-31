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
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/siddontang/go/log"

	"github.com/gorilla/mux"
	"github.com/nats-io/nats.go"
)

const (
	streamName     = "NETBOX"
	streamSubjects = "NETBOX.*.*"
)

type Publisher struct {
	js     nats.JetStreamContext
	Router *mux.Router
}

func NewPublisher(nc *nats.Conn) (p *Publisher, err error) {
	js, err := nc.JetStream()
	if err != nil {
		return
	}
	p = &Publisher{
		js:     js,
		Router: mux.NewRouter(),
	}
	if err = p.createStream(); err != nil {
		return
	}
	p.Router.HandleFunc("/handler/netbox/webhook", p.webhookHandler).Methods("POST")
	return
}

func (p *Publisher) publish(subj string, data []byte) (err error) {
	log.Debug("publishing new event")
	_, err = p.js.Publish(subj, data)
	return
}

func (p *Publisher) createStream() (err error) {
	stream, _ := p.js.StreamInfo(streamName)
	if stream == nil {
		log.Debugf("creating stream %q and subjects %q", streamName, streamSubjects)
		_, err = p.js.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: []string{streamSubjects},
			MaxAge:   1 * time.Hour,
		})
		if err != nil {
			return
		}
	} else {
		_, err = p.js.UpdateStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: []string{streamSubjects},
			MaxAge:   1 * time.Hour,
		})
		if err != nil {
			return
		}
	}
	return
}

func (p *Publisher) webhookHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	wb := WebhookBody{}

	if err := json.NewDecoder(r.Body).Decode(&wb); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	region := getRegionFromSite(wb.Data.Site.Slug)
	log.Debugf("incoming webhook event: %s, region: %s, device-name: %s, status: %s, role: %s",
		wb.Event, region, wb.Data.Name, wb.Data.Status.Value, wb.Data.Role.Slug)

	data, err := json.Marshal(wb)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	if err = p.publish(fmt.Sprintf("NETBOX.%s.%s", region, wb.Model), data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}

func getRegionFromSite(site string) string {
	r, s := utf8.DecodeLastRuneInString(site)
	if r == utf8.RuneError && (s == 0 || s == 1) {
		s = 0
	}
	return site[:len(site)-s]
}
