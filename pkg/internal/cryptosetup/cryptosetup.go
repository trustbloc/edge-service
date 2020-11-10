/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cryptosetup

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/google/tink/go/keyset"
	cryptoapi "github.com/hyperledger/aries-framework-go/pkg/crypto"
	"github.com/hyperledger/aries-framework-go/pkg/crypto/tinkcrypto/primitive/composite/keyio"
	"github.com/hyperledger/aries-framework-go/pkg/doc/jose"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/trustbloc/edge-core/pkg/storage"
)

const (
	vcIDEDVIndexName     = "vcID"
	keyIDStoreName       = "keyid"
	hmacKeyIDDBKeyName   = "hmackeyid"
	ecdhesKeyIDDBKeyName = "ecdheskeyid"
)

var errKeySetHandleAssertionFailure = errors.New("unable to assert key handle as a key set handle pointer")

type unmarshalFunc func([]byte, interface{}) error
type newJWEEncryptFunc func(encAlg jose.EncAlg, encType, senderKID string, senderKH *keyset.Handle,
	recipientsPubKeys []*cryptoapi.PublicKey, crypto cryptoapi.Crypto) (*jose.JWEEncrypt, error)

// PrepareJWECrypto prepares necessary JWE crypto data for edge-service operations
func PrepareJWECrypto(keyManager kms.KeyManager, storeProvider storage.Provider, c cryptoapi.Crypto,
	encAlg jose.EncAlg, keyType kms.KeyType) (*jose.JWEEncrypt, *jose.JWEDecrypt, error) {
	keyHandle, err := prepareKeyHandle(storeProvider, keyManager, ecdhesKeyIDDBKeyName, keyType)
	if err != nil {
		return nil, nil, err
	}

	// passing encryption type is hard coded to `composite.DIDCommEncType` since the encrypter only supports
	// Anoncrypt (ECDHES key types)
	jweEncrypter, err := createJWEEncrypter(keyHandle, encAlg, jose.DIDCommEncType,
		json.Unmarshal, jose.NewJWEEncrypt, c)
	if err != nil {
		return nil, nil, err
	}

	jweDecrypter := jose.NewJWEDecrypt(nil, c, keyManager)

	return jweEncrypter, jweDecrypter, nil
}

func createJWEEncrypter(keyHandle *keyset.Handle, encAlg jose.EncAlg, encType string, unmarshal unmarshalFunc,
	newJWEEncrypt newJWEEncryptFunc, crypto cryptoapi.Crypto) (*jose.JWEEncrypt, error) {
	pubKH, err := keyHandle.Public()
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	pubKeyWriter := keyio.NewWriter(buf)

	err = pubKH.WriteWithNoSecrets(pubKeyWriter)
	if err != nil {
		return nil, err
	}

	ecPubKey := new(cryptoapi.PublicKey)

	err = unmarshal(buf.Bytes(), ecPubKey)
	if err != nil {
		return nil, err
	}

	// since this is anoncrypt, sender key is not set here
	jweEncrypter, err := newJWEEncrypt(encAlg, encType, "", nil, []*cryptoapi.PublicKey{ecPubKey},
		crypto)
	if err != nil {
		return nil, err
	}

	return jweEncrypter, nil
}

// PrepareMACCrypto prepares necessary MAC crypto data for edge-service operations
func PrepareMACCrypto(keyManager kms.KeyManager, storeProvider storage.Provider,
	crypto cryptoapi.Crypto, keyType kms.KeyType) (*keyset.Handle, string, error) {
	keyHandle, err := prepareKeyHandle(storeProvider, keyManager, hmacKeyIDDBKeyName, keyType)
	if err != nil {
		return nil, "", err
	}

	vcIDIndexNameMAC, err := crypto.ComputeMAC([]byte(vcIDEDVIndexName), keyHandle)
	if err != nil {
		return nil, "", err
	}

	return keyHandle, base64.URLEncoding.EncodeToString(vcIDIndexNameMAC), nil
}

func prepareKeyHandle(storeProvider storage.Provider, keyManager kms.KeyManager,
	keyIDDBKeyName string, keyType kms.KeyType) (*keyset.Handle, error) {
	keyIDStore, err := prepareKeyIDStore(storeProvider)
	if err != nil {
		return nil, err
	}

	keyIDBytes, err := keyIDStore.Get(keyIDDBKeyName)
	if errors.Is(err, storage.ErrValueNotFound) {
		keyID, keyHandleUntyped, createErr := keyManager.Create(keyType)
		if createErr != nil {
			return nil, createErr
		}

		kh, ok := keyHandleUntyped.(*keyset.Handle)
		if !ok {
			return nil, errKeySetHandleAssertionFailure
		}

		err = keyIDStore.Put(keyIDDBKeyName, []byte(keyID))
		if err != nil {
			// TODO rollback key creation in KMS that was added during keyManager.Create() call above
			return nil, err
		}

		return kh, nil
	} else if err != nil {
		return nil, err
	}

	keyHandleUntyped, getErr := keyManager.Get(string(keyIDBytes))
	if getErr != nil {
		return nil, getErr
	}

	kh, ok := keyHandleUntyped.(*keyset.Handle)
	if !ok {
		return nil, errKeySetHandleAssertionFailure
	}

	return kh, nil
}

func prepareKeyIDStore(storeProvider storage.Provider) (storage.Store, error) {
	err := storeProvider.CreateStore(keyIDStoreName)
	if err != nil {
		if !errors.Is(err, storage.ErrDuplicateStore) {
			return nil, err
		}
	}

	return storeProvider.OpenStore(keyIDStoreName)
}
