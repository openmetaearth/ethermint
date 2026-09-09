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
package keeper_test

import (
	"math/big"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/evmos/ethermint/x/evm/vm/geth"
	"github.com/iden3/go-iden3-crypto/v2/poseidon"
)

func padBigInt32(val *big.Int) []byte {
	b := make([]byte, 32)
	valBytes := val.Bytes()
	copy(b[32-len(valBytes):], valBytes)
	return b
}

func (suite *KeeperTestSuite) TestPoseidonPrecompile_ApplyMessage() {
	suite.SetupTest()

	poseidonAddr := geth.PoseidonAddress

	// 1. Prepare valid message: Poseidon([1, 2])
	in1 := big.NewInt(1)
	in2 := big.NewInt(2)
	data := append(padBigInt32(in1), padBigInt32(in2)...)

	msg := ethtypes.NewMessage(
		suite.address,
		&poseidonAddr,
		0,
		big.NewInt(0),
		100000,
		big.NewInt(1),
		big.NewInt(1),
		big.NewInt(0),
		data,
		ethtypes.AccessList{},
		false,
	)

	res, err := suite.app.EvmKeeper.ApplyMessage(suite.ctx, msg, nil, true)
	suite.Require().NoError(err)
	suite.Require().False(res.Failed(), "Poseidon precompile execution should succeed")

	expectedBI, err := poseidon.Hash([]*big.Int{in1, in2})
	suite.Require().NoError(err)
	expectedBytes := padBigInt32(expectedBI)

	suite.Require().Equal(expectedBytes, res.Ret, "Return value must match Poseidon([1, 2]) hash")
	suite.Require().Greater(res.GasUsed, uint64(21000), "Gas used should include intrinsic gas + precompile gas")

	// 2. Prepare invalid message: len(data) % 32 != 0
	invalidMsg := ethtypes.NewMessage(
		suite.address,
		&poseidonAddr,
		1,
		big.NewInt(0),
		100000,
		big.NewInt(1),
		big.NewInt(1),
		big.NewInt(0),
		make([]byte, 33),
		ethtypes.AccessList{},
		false,
	)

	resInvalid, err := suite.app.EvmKeeper.ApplyMessage(suite.ctx, invalidMsg, nil, true)
	suite.Require().NoError(err)
	suite.Require().True(resInvalid.Failed(), "Invalid input length should fail EVM execution")
	suite.Require().Equal(geth.ErrPoseidonInvalidInputLength.Error(), resInvalid.VmError)

	// 3. Prepare invalid message: scalar exceeds BN254 modulus
	invalidScalar := make([]byte, 64)
	copy(invalidScalar[0:32], padBigInt32(geth.SnarkScalarField))
	invalidModMsg := ethtypes.NewMessage(
		suite.address,
		&poseidonAddr,
		2,
		big.NewInt(0),
		100000,
		big.NewInt(1),
		big.NewInt(1),
		big.NewInt(0),
		invalidScalar,
		ethtypes.AccessList{},
		false,
	)

	resMod, err := suite.app.EvmKeeper.ApplyMessage(suite.ctx, invalidModMsg, nil, true)
	suite.Require().NoError(err)
	suite.Require().True(resMod.Failed(), "Scalar >= field modulus should fail EVM execution")
	suite.Require().Equal(geth.ErrPoseidonInputExceedsModulus.Error(), resMod.VmError)
}
