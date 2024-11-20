package selectlogging

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	"google.golang.org/protobuf/proto"
)

// Need to copy because it is not exported.
type reporter struct {
	interceptors.CallMeta

	ctx             context.Context
	kind            string
	startCallLogged bool

	opts   *options
	fields logging.Fields
	logger logging.Logger
}

func (c *reporter) PostCall(err error, duration time.Duration) {
	if !has(c.opts.loggableEvents, FinishCall) {
		return
	}
	if errors.Is(err, io.EOF) {
		err = nil
	}

	code := c.opts.codeFunc(err)
	fields := c.fields.WithUnique(logging.ExtractFields(c.ctx))
	fields = fields.AppendUnique(logging.Fields{"grpc.code", code.String()})
	if err != nil {
		fields = fields.AppendUnique(logging.Fields{"grpc.error", fmt.Sprintf("%v", err)})
	}
	c.logger.Log(c.ctx, c.opts.levelFunc(code), "finished call", fields.AppendUnique(c.opts.durationFieldFunc(duration))...)
}

func (c *reporter) PostMsgSend(payload any, err error, duration time.Duration) {
	logLvl := c.opts.levelFunc(c.opts.codeFunc(err))
	fields := c.fields.WithUnique(logging.ExtractFields(c.ctx))
	if err != nil {
		fields = fields.AppendUnique(logging.Fields{"grpc.error", fmt.Sprintf("%v", err)})
	}
	if !c.startCallLogged && has(c.opts.loggableEvents, StartCall) {
		c.startCallLogged = true
		c.logger.Log(c.ctx, logLvl, "started call", fields.AppendUnique(c.opts.durationFieldFunc(duration))...)
	}

	if err != nil || !has(c.opts.loggableEvents, PayloadSent) {
		return
	}
	if c.CallMeta.IsClient {
		p, ok := payload.(proto.Message)
		if !ok {
			c.logger.Log(
				c.ctx,
				logging.LevelError,
				"payload is not a google.golang.org/protobuf/proto.Message; programmatic error?",
				fields.AppendUnique(logging.Fields{"grpc.request.type", fmt.Sprintf("%T", payload)})...,
			)
			return
		}

		fields = fields.AppendUnique(logging.Fields{"grpc.send.duration", duration.String(), "grpc.request.content", p})
		fields = fields.AppendUnique(c.opts.durationFieldFunc(duration))
		c.logger.Log(c.ctx, logLvl, "request sent", fields...)
	} else {
		p, ok := payload.(proto.Message)
		if !ok {
			c.logger.Log(
				c.ctx,
				logging.LevelError,
				"payload is not a google.golang.org/protobuf/proto.Message; programmatic error?",
				fields.AppendUnique(logging.Fields{"grpc.response.type", fmt.Sprintf("%T", payload)})...,
			)
			return
		}

		fields = fields.AppendUnique(logging.Fields{"grpc.send.duration", duration.String(), "grpc.response.content", p})
		fields = fields.AppendUnique(c.opts.durationFieldFunc(duration))
		c.logger.Log(c.ctx, logLvl, "response sent", fields...)
	}
}

func (c *reporter) PostMsgReceive(payload any, err error, duration time.Duration) {
	logLvl := c.opts.levelFunc(c.opts.codeFunc(err))
	fields := c.fields.WithUnique(logging.ExtractFields(c.ctx))
	if err != nil {
		fields = fields.AppendUnique(logging.Fields{"grpc.error", fmt.Sprintf("%v", err)})
	}
	if !c.startCallLogged && has(c.opts.loggableEvents, StartCall) {
		c.startCallLogged = true
		c.logger.Log(c.ctx, logLvl, "started call", fields.AppendUnique(c.opts.durationFieldFunc(duration))...)
	}

	if err != nil || !has(c.opts.loggableEvents, PayloadReceived) {
		return
	}
	if !c.CallMeta.IsClient {
		p, ok := payload.(proto.Message)
		if !ok {
			c.logger.Log(
				c.ctx,
				logging.LevelError,
				"payload is not a google.golang.org/protobuf/proto.Message; programmatic error?",
				fields.AppendUnique(logging.Fields{"grpc.request.type", fmt.Sprintf("%T", payload)})...,
			)
			return
		}

		fields = fields.AppendUnique(logging.Fields{"grpc.recv.duration", duration.String(), "grpc.request.content", p})
		fields = fields.AppendUnique(c.opts.durationFieldFunc(duration))
		c.logger.Log(c.ctx, logLvl, "request received", fields...)
	} else {
		p, ok := payload.(proto.Message)
		if !ok {
			c.logger.Log(
				c.ctx,
				logging.LevelError,
				"payload is not a google.golang.org/protobuf/proto.Message; programmatic error?",
				fields.AppendUnique(logging.Fields{"grpc.response.type", fmt.Sprintf("%T", payload)})...,
			)
			return
		}

		fields = fields.AppendUnique(logging.Fields{"grpc.recv.duration", duration.String(), "grpc.response.content", p})
		fields = fields.AppendUnique(c.opts.durationFieldFunc(duration))
		c.logger.Log(c.ctx, logLvl, "response received", fields...)
	}
}
