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

package config

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	DistributorList []Distributor `yaml:"distributor_list"`
}

type Distributor struct {
	Name           string              `yaml:"name"`
	URL            string              `yaml:"url"`
	Region         string              `yaml:"region"`
	NetboxWebhooks map[string][]string `yaml:"netbox_webhooks"`
}

func GetConfig(opts Options) (cfg Config, err error) {
	if opts.ConfigFilePath == "" {
		return cfg, nil
	}
	yamlBytes, err := ioutil.ReadFile(opts.ConfigFilePath)
	if err != nil {
		return cfg, fmt.Errorf("read file file: %s", err.Error())
	}
	err = yaml.Unmarshal(yamlBytes, &cfg)
	if err != nil {
		return cfg, fmt.Errorf("parse config file: %s", err.Error())
	}

	return cfg, nil
}
