package payload

import (
	"context"
	"fmt"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/cloudwego/kitex/pkg/protocol/bthrift"
	"github.com/cloudwego/kitex/pkg/remote"
	"github.com/cloudwego/kitex/pkg/remote/codec"
	"github.com/cloudwego/kitex/pkg/remote/codec/perrors"
	keThrift "github.com/cloudwego/kitex/pkg/remote/codec/thrift"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/pkg/stats"

	"github.com/arklib/ark/errx"
	"github.com/arklib/ark/serializer"
)

// reference: https://github.com/cloudwego/kitex/blob/main/pkg/remote/codec/thrift/thrift.go

type Codec struct {
	name       string
	serializer serializer.Serializer
}

func NewCodec(name string, serializer serializer.Serializer) Codec {
	return Codec{name: name, serializer: serializer}
}

func (c Codec) Name() string {
	return c.name
}

func (c Codec) Marshal(_ context.Context, message remote.Message, out remote.ByteBuffer) error {
	method := message.RPCInfo().Invocation().MethodName()
	if method == "" {
		return errx.New("empty method in thrift Marshal")
	}

	if err := codec.NewDataIfNeeded(method, message); err != nil {
		return err
	}

	// validate data
	object, err := validateData(method, message)
	if err != nil {
		return err
	}

	typeID := message.MessageType()
	seqID := message.RPCInfo().Invocation().SeqID()

	// encode object
	data, err := c.serializer.Encode(object)
	if err != nil {
		return perrors.NewProtocolErrorWithMsg(err.Error())
	}

	// calc message buffer
	beginLen := bthrift.Binary.MessageBeginLength(
		method,
		thrift.TMessageType(typeID),
		seqID)
	dataLen := len(data)
	endLen := bthrift.Binary.MessageEndLength()

	// malloc message buffer
	buf, err := out.Malloc(beginLen + dataLen + endLen)
	if err != nil {
		errMsg := fmt.Sprintf("thrift marshal, Malloc failed: %s", err.Error())
		return perrors.NewProtocolErrorWithMsg(errMsg)
	}

	// write begin message
	offset := bthrift.Binary.WriteMessageBegin(
		buf,
		method,
		thrift.TMessageType(typeID),
		seqID)

	// write data message
	copy(buf[offset:], data)

	// write end message
	offset += dataLen
	bthrift.Binary.WriteMessageEnd(buf[offset:])
	return nil
}

func (c Codec) Unmarshal(ctx context.Context, message remote.Message, in remote.ByteBuffer) error {
	bp := keThrift.NewBinaryProtocol(in)

	method, typeID, seqID, err := bp.ReadMessageBegin()
	if err != nil {
		errMsg := fmt.Sprintf(
			"thrift unmarshal, ReadMessageBegin failed: %s",
			err.Error())
		return perrors.NewProtocolErrorWithErrMsg(err, errMsg)
	}

	// update message type
	if err = codec.UpdateMsgType(uint32(typeID), message); err != nil {
		return err
	}

	// exception message
	if message.MessageType() == remote.Exception {
		return keThrift.UnmarshalThriftException(bp)
	}

	// validate message decode
	if err = codec.SetOrCheckSeqID(seqID, message); err != nil {
		return err
	}
	if err = codec.SetOrCheckMethodName(method, message); err != nil {
		return err
	}
	if err = codec.NewDataIfNeeded(method, message); err != nil {
		return err
	}

	// record read start
	ri := message.RPCInfo()
	rpcinfo.Record(ctx, ri, stats.WaitReadStart, nil)

	// calc data length
	beginLen := bthrift.Binary.MessageBeginLength(method, typeID, seqID)
	dataLen := message.PayloadLen() - beginLen - bthrift.Binary.MessageEndLength()
	if dataLen <= 0 {
		errMsg := fmt.Sprintf("caught in %s using SkipDecoder Buffer", c.name)
		return remote.NewTransError(remote.ProtocolError, errx.New(errMsg))
	}

	// read data message
	trans := bp.ByteBuffer()
	data, err := trans.Next(dataLen - bthrift.Binary.MessageEndLength())

	// record read finish
	rpcinfo.Record(ctx, ri, stats.WaitReadFinish, err)
	if err != nil {
		return remote.NewTransError(remote.ProtocolError, perrors.NewProtocolError(err))
	}

	// decode object
	object := message.Data()
	err = c.serializer.Decode(data, object)
	if err != nil {
		return remote.NewTransError(remote.ProtocolError, err)
	}

	// read message end
	if err = bp.ReadMessageEnd(); err != nil {
		return remote.NewTransError(remote.ProtocolError, err)
	}

	bp.Recycle()
	return err
}

func validateData(methodName string, message remote.Message) (any, error) {
	if err := codec.NewDataIfNeeded(methodName, message); err != nil {
		return nil, err
	}

	data := message.Data()
	if message.MessageType() != remote.Exception {
		return data, nil
	}

	transErr, isTransErr := data.(*remote.TransError)
	if isTransErr {
		data = thrift.NewTApplicationException(transErr.TypeID(), transErr.Error())
		return data, nil
	}

	if err, isError := data.(error); isError {
		data = thrift.NewTApplicationException(remote.InternalError, err.Error())
		return data, nil
	}

	return nil, errx.New("exception relay need error type data")
}
