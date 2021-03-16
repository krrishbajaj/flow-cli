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

package emulator

import (
	"github.com/onflow/flow-cli/flow/config"
	"github.com/onflow/flow-cli/flow/lib"
	"github.com/onflow/flow-cli/flow/lib/keys"
	"github.com/onflow/flow-emulator/cmd/emulator/start"
	"github.com/onflow/flow-go-sdk/crypto"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:              "emulator",
	Short:            "Flow emulator server",
	TraverseChildren: true,
}

func configuredServiceKey(
	_ bool,
	_ crypto.SignatureAlgorithm,
	_ crypto.HashAlgorithm,
) (
	crypto.PrivateKey,
	crypto.SignatureAlgorithm,
	crypto.HashAlgorithm,
) {
	project, err := lib.LoadProject(lib.ConfigPath)
	if err != nil {
		lib.Exitf(1, err.Error())
	}

	serviceAccount, _ := project.EmulatorServiceAccount()

	serviceKeyHex, ok := serviceAccount.DefaultKey().(*keys.HexAccountKey)
	if !ok {
		lib.Exit(1, "Only hexadecimal keys can be used as the emulator service account key.")
	}

	privateKey, err := crypto.DecodePrivateKeyHex(serviceKeyHex.SigAlgo(), serviceKeyHex.PrivateKeyHex())
	if err != nil {
		lib.Exitf(
			1,
			"Invalid private key in \"%s\" emulator configuration",
			config.DefaultEmulatorConfigName,
		)
	}

	return privateKey, serviceKeyHex.SigAlgo(), serviceKeyHex.HashAlgo()
}

func init() {
	Cmd = start.Cmd(configuredServiceKey)
	Cmd.Use = "emulator"
}
