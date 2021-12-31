// Copyright (c) 2014 The SurgeMQ Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package message

// ConnackCode is the type representing the return code in the CONNACK message,
// returned after the initial CONNECT message
type ConnackCode uint64

const (
	// Connection accepted
	ConnectionAccepted ConnackCode = iota

	// The Server does not support the level of the MQTT protocol requested by the Client
	ErrInvalidProtocolVersion

	// The Client identifier is correct UTF-8 but not allowed by the server
	ErrIdentifierRejected

	// The Network Connection has been made but the MQTT service is unavailable
	ErrServerUnavailable

	// The data in the client id or user name or password is malformed
	ErrBadClientIdOrUsernameOrPassword

	// AuthServer Login failed
	ErrAuthServerLoginFailed

	// AuthServe Logout failed
	ErrAuthServerLogoutFailed

	// Authorize Parameters Exist To be Empty
	ParamExistTobeEmpty

	// Authorize Timestamp Parameter Error
	ParamTimestampErr

	// Authorize Product's Device Not Active
	ProductDeviceNotActive

	// Authorize Product Or Device Not Exist
	ProductOrDeviceNotExist

	// Authorize Signature Mismatch
	SignatureMismatch
)

const (
	UploadMessageErrJsonIllegal                = "upload msssage failed cause by json illegal"
	UploadMessageErrJsonSchemaCheckFailed      = "upload msssage failed cause by jsonschema check failed"
	UploadMessageErrOtherInternalServiceErrors = "upload msssage failed cause by other internal service errors"
)

// Value returns the value of the ConnackCode, which is just the byte representation
func (this ConnackCode) Value() byte {
	return byte(this)
}

// Desc returns the description of the ConnackCode
func (this ConnackCode) Desc() string {
	switch this {
	case 0:
		return "Connection accepted"
	case 1:
		return "The Server does not support the level of the MQTT protocol requested by the Client"
	case 2:
		return "The Client identifier is correct UTF-8 but not allowed by the server"
	case 3:
		return "The Network Connection has been made but the MQTT service is unavailable"
	case 4:
		return "The data in the user name or password is malformed"
	case 5:
		return "The Client auth failed"
	case 6:
		return "The Client login failed"
	case 7:
		return "The Client logout failed"

	}

	return ""
}

// Valid checks to see if the ConnackCode is valid. Currently valid codes are <= 5
func (this ConnackCode) Valid() bool {
	return this <= 7
}

// Error returns the corresonding error string for the ConnackCode
func (this ConnackCode) Error() string {
	switch this {
	case 0:
		return "Connection accepted"
	case 1:
		return "Connection Refused, unacceptable protocol version"
	case 2:
		return "Connection Refused, identifier rejected"
	case 3:
		return "Connection Refused, Server unavailable"
	case 4:
		return "Connection Refused, bad user name or password"
	case 5:
		return "Connection Refused, not authorized"
	case 6:
		return "Connection login failed"
	case 7:
		return "Connection logout failed"
	}

	return "Unknown error"
}
