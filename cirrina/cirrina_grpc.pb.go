// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v3.21.9
// source: cirrina.proto

package cirrina

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

const (
	VMInfo_AddVM_FullMethodName         = "/cirrina.VMInfo/AddVM"
	VMInfo_GetVM_FullMethodName         = "/cirrina.VMInfo/GetVM"
	VMInfo_GetVMs_FullMethodName        = "/cirrina.VMInfo/GetVMs"
	VMInfo_StartVM_FullMethodName       = "/cirrina.VMInfo/StartVM"
	VMInfo_StopVM_FullMethodName        = "/cirrina.VMInfo/StopVM"
	VMInfo_RequestStatus_FullMethodName = "/cirrina.VMInfo/RequestStatus"
	VMInfo_GetVMState_FullMethodName    = "/cirrina.VMInfo/GetVMState"
)

// VMInfoClient is the client API for VMInfo service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type VMInfoClient interface {
	AddVM(ctx context.Context, in *VM, opts ...grpc.CallOption) (*VmID, error)
	GetVM(ctx context.Context, in *VmID, opts ...grpc.CallOption) (*VM, error)
	GetVMs(ctx context.Context, in *VMsQuery, opts ...grpc.CallOption) (VMInfo_GetVMsClient, error)
	StartVM(ctx context.Context, in *VmID, opts ...grpc.CallOption) (*RequestID, error)
	StopVM(ctx context.Context, in *VmID, opts ...grpc.CallOption) (*RequestID, error)
	RequestStatus(ctx context.Context, in *RequestID, opts ...grpc.CallOption) (*ReqStatus, error)
	GetVMState(ctx context.Context, in *VmID, opts ...grpc.CallOption) (*VMState, error)
}

type vMInfoClient struct {
	cc grpc.ClientConnInterface
}

func NewVMInfoClient(cc grpc.ClientConnInterface) VMInfoClient {
	return &vMInfoClient{cc}
}

