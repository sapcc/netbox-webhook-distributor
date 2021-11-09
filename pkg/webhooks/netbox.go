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

package webhooks

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sapcc/netbox-webhook-distributor/pkg/events"
	"github.com/siddontang/go/log"
)

type webhookBody struct {
	Event     string
	Timestamp string
	Model     string
	Username  string
	Data      data
	Snapshots snapshot `json:"snapshots"`
}

type data struct {
	ID       int
	Name     string
	Region   string
	Role     role `json:"device_role"`
	Status   status
	Comments string
	Site     site
}

type role struct {
	Display string
	Slug    string
	ID      int `json:"id"`
}

type status struct {
	Value string
	Label string
}

type site struct {
	ID   int    `json:"id"`
	Slug string `json:"slug"`
}

type snapshot struct {
	PreChange  change `json:"prechange"`
	PostChange change `json:"postchange"`
}

type change struct {
	Status string `json:"status"`
}

// Netbox for http requests
type Netbox struct {
	Router    *mux.Router
	Publisher *events.Publisher
	//cfg    config.Config
}

// New http handler
func NewNetbox(p *events.Publisher) *Netbox {
	h := Netbox{mux.NewRouter(), p}
	h.Router.HandleFunc("/handler/netbox/webhook", h.webhookHandler).Methods("POST")
	return &h
}

func (h *Netbox) webhookHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	wb := webhookBody{}
	if err := json.NewDecoder(r.Body).Decode(&wb); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	log.Debugf("incoming webhook event: %s, region: %s, device-name: %s, status: %s, role: %s",
		wb.Event, wb.Data.Site.Slug, wb.Data.Name, wb.Data.Status.Value, wb.Data.Role.Slug)

	data, err := json.Marshal(wb)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	if err = h.Publisher.Publish(fmt.Sprintf("NETBOX.%s.%s", wb.Model, wb.Event), data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}
