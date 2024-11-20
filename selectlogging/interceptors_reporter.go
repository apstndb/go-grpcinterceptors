// Copyright (c) The go-grpc-middleware Authors.
// Licensed under the Apache License 2.0.

package selectlogging

import (
	"context"
	"time"

	"google.golang.org/grpc/peer"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
)

type ClientReportable interface {
	ClientReporter(context.Context, interceptors.CallMeta) (Reporter, context.Context)
}

type ServerReportable interface {
	ServerReporter(context.Context, interceptors.CallMeta) (Reporter, context.Context)
}

type Reporter interface {
	PostCall(err error, rpcDuration time.Duration)
	PostMsgSend(reqProto any, err error, sendDuration time.Duration)
	PostMsgReceive(replyProto any, err error, recvDuration time.Duration)
}

type report struct {
	callMeta  interceptors.CallMeta
	startTime time.Time
}

func newReport(callMeta interceptors.CallMeta) report {
	r := report{
		startTime: time.Now(),
		callMeta:  callMeta,
	}
	return r
}

func reportable(logger logging.Logger, opts *options) interceptors.CommonReportableFunc {
	return func(ctx context.Context, c interceptors.CallMeta) (interceptors.Reporter, context.Context) {
		kind := logging.KindServerFieldValue
		if c.IsClient {
			kind = logging.KindClientFieldValue
		}

		// Field dups from context override the common fields.
		fields := newCommonFields(kind, c)
		if opts.disableGrpcLogFields != nil {
			fields = disableCommonLoggingFields(kind, c, opts.disableGrpcLogFields)
		}
		fields = fields.WithUnique(logging.ExtractFields(ctx))

		if !c.IsClient {
			if peer, ok := peer.FromContext(ctx); ok {
				fields = append(fields, "peer.address", peer.Addr.String())
			}
		}
		if opts.fieldsFromCtxCallMetaFn != nil {
			// fieldsFromCtxFn dups override the existing fields.
			fields = opts.fieldsFromCtxCallMetaFn(ctx, c).AppendUnique(fields)
		}

		singleUseFields := logging.Fields{"grpc.start_time", time.Now().Format(opts.timestampFormat)}
		if d, ok := ctx.Deadline(); ok {
			singleUseFields = singleUseFields.AppendUnique(logging.Fields{"grpc.request.deadline", d.Format(opts.timestampFormat)})
		}
		return &reporter{
			CallMeta:        c,
			ctx:             ctx,
			startCallLogged: false,
			opts:            opts,
			fields:          fields.WithUnique(singleUseFields),
			logger:          logger,
			kind:            kind,
		}, logging.InjectFields(ctx, fields)
	}
}