func (c *vMInfoClient) AddVM(ctx context.Context, in *VM, opts ...grpc.CallOption) (*VmID, error) {
	out := new(VmID)
	err := c.cc.Invoke(ctx, VMInfo_AddVM_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *vMInfoClient) GetVM(ctx context.Context, in *VmID, opts ...grpc.CallOption) (*VM, error) {
	out := new(VM)
	err := c.cc.Invoke(ctx, VMInfo_GetVM_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *vMInfoClient) GetVMs(ctx context.Context, in *VMsQuery, opts ...grpc.CallOption) (VMInfo_GetVMsClient, error) {
	stream, err := c.cc.NewStream(ctx, &VMInfo_ServiceDesc.Streams[0], VMInfo_GetVMs_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &vMInfoGetVMsClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type VMInfo_GetVMsClient interface {
	Recv() (*VmID, error)
	grpc.ClientStream
}

type vMInfoGetVMsClient struct {
	grpc.ClientStream
}

func (x *vMInfoGetVMsClient) Recv() (*VmID, error) {
	m := new(VmID)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *vMInfoClient) StartVM(ctx context.Context, in *VmID, opts ...grpc.CallOption) (*RequestID, error) {
	out := new(RequestID)
	err := c.cc.Invoke(ctx, VMInfo_StartVM_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *vMInfoClient) StopVM(ctx context.Context, in *VmID, opts ...grpc.CallOption) (*RequestID, error) {
	out := new(RequestID)
	err := c.cc.Invoke(ctx, VMInfo_StopVM_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *vMInfoClient) RequestStatus(ctx context.Context, in *RequestID, opts ...grpc.CallOption) (*ReqStatus, error) {
	out := new(ReqStatus)
	err := c.cc.Invoke(ctx, VMInfo_RequestStatus_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *vMInfoClient) GetVMState(ctx context.Context, in *VmID, opts ...grpc.CallOption) (*VMState, error) {
	out := new(VMState)
	err := c.cc.Invoke(ctx, VMInfo_GetVMState_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// VMInfoServer is the server API for VMInfo service.
// All implementations must embed UnimplementedVMInfoServer
// for forward compatibility
type VMInfoServer interface {
	AddVM(context.Context, *VM) (*VmID, error)
	GetVM(context.Context, *VmID) (*VM, error)
	GetVMs(*VMsQuery, VMInfo_GetVMsServer) error
	StartVM(context.Context, *VmID) (*RequestID, error)
	StopVM(context.Context, *VmID) (*RequestID, error)
	RequestStatus(context.Context, *RequestID) (*ReqStatus, error)
	GetVMState(context.Context, *VmID) (*VMState, error)
	mustEmbedUnimplementedVMInfoServer()
}

// UnimplementedVMInfoServer must be embedded to have forward compatible implementations.
type UnimplementedVMInfoServer struct {
}

func (UnimplementedVMInfoServer) AddVM(context.Context, *VM) (*VmID, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddVM not implemented")
}
func (UnimplementedVMInfoServer) GetVM(context.Context, *VmID) (*VM, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetVM not implemented")
}
func (UnimplementedVMInfoServer) GetVMs(*VMsQuery, VMInfo_GetVMsServer) error {
	return status.Errorf(codes.Unimplemented, "method GetVMs not implemented")
}
func (UnimplementedVMInfoServer) StartVM(context.Context, *VmID) (*RequestID, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StartVM not implemented")
}
func (UnimplementedVMInfoServer) StopVM(context.Context, *VmID) (*RequestID, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StopVM not implemented")
}
func (UnimplementedVMInfoServer) RequestStatus(context.Context, *RequestID) (*ReqStatus, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RequestStatus not implemented")
}
func (UnimplementedVMInfoServer) GetVMState(context.Context, *VmID) (*VMState, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetVMState not implemented")
}
func (UnimplementedVMInfoServer) mustEmbedUnimplementedVMInfoServer() {}

// UnsafeVMInfoServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to VMInfoServer will
// result in compilation errors.
type UnsafeVMInfoServer interface {
	mustEmbedUnimplementedVMInfoServer()
}

func RegisterVMInfoServer(s grpc.ServiceRegistrar, srv VMInfoServer) {
	s.RegisterService(&VMInfo_ServiceDesc, srv)
}

func _VMInfo_AddVM_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(VM)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VMInfoServer).AddVM(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: VMInfo_AddVM_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VMInfoServer).AddVM(ctx, req.(*VM))
	}
	return interceptor(ctx, in, info, handler)
}

func _VMInfo_GetVM_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(VmID)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VMInfoServer).GetVM(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: VMInfo_GetVM_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VMInfoServer).GetVM(ctx, req.(*VmID))
	}
	return interceptor(ctx, in, info, handler)
}

func _VMInfo_GetVMs_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(VMsQuery)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(VMInfoServer).GetVMs(m, &vMInfoGetVMsServer{stream})
}

type VMInfo_GetVMsServer interface {
	Send(*VmID) error
	grpc.ServerStream
}

type vMInfoGetVMsServer struct {
	grpc.ServerStream
}

func (x *vMInfoGetVMsServer) Send(m *VmID) error {
	return x.ServerStream.SendMsg(m)
}

func _VMInfo_StartVM_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(VmID)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VMInfoServer).StartVM(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: VMInfo_StartVM_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VMInfoServer).StartVM(ctx, req.(*VmID))
	}
	return interceptor(ctx, in, info, handler)
}

func _VMInfo_StopVM_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(VmID)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VMInfoServer).StopVM(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: VMInfo_StopVM_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VMInfoServer).StopVM(ctx, req.(*VmID))
	}
	return interceptor(ctx, in, info, handler)
}

func _VMInfo_RequestStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RequestID)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VMInfoServer).RequestStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: VMInfo_RequestStatus_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VMInfoServer).RequestStatus(ctx, req.(*RequestID))
	}
	return interceptor(ctx, in, info, handler)
}

func _VMInfo_GetVMState_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(VmID)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VMInfoServer).GetVMState(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: VMInfo_GetVMState_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VMInfoServer).GetVMState(ctx, req.(*VmID))
	}
	return interceptor(ctx, in, info, handler)
}

// VMInfo_ServiceDesc is the grpc.ServiceDesc for VMInfo service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var VMInfo_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "cirrina.VMInfo",
	HandlerType: (*VMInfoServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "AddVM",
			Handler:    _VMInfo_AddVM_Handler,
		},
		{
			MethodName: "GetVM",
			Handler:    _VMInfo_GetVM_Handler,
		},
		{
			MethodName: "StartVM",
			Handler:    _VMInfo_StartVM_Handler,
		},
		{
			MethodName: "StopVM",
			Handler:    _VMInfo_StopVM_Handler,
		},
		{
			MethodName: "RequestStatus",
			Handler:    _VMInfo_RequestStatus_Handler,
		},
		{
			MethodName: "GetVMState",
			Handler:    _VMInfo_GetVMState_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "GetVMs",
			Handler:       _VMInfo_GetVMs_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "cirrina.proto",
}
