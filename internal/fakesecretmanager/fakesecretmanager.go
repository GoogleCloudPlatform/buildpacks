// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package fakesecretmanager provides stubs for secret manager service.
package fakesecretmanager

import (
	"context"
	"fmt"

	"github.com/googleapis/gax-go/v2"
	smpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

// FakeSecretClient is a fake of the SecretClient.
type FakeSecretClient struct {
	SecretVersionResponses       map[string]GetSecretVersionResponse
	AccessSecretVersionResponses map[string]AccessSecretVersionResponse
}

// GetSecretVersionResponse is a wrapper for secret manager service GetSecretVersion api
// response.
type GetSecretVersionResponse struct {
	SecretVersion *smpb.SecretVersion
	Error         error
}

// AccessSecretVersionResponse is a wrapper for secret manager service AccessSecretVersion
// api response.
type AccessSecretVersionResponse struct {
	Response *smpb.AccessSecretVersionResponse
	Error    error
}

// GetSecretVersion gets the secret version for the given request.
func (s *FakeSecretClient) GetSecretVersion(ctx context.Context, req *smpb.GetSecretVersionRequest, opts ...gax.CallOption) (*smpb.SecretVersion, error) {
	resp, ok := s.SecretVersionResponses[req.GetName()]
	if !ok || resp.SecretVersion == nil {
		return nil, fmt.Errorf("fake client secret version is not found for %s", req.GetName())
	}

	return resp.SecretVersion, nil
}

// AccessSecretVersion accesses the secret material for the given request.
func (s *FakeSecretClient) AccessSecretVersion(ctx context.Context, req *smpb.AccessSecretVersionRequest, opts ...gax.CallOption) (*smpb.AccessSecretVersionResponse, error) {
	resp, ok := s.AccessSecretVersionResponses[req.GetName()]
	if !ok || resp.Response == nil {
		return nil, fmt.Errorf("fake client secret version is not found for %s", req.GetName())
	}

	return resp.Response, nil
}
