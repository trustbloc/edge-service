// Copyright SecureKey Technologies Inc. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

module github.com/trustbloc/edge-service/cmd/did-rest

require (
	github.com/gorilla/mux v1.7.4
	github.com/hyperledger/aries-framework-go v0.1.4
	github.com/rs/cors v1.7.0
	github.com/spf13/cobra v0.0.6
	github.com/stretchr/testify v1.6.1
	github.com/trustbloc/edge-core v0.1.5-0.20201016154334-cc7dc2f523b0
	github.com/trustbloc/edge-service v0.0.0
)

replace github.com/trustbloc/edge-service => ../..

go 1.15
