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
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/iden3/go-iden3-crypto/v2/poseidon"
)

var (
	// PoseidonAddress is the precompiled contract address for Poseidon Hash (0x100).
	PoseidonAddress = common.HexToAddress("0x0000000000000000000000000000000000000100")

	// SnarkScalarField is the prime order of the BN254 / alt_bn128 scalar field:
	// 21888242871839275222246405745257275088548364400416034343698204186575808495617
	SnarkScalarField, _ = new(big.Int).SetString("21888242871839275222246405745257275088548364400416034343698204186575808495617", 10)

	// ErrPoseidonInvalidInputLength is returned when the input byte slice is empty or not a multiple of 32 bytes.
	ErrPoseidonInvalidInputLength = errors.New("invalid input length: input must be a non-empty multiple of 32 bytes")
	// ErrPoseidonInputTooLarge is returned when the number of 32-byte scalar inputs exceeds 16.
	ErrPoseidonInputTooLarge = errors.New("invalid input: number of inputs exceeds maximum allowed (16)")
	// ErrPoseidonInputExceedsModulus is returned when an input scalar is greater than or equal to the BN254 scalar field modulus.
	ErrPoseidonInputExceedsModulus = errors.New("invalid input: scalar element exceeds BN254 field modulus")
)

const (
	// PoseidonGasBase is the base gas cost for executing Poseidon hash.
	PoseidonGasBase uint64 = 60
	// PoseidonGasPerWord is the gas cost per 32-byte input word.
	PoseidonGasPerWord uint64 = 40
)

// PoseidonContract implements vm.PrecompiledContract for the Poseidon Hash algorithm
// defined over the BN254 scalar field (compatible with iden3 / Circomlib / Polygon ID).
type PoseidonContract struct{}

// RequiredGas calculates the gas required for the Poseidon hash computation.
func (p *PoseidonContract) RequiredGas(input []byte) uint64 {
	numWords := uint64((len(input) + 31) / 32)
	return PoseidonGasBase + numWords*PoseidonGasPerWord
}

// Run executes the Poseidon hash on the provided 32-byte chunked inputs.
func (p *PoseidonContract) Run(input []byte) ([]byte, error) {
	if len(input) == 0 || len(input)%32 != 0 {
		return nil, ErrPoseidonInvalidInputLength
	}

	numInputs := len(input) / 32
	if numInputs < 1 || numInputs > 16 {
		return nil, ErrPoseidonInputTooLarge
	}

	inputs := make([]*big.Int, numInputs)
	for i := 0; i < numInputs; i++ {
		val := new(big.Int).SetBytes(input[i*32 : (i+1)*32])
		if val.Cmp(SnarkScalarField) >= 0 {
			return nil, ErrPoseidonInputExceedsModulus
		}
		inputs[i] = val
	}

	res, err := poseidon.Hash(inputs)
	if err != nil {
		return nil, err
	}

	output := make([]byte, 32)
	resBytes := res.Bytes()
	copy(output[32-len(resBytes):], resBytes)
	return output, nil
}
