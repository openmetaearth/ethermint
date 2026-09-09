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
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/iden3/go-iden3-crypto/v2/poseidon"
	"github.com/stretchr/testify/require"
)

func padBigInt(val *big.Int) []byte {
	b := make([]byte, 32)
	valBytes := val.Bytes()
	copy(b[32-len(valBytes):], valBytes)
	return b
}

func TestPoseidonAddress(t *testing.T) {
	expected := "0x0000000000000000000000000000000000000100"
	require.Equal(t, expected, PoseidonAddress.Hex())
	require.Equal(t, int64(0x100), new(big.Int).SetBytes(PoseidonAddress.Bytes()).Int64())
}

func TestPoseidonContract_Gas(t *testing.T) {
	contract := &PoseidonContract{}

	// 2 inputs = 64 bytes
	gas2 := contract.RequiredGas(make([]byte, 64))
	require.Equal(t, uint64(60+2*40), gas2)

	// 3 inputs = 96 bytes
	gas3 := contract.RequiredGas(make([]byte, 96))
	require.Equal(t, uint64(60+3*40), gas3)

	// 6 inputs = 192 bytes
	gas6 := contract.RequiredGas(make([]byte, 192))
	require.Equal(t, uint64(60+6*40), gas6)
}

func TestPoseidonContract_InputValidation(t *testing.T) {
	contract := &PoseidonContract{}

	// Empty input
	_, err := contract.Run([]byte{})
	require.ErrorIs(t, err, ErrPoseidonInvalidInputLength)

	// Non-multiple of 32 bytes
	_, err = contract.Run(make([]byte, 33))
	require.ErrorIs(t, err, ErrPoseidonInvalidInputLength)

	// More than 16 inputs (17 inputs = 544 bytes)
	_, err = contract.Run(make([]byte, 17*32))
	require.ErrorIs(t, err, ErrPoseidonInputTooLarge)

	// Input scalar exceeds modulus
	invalidScalar := make([]byte, 64)
	copy(invalidScalar[0:32], padBigInt(SnarkScalarField)) // exactly modulus (invalid)
	_, err = contract.Run(invalidScalar)
	require.ErrorIs(t, err, ErrPoseidonInputExceedsModulus)

	// Exceeds modulus (modulus + 1)
	overModulus := new(big.Int).Add(SnarkScalarField, big.NewInt(1))
	copy(invalidScalar[0:32], padBigInt(overModulus))
	_, err = contract.Run(invalidScalar)
	require.ErrorIs(t, err, ErrPoseidonInputExceedsModulus)
}

func TestPoseidonContract_HashComputations(t *testing.T) {
	contract := &PoseidonContract{}

	tests := []struct {
		name   string
		inputs []*big.Int
	}{
		{
			name:   "Single input [1]",
			inputs: []*big.Int{big.NewInt(1)},
		},
		{
			name:   "Two inputs [1, 2]",
			inputs: []*big.Int{big.NewInt(1), big.NewInt(2)},
		},
		{
			name:   "Two inputs [0, 0]",
			inputs: []*big.Int{big.NewInt(0), big.NewInt(0)},
		},
		{
			name:   "Three inputs [100, 200, 300]",
			inputs: []*big.Int{big.NewInt(100), big.NewInt(200), big.NewInt(300)},
		},
		{
			name:   "Six inputs",
			inputs: []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3), big.NewInt(4), big.NewInt(5), big.NewInt(6)},
		},
		{
			name: "Max allowed 16 inputs",
			inputs: func() []*big.Int {
				arr := make([]*big.Int, 16)
				for i := 0; i < 16; i++ {
					arr[i] = big.NewInt(int64(i + 1))
				}
				return arr
			}(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var rawInput []byte
			for _, in := range tc.inputs {
				rawInput = append(rawInput, padBigInt(in)...)
			}

			// Run precompile
			res, err := contract.Run(rawInput)
			require.NoError(t, err)
			require.Len(t, res, 32)

			// Reference calculation from go-iden3-crypto/poseidon
			expectedBI, err := poseidon.Hash(tc.inputs)
			require.NoError(t, err)
			expectedBytes := padBigInt(expectedBI)

			require.Equal(t, hex.EncodeToString(expectedBytes), hex.EncodeToString(res))
		})
	}
}

func TestPoseidonContract_PrecompileRegistration(t *testing.T) {
	cfg := params.AllEthashProtocolChanges // Berlin active

	precompiles := GetPrecompiles(cfg, big.NewInt(100))
	contract, found := precompiles[PoseidonAddress]
	require.True(t, found, "Poseidon precompile at 0x100 must be found in GetPrecompiles")
	require.NotNil(t, contract)

	// Check ActivePrecompiles
	rules := cfg.Rules(big.NewInt(100), false)
	activeAddrs := vm.ActivePrecompiles(rules)
	var inActive bool
	for _, addr := range activeAddrs {
		if addr == PoseidonAddress {
			inActive = true
			break
		}
	}
	require.True(t, inActive, "PoseidonAddress (0x100) must be in ActivePrecompiles")
}

func TestPoseidonContract_EVMStaticCall(t *testing.T) {
	chainCfg := params.AllEthashProtocolChanges
	db := state.NewDatabase(rawdb.NewMemoryDatabase())
	statedb, err := state.New(common.Hash{}, db, nil)
	require.NoError(t, err)

	blockCtx := vm.BlockContext{
		BlockNumber: big.NewInt(1),
		GasLimit:    10000000,
	}
	txCtx := vm.TxContext{
		Origin:   common.HexToAddress("0x1111111111111111111111111111111111111111"),
		GasPrice: big.NewInt(1),
	}

	evmInstance := NewEVM(blockCtx, txCtx, statedb, chainCfg, vm.Config{}, nil)

	// Prepare input: Poseidon([1, 2])
	input := append(padBigInt(big.NewInt(1)), padBigInt(big.NewInt(2))...)
	caller := vm.AccountRef(common.HexToAddress("0x2222222222222222222222222222222222222222"))

	gas := uint64(100000)
	ret, leftOverGas, err := evmInstance.StaticCall(caller, PoseidonAddress, input, gas)
	require.NoError(t, err)
	require.Len(t, ret, 32)

	// Check gas consumed
	expectedGasCost := (&PoseidonContract{}).RequiredGas(input)
	require.Equal(t, gas-expectedGasCost, leftOverGas)

	// Check output
	expectedBI, err := poseidon.Hash([]*big.Int{big.NewInt(1), big.NewInt(2)})
	require.NoError(t, err)
	require.Equal(t, padBigInt(expectedBI), ret)
}
