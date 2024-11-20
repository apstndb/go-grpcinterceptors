// Copyright (c) The go-grpc-middleware Authors.
// Licensed under the Apache License 2.0.

package selectlogging

import (
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
)

func newCommonFields(kind string, c interceptors.CallMeta) logging.Fields {
	return logging.Fields{
		logging.SystemTag[0], logging.SystemTag[1],
		logging.ComponentFieldKey, kind,
		logging.ServiceFieldKey, c.Service,
		logging.MethodFieldKey, c.Method,
		logging.MethodTypeFieldKey, string(c.Typ),
	}
}

// disableCommonLoggingFields returns copy of newCommonFields with disabled fields removed from the following
// default list. The following are the default logging fields:
//   - SystemTag[0]
//   - ComponentFieldKey
//   - ServiceFieldKey
//   - MethodFieldKey
//   - MethodTypeFieldKey
func disableCommonLoggingFields(kind string, c interceptors.CallMeta, disableFields []string) logging.Fields {
	commonFields := newCommonFields(kind, c)
	for _, key := range disableFields {
		commonFields.Delete(key)
	}
	return commonFields
}
