/*
 * Flow CLI
 *
 * Copyright 2019 Dapper Labs, Inc.
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

package resolvers

import (
	"fmt"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/flow-cli/pkg/flowkit"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

type deployContract struct {
	index int64
	*flowkit.Contract
	program      *ast.Program
	dependencies map[string]*deployContract
}

func newContract(contract *flowkit.Contract, index int64, code []byte) (*deployContract, error) {
	program, err := parser.ParseProgram(code, nil)
	if err != nil {
		return nil, err
	}

	return &deployContract{
		index:    index,
		Contract: contract,
		program:  program,
	}, nil
}

func (d *deployContract) ID() int64 {
	return d.index
}

// imports returns a list of all imports found in contract.
// todo use a resolver to find imports not direct implementation since imports can be in different formats
func (d *deployContract) imports() []string {
	imports := make([]string, 0)

	for _, imp := range d.program.ImportDeclarations() {
		location, ok := imp.Location.(common.StringLocation)
		if ok {
			imports = append(imports, location.String())
		}
	}

	return imports
}

func (d *deployContract) addDependency(location string, dep *deployContract) {
	d.dependencies[location] = dep
}

// Deployment contains logic to sort deployment order of contracts.
//
// Deployment makes sure the contract containing imports is deployed after all importing contracts are deployed.
// This way we can deploy all contracts without missing imports.
// Contracts are iterated and dependency graph is built which is then later sorted
type Deployment struct {
	contracts []*deployContract
	// map of contracts by their location specified in state
	contractsByLocation map[string]*deployContract
	loader              Loader
}

// NewDeployment from the flowkit Contracts and loaded from the contract location using a loader.
func NewDeployment(contracts []*flowkit.Contract, loader Loader) (*Deployment, error) {
	deployment := &Deployment{
		loader:              loader,
		contractsByLocation: make(map[string]*deployContract),
	}

	for _, contract := range contracts {
		err := deployment.add(contract)
		if err != nil {
			return nil, err
		}
	}

	return deployment, nil
}

func (d *Deployment) add(contract *flowkit.Contract) error {
	// TODO implement group of loaders detecting the location format and choosing the one supporting that format to load the contract - this will be relevant for multiple locations like flow, ifps, github etc
	code, err := d.loader.Load(contract.Location)
	if err != nil {
		return err
	}

	c, err := newContract(contract, int64(len(d.contracts)), code)
	if err != nil {
		return err
	}

	d.contracts = append(d.contracts, c)
	d.contractsByLocation[c.Location] = c

	return nil
}

// Sort contracts by deployment order.
//
// Order of sorting is dependent on the possible imports contract contains, since
// any imported contract must be deployed before deploying the contract with that import.
// Only applicable to contracts.
func (d *Deployment) Sort() ([]*flowkit.Contract, error) {
	err := d.buildDependencies()
	if err != nil {
		return nil, err
	}

	sorted, err := sortByDeploymentOrder(d.contracts)
	if err != nil {
		return nil, err
	}

	contracts := make([]*flowkit.Contract, len(d.contracts))
	for i, s := range sorted {
		contracts[i] = s.Contract
	}

	return contracts, nil
}

// buildDependencies iterates over all contracts and checks the imports which are added as its dependencies.
func (d *Deployment) buildDependencies() error {
	for _, contract := range d.contracts {
		for _, location := range contract.imports() {
			importPath := location // TODO: i.loader.Normalize(program.source, source)
			importContract, isContract := d.contractsByLocation[importPath]
			// todo is it we removed aliases here?
			if !isContract {
				return fmt.Errorf(
					"import from %s could not be found: %s, make sure import path is correct",
					contract.Name,
					importPath,
				)

			}

			contract.addDependency(location, importContract)
		}
	}

	return nil
}

// sortByDeploymentOrder sorts the given set of contracts in order of deployment.
//
// The resulting ordering ensures that each contract is deployed after all of its
// dependencies are deployed. This function returns an error if an import cycle exists.
//
// This function constructs a directed graph in which contracts are nodes and imports are edges.
// The ordering is computed by performing a topological sort on the constructed graph.
func sortByDeploymentOrder(contracts []*deployContract) ([]*deployContract, error) {
	g := simple.NewDirectedGraph()

	for _, c := range contracts {
		g.AddNode(c)
	}

	for _, c := range contracts {
		for _, dep := range c.dependencies {
			g.SetEdge(g.NewEdge(dep, c))
		}
	}

	sorted, err := topo.SortStabilized(g, nil)
	if err != nil {
		switch topoErr := err.(type) {
		case topo.Unorderable:
			return nil, &CyclicImportError{Cycles: nodeSetsToContractSets(topoErr)}
		default:
			return nil, err
		}
	}

	return nodesToContracts(sorted), nil
}

func nodeSetsToContractSets(nodes [][]graph.Node) [][]*deployContract {
	contracts := make([][]*deployContract, len(nodes))

	for i, s := range nodes {
		contracts[i] = nodesToContracts(s)
	}

	return contracts
}

func nodesToContracts(nodes []graph.Node) []*deployContract {
	contracts := make([]*deployContract, len(nodes))

	for i, s := range nodes {
		contracts[i] = s.(*deployContract)
	}

	return contracts
}

// CyclicImportError is returned when contract contain cyclic imports one to the
// other which is not possible to be resolved and deployed.
type CyclicImportError struct {
	Cycles [][]*deployContract
}

func (e *CyclicImportError) contractNames() [][]string {
	cycles := make([][]string, 0, len(e.Cycles))

	for _, cycle := range e.Cycles {
		contracts := make([]string, 0, len(cycle))
		for _, contract := range cycle {
			contracts = append(contracts, contract.Name)
		}

		cycles = append(cycles, contracts)
	}

	return cycles
}

func (e *CyclicImportError) Error() string {
	return fmt.Sprintf(
		"contracts: import cycle(s) detected: %v",
		e.contractNames(),
	)
}
