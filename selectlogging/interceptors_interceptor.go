package selectlogging

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/selector"
	"google.golang.org/grpc"
)

// StreamClientInterceptor is a gRPC client-side interceptor that provides reporting for Stream RPCs.
func StreamClientInterceptor(logger logging.Logger, m selector.Matcher, opts ...Option) grpc.StreamClientInterceptor {
	o := evaluateClientOpt(opts)
	reporter := reportable(logger, o)
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		r := newReport(interceptors.NewClientCallMeta(method, desc, nil))
		reporter, newCtx := reporter.ClientReporter(ctx, r.callMeta)

		rawCs, err := streamer(newCtx, desc, cc, method, opts...)
		if err != nil {
			return nil, err
		}
		return &monitoredClientStream{startTime: r.startTime, desc: desc, method: method, ClientStream: rawCs, matcher: m, reporter: reporter}, nil
	}
}

// monitoredClientStream wraps grpc.ClientStream allowing each Sent/Recv of message to report.
// Note: modified version to support Matcher in SendMsg/RecvMsg
type monitoredClientStream struct {
	grpc.ClientStream

	reporter  Reporter
	startTime time.Time

	method  string
	desc    *grpc.StreamDesc
	matcher selector.Matcher
	ignore  bool
}

func (s *monitoredClientStream) SendMsg(m any) error {
	cm := interceptors.NewClientCallMeta(s.method, s.desc, m)

	start := time.Now()
	err := s.ClientStream.SendMsg(m)
	if s.matcher.Match(s.Context(), cm) {
		s.reporter.PostMsgSend(m, err, time.Since(start))
	} else {
		s.ignore = true
	}
	return err
}

func (s *monitoredClientStream) RecvMsg(m any) error {
	start := time.Now()
	err := s.ClientStream.RecvMsg(m)
	if !s.ignore {
		s.reporter.PostMsgReceive(m, err, time.Since(start))
	}

	if err == nil {
		return nil
	}
	var postErr error
	if errors.Is(err, io.EOF) {
		postErr = err
	}
	if !s.ignore {
		s.reporter.PostCall(postErr, time.Since(s.startTime))
	}
	return err
}
