/*
 * Copyright (C) 2019 The TesraSupernet Authors
 * This file is part of The TesraSupernet library.
 *
 * The TesraSupernet is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or * (at your option) any later version.
 *
 * The TesraSupernet is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The TesraSupernet.  If not, see <http://www.gnu.org/licenses/>.
 */
package tstid

import (
	"encoding/hex"
	"errors"

	"github.com/TesraSupernet/Tesra/common"
	"github.com/TesraSupernet/Tesra/core/states"
	"github.com/TesraSupernet/Tesra/core/types"
	"github.com/TesraSupernet/Tesra/smartcontract/service/native"
	"github.com/TesraSupernet/Tesra/smartcontract/service/native/utils"
	"github.com/TesraSupernet/tesracrypto/keypair"
)

func isValid(srvc *native.NativeService, encID []byte) bool {
	return checkIDState(srvc, encID) == flag_valid
}

func checkIDState(srvc *native.NativeService, encID []byte) byte {
	val, err := srvc.CacheDB.Get(encID)
	if err == nil {
		val, err := states.GetValueFromRawStorageItem(val)
		if err == nil {
			if len(val) > 0 {
				return val[0]
			}
		}
	}
	return flag_not_exist
}

const (
	flag_not_exist byte = 0x00
	flag_valid     byte = 0x01
	flag_revoke    byte = 0x02

	FIELD_VERSION byte = 0
	FLAG_VERSION  byte = 0x01

	FIELD_PK         byte = 1
	FIELD_ATTR       byte = 2
	FIELD_RECOVERY   byte = 3
	FIELD_CONTROLLER byte = 4
)

func encodeID(id []byte) ([]byte, error) {
	length := len(id)
	if length == 0 || length > 255 {
		return nil, errors.New("encode TST ID error: invalid ID length")
	}
	//enc := []byte{byte(length)}
	enc := append(utils.TstIDContractAddress[:], byte(length))
	enc = append(enc, id...)
	return enc, nil
}

func decodeID(data []byte) ([]byte, error) {
	prefix := len(utils.TstIDContractAddress)
	size := len(data)
	if size < prefix || size != int(data[prefix])+1+prefix {
		return nil, errors.New("decode TST ID error: invalid data length")
	}
	return data[prefix+1:], nil
}

func checkWitness(srvc *native.NativeService, key []byte) error {
	// try as if key is a public key
	pk, err := keypair.DeserializePublicKey(key)
	if err == nil {
		addr := types.AddressFromPubKey(pk)
		if srvc.ContextRef.CheckWitness(addr) {
			return nil
		}
	}

	// try as if key is an address
	addr, err := common.AddressParseFromBytes(key)
	if srvc.ContextRef.CheckWitness(addr) {
		return nil
	}

	return errors.New("check witness failed, " + hex.EncodeToString(key))
}

func deleteID(srvc *native.NativeService, encID []byte) error {
	key := append(encID, FIELD_PK)
	srvc.CacheDB.Delete(key)

	key = append(encID, FIELD_CONTROLLER)
	srvc.CacheDB.Delete(key)

	key = append(encID, FIELD_RECOVERY)
	srvc.CacheDB.Delete(key)

	err := deleteAllAttr(srvc, encID)
	if err != nil {
		return err
	}

	//set flag to revoke
	utils.PutBytes(srvc, encID, []byte{flag_revoke})
	return nil
}