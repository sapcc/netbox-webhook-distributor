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
	"github.com/gorilla/mux"
)

type WebhookBody struct {
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

type Webhook struct {
	Router    *mux.Router
	Publisher *Publisher
}
