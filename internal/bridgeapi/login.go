/*
Copyright 2021 Crunchy Data Solutions, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package bridgeapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// TODO: move login manager from package global to client internal
var primaryLogin *loginManager

type loginManager struct {
	activeToken   string
	activeTokenID string
	apiTarget     *url.URL
	loginSource   CredentialProvider
}

func newLoginManager(cp CredentialProvider, target *url.URL) (*loginManager, error) {
	lm := &loginManager{
		loginSource: cp,
		apiTarget:   target,
	}
	if err := lm.login(); err != nil {
		return lm, err
	}

	return lm, nil
}

func (lm *loginManager) login() error {
	creds, err := lm.loginSource.ProvideCredential()
	if err != nil {
		pkgLog.Error(err, "error retrieving credentials")
		return err
	}

	req, err := http.NewRequest(http.MethodPost, lm.apiTarget.String()+"/token", nil)
	if err != nil {
		pkgLog.Error(err, "error creating token login request")
		return err
	}
	req.SetBasicAuth(creds.Key, creds.Secret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		pkgLog.Error(err, "error creating http client")
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
		pkgLog.Error(fmt.Errorf("API returned status %d for login [%s]", resp.StatusCode, creds.Key), "login failure")
		return fmt.Errorf("API returned status %d for login [%s]", resp.StatusCode, creds.Key)
	} else if resp.StatusCode != http.StatusOK {
		pkgLog.Error(
			fmt.Errorf("API returned unexpected response %d for login [%s]", resp.StatusCode, creds.Key),
			"unexpected login response")
		return fmt.Errorf("API returned unexpected response %d for login [%s]", resp.StatusCode, creds.Key)
	}

	var tr tokenResponse
	err = json.NewDecoder(resp.Body).Decode(&tr)
	if err != nil {
		pkgLog.Error(err, "error unmarshaling token response body")
		return err
	}

	lm.activeToken = tr.Token
	lm.activeTokenID = tr.TokenID

	return nil
}

func (lm *loginManager) UpdateLogin(cp CredentialProvider) {
	lm.loginSource = cp
}

func (lm *loginManager) UpdateAuthURL(target *url.URL) {
	lm.apiTarget = target
}

func SetLogin(cp CredentialProvider, authBaseURL *url.URL) error {
	var err error = nil
	primaryLogin, err = newLoginManager(cp, authBaseURL)
	if err != nil {
		return err
	}
	return nil
}

type tokenResponse struct {
	Token     string `json:"access_token"`
	ExpiresIn int64  `json:"expires_in"`
	TokenID   string `json:"id"`
}
