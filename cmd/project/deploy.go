/*
 * Flow CLI
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package project

import (
	"github.com/onflow/flow-cli/cmd"
	"github.com/onflow/flow-cli/flow/lib"
	"github.com/onflow/flow-cli/flow/lib/contracts"
	"github.com/onflow/flow-cli/flow/services"
	"github.com/psiemens/sconfig"
	"github.com/spf13/cobra"
)

type flagsDeploy struct {
	Network string `flag:"network" default:"emulator" info:"network configuration to use"`
	Update  bool   `flag:"update" default:"false" info:"use update flag to update existing contracts"`
}

type cmdDeploy struct {
	cmd   *cobra.Command
	flags flagsDeploy
}

// NewDeployCmd creates new deploy command
func NewDeployCmd() cmd.Command {
	return &cmdDeploy{
		cmd: &cobra.Command{
			Use:   "deploy",
			Short: "Deploy Cadence contracts",
		},
	}
}

// Run script command
func (s *cmdDeploy) Run(
	cmd *cobra.Command,
	args []string,
	project *lib.Project,
	services *services.Services,
) (cmd.Result, error) {
	c, err := services.Project.Deploy(s.flags.Network, s.flags.Update)
	return &DeployResult{contracts: c, project: project}, err
}

// GetFlags for script
func (s *cmdDeploy) GetFlags() *sconfig.Config {
	return sconfig.New(&s.flags)
}

// GetCmd get command
func (s *cmdDeploy) GetCmd() *cobra.Command {
	return s.cmd
}

// DeployResult result structure
type DeployResult struct {
	contracts []*contracts.Contract
	project   *lib.Project
}

// JSON convert result to JSON
func (r *DeployResult) JSON() interface{} {
	result := make(map[string]string, 0)

	for _, contract := range r.contracts {
		result[contract.Name()] = contract.Target().String()
	}

	return result
}

// String convert result to string
func (r *DeployResult) String() string {
	return ""
}

// Oneliner show result as one liner grep friendly
func (r *DeployResult) Oneliner() string {
	return ""
}
