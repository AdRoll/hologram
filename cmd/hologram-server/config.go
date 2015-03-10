// Copyright 2014 AdRoll, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

type Config struct {
	LDAP struct {
		Bind struct {
			DN       string `json:"dn"`
			Password string `json:"password"`
		} `json:"bind"`
		BaseDN       string `json:"basedn"`
		Host         string `json:"host"`
		InsecureLDAP bool   `json:"insecureldap"`
	} `json:"ldap"`
	AWS struct {
		Account     string `json:"account"`
		DefaultRole string `json:"defaultrole"`
	} `json:"aws"`
	Stats  string `json:"stats"`
	Listen string `json:"listen"`
}
