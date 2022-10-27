// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.12.4
// source: sinks/pb/sinks.proto

package pb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// SinkServiceClient is the client API for SinkService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type SinkServiceClient interface {
	RetrieveSink(ctx context.Context, in *SinkByIDReq, opts ...grpc.CallOption) (*SinkRes, error)
	RetrieveSinks(ctx context.Context, in *SinksFilterReq, opts ...grpc.CallOption) (*SinksRes, error)
}

type sinkServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewSinkServiceClient(cc grpc.ClientConnInterface) SinkServiceClient {
	return &sinkServiceClient{cc}
}

func (c *sinkServiceClient) RetrieveSink(ctx context.Context, in *SinkByIDReq, opts ...grpc.CallOption) (*SinkRes, error) {
	out := new(SinkRes)
	err := c.cc.Invoke(ctx, "/sinks.SinkService/RetrieveSink", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sinkServiceClient) RetrieveSinks(ctx context.Context, in *SinksFilterReq, opts ...grpc.CallOption) (*SinksRes, error) {
	out := new(SinksRes)
	err := c.cc.Invoke(ctx, "/sinks.SinkService/RetrieveSinks", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// SinkServiceServer is the server API for SinkService service.
// All implementations must embed UnimplementedSinkServiceServer
// for forward compatibility
type SinkServiceServer interface {
	RetrieveSink(context.Context, *SinkByIDReq) (*SinkRes, error)
	RetrieveSinks(context.Context, *SinksFilterReq) (*SinksRes, error)
	mustEmbedUnimplementedSinkServiceServer()
}

// UnimplementedSinkServiceServer must be embedded to have forward compatible implementations.
type UnimplementedSinkServiceServer struct {
}

func (UnimplementedSinkServiceServer) RetrieveSink(context.Context, *SinkByIDReq) (*SinkRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RetrieveSink not implemented")
}
func (UnimplementedSinkServiceServer) RetrieveSinks(context.Context, *SinksFilterReq) (*SinksRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RetrieveSinks not implemented")
}
func (UnimplementedSinkServiceServer) mustEmbedUnimplementedSinkServiceServer() {}

// UnsafeSinkServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to SinkServiceServer will
// result in compilation errors.
type UnsafeSinkServiceServer interface {
	mustEmbedUnimplementedSinkServiceServer()
}

func RegisterSinkServiceServer(s grpc.ServiceRegistrar, srv SinkServiceServer) {
	s.RegisterService(&SinkService_ServiceDesc, srv)
}

func _SinkService_RetrieveSink_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SinkByIDReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SinkServiceServer).RetrieveSink(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/sinks.SinkService/RetrieveSink",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SinkServiceServer).RetrieveSink(ctx, req.(*SinkByIDReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _SinkService_RetrieveSinks_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SinksFilterReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SinkServiceServer).RetrieveSinks(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/sinks.SinkService/RetrieveSinks",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SinkServiceServer).RetrieveSinks(ctx, req.(*SinksFilterReq))
	}
	return interceptor(ctx, in, info, handler)
}

// SinkService_ServiceDesc is the grpc.ServiceDesc for SinkService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var SinkService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "sinks.SinkService",
	HandlerType: (*SinkServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "RetrieveSink",
			Handler:    _SinkService_RetrieveSink_Handler,
		},
		{
			MethodName: "RetrieveSinks",
			Handler:    _SinkService_RetrieveSinks_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "sinks/pb/sinks.proto",
}
