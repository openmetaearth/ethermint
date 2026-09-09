// Copyright 2021 Evmos Foundation
// This file is part of Evmos' Ethermint library.
//
// The Ethermint library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Ethermint library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Ethermint library. If not, see https://github.com/evmos/ethermint/blob/main/LICENSE
package geth

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"

	evm "github.com/evmos/ethermint/x/evm/vm"
)

func init() {
	poseidonContract := &PoseidonContract{}

	// Register Poseidon (0x100) in the contract maps
	if vm.PrecompiledContractsBerlin != nil {
		vm.PrecompiledContractsBerlin[PoseidonAddress] = poseidonContract
	}
	if vm.PrecompiledContractsIstanbul != nil {
		vm.PrecompiledContractsIstanbul[PoseidonAddress] = poseidonContract
	}
	if vm.PrecompiledContractsByzantium != nil {
		vm.PrecompiledContractsByzantium[PoseidonAddress] = poseidonContract
	}

	// Register in the active precompile address slices if not already present
	registerPrecompileAddress := func(addresses *[]common.Address, addr common.Address) {
		for _, a := range *addresses {
			if a == addr {
				return
			}
		}
		*addresses = append(*addresses, addr)
	}

	registerPrecompileAddress(&vm.PrecompiledAddressesBerlin, PoseidonAddress)
	registerPrecompileAddress(&vm.PrecompiledAddressesIstanbul, PoseidonAddress)
	registerPrecompileAddress(&vm.PrecompiledAddressesByzantium, PoseidonAddress)
}

// GetPrecompiles returns all the precompiled contracts defined given the
// current chain configuration and block height.
func GetPrecompiles(cfg *params.ChainConfig, blockNumber *big.Int) evm.PrecompiledContracts {
	var precompiles evm.PrecompiledContracts
	switch {
	case cfg.IsBerlin(blockNumber):
		precompiles = vm.PrecompiledContractsBerlin
	case cfg.IsIstanbul(blockNumber):
		precompiles = vm.PrecompiledContractsIstanbul
	case cfg.IsByzantium(blockNumber):
		precompiles = vm.PrecompiledContractsByzantium
	default:
		precompiles = vm.PrecompiledContractsHomestead
	}
	return precompiles
}
