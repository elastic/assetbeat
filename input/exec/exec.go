// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package exec

import (
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
	execOS "os/exec"
	"strings"
)

func Plugin() input.Plugin {
	return input.Plugin{
		Name:       "exec",
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "exec",
		Manager:    stateless.NewInputManager(configure),
	}
}

func configure(cfg *conf.C) (stateless.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	return newExec(config)
}

func newExec(config config) (*execCmd, error) {
	return &execCmd{config}, nil
}

type Config struct {
	Cmd  string `config:"cmd"`
	Args string `config:"args"`
}

func defaultConfig() config {
	return config{
		Config: Config{
			Cmd:  "",
			Args: "",
		},
	}
}

type execCmd struct {
	config
}

type config struct {
	Config `config:",inline"`
}

func (s *execCmd) Name() string { return "exec" }

func (s *execCmd) Test(_ input.TestContext) error {
	return nil
}

func (s *execCmd) Run(ctx input.Context, publisher stateless.Publisher) error {

	cmd := s.config.Config.Cmd
	args := s.config.Config.Args
	log := ctx.Logger.With("exec")

	log.Info("exec run started")
	defer log.Info("exec run stopped")

	argsList := strings.Split(args, " ")
	log.Info("Args list", argsList, len(argsList))
	var out []byte
	var err error
	if len(argsList) == 1 {
		log.Info("SHORT OPTION")
		out, err = execOS.Command(cmd).Output()
		if err != nil {
			log.Error(err)
		}
	} else {
		log.Info("LONG OPTION")
		out, err = execOS.Command(cmd, strings.Split(args, " ")...).Output()
		if err != nil {
			log.Error(err)
		}
	}
	log.Infof("The cmd output is: %s\n", out)

	event := beat.Event{
		Fields: mapstr.M{
			"output": out,
		},
	}
	publisher.Publish(event)
	return nil
}

// TODO: Add metrics
