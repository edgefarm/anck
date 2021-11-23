// Code generated by protoc-gen-go. DO NOT EDIT.
// source: config.proto

package v1alpha1

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type DesiredStateRequest struct {
	Account              string   `protobuf:"bytes,1,opt,name=account,proto3" json:"account,omitempty"`
	Username             []string `protobuf:"bytes,2,rep,name=username,proto3" json:"username,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DesiredStateRequest) Reset()         { *m = DesiredStateRequest{} }
func (m *DesiredStateRequest) String() string { return proto.CompactTextString(m) }
func (*DesiredStateRequest) ProtoMessage()    {}
func (*DesiredStateRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_3eaf2c85e69e9ea4, []int{0}
}

func (m *DesiredStateRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DesiredStateRequest.Unmarshal(m, b)
}
func (m *DesiredStateRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DesiredStateRequest.Marshal(b, m, deterministic)
}
func (m *DesiredStateRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DesiredStateRequest.Merge(m, src)
}
func (m *DesiredStateRequest) XXX_Size() int {
	return xxx_messageInfo_DesiredStateRequest.Size(m)
}
func (m *DesiredStateRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_DesiredStateRequest.DiscardUnknown(m)
}

var xxx_messageInfo_DesiredStateRequest proto.InternalMessageInfo

func (m *DesiredStateRequest) GetAccount() string {
	if m != nil {
		return m.Account
	}
	return ""
}

func (m *DesiredStateRequest) GetUsername() []string {
	if m != nil {
		return m.Username
	}
	return nil
}

type DesiredStateResponse struct {
	Credentials          map[string]string `protobuf:"bytes,1,rep,name=credentials,proto3" json:"credentials,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *DesiredStateResponse) Reset()         { *m = DesiredStateResponse{} }
func (m *DesiredStateResponse) String() string { return proto.CompactTextString(m) }
func (*DesiredStateResponse) ProtoMessage()    {}
func (*DesiredStateResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_3eaf2c85e69e9ea4, []int{1}
}

func (m *DesiredStateResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DesiredStateResponse.Unmarshal(m, b)
}
func (m *DesiredStateResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DesiredStateResponse.Marshal(b, m, deterministic)
}
func (m *DesiredStateResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DesiredStateResponse.Merge(m, src)
}
func (m *DesiredStateResponse) XXX_Size() int {
	return xxx_messageInfo_DesiredStateResponse.Size(m)
}
func (m *DesiredStateResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_DesiredStateResponse.DiscardUnknown(m)
}

var xxx_messageInfo_DesiredStateResponse proto.InternalMessageInfo

func (m *DesiredStateResponse) GetCredentials() map[string]string {
	if m != nil {
		return m.Credentials
	}
	return nil
}

type DeleteAccountRequest struct {
	Account              string   `protobuf:"bytes,1,opt,name=account,proto3" json:"account,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DeleteAccountRequest) Reset()         { *m = DeleteAccountRequest{} }
func (m *DeleteAccountRequest) String() string { return proto.CompactTextString(m) }
func (*DeleteAccountRequest) ProtoMessage()    {}
func (*DeleteAccountRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_3eaf2c85e69e9ea4, []int{2}
}

func (m *DeleteAccountRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DeleteAccountRequest.Unmarshal(m, b)
}
func (m *DeleteAccountRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DeleteAccountRequest.Marshal(b, m, deterministic)
}
func (m *DeleteAccountRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DeleteAccountRequest.Merge(m, src)
}
func (m *DeleteAccountRequest) XXX_Size() int {
	return xxx_messageInfo_DeleteAccountRequest.Size(m)
}
func (m *DeleteAccountRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_DeleteAccountRequest.DiscardUnknown(m)
}

var xxx_messageInfo_DeleteAccountRequest proto.InternalMessageInfo

func (m *DeleteAccountRequest) GetAccount() string {
	if m != nil {
		return m.Account
	}
	return ""
}

type DeleteAccountResponse struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DeleteAccountResponse) Reset()         { *m = DeleteAccountResponse{} }
func (m *DeleteAccountResponse) String() string { return proto.CompactTextString(m) }
func (*DeleteAccountResponse) ProtoMessage()    {}
func (*DeleteAccountResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_3eaf2c85e69e9ea4, []int{3}
}

func (m *DeleteAccountResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DeleteAccountResponse.Unmarshal(m, b)
}
func (m *DeleteAccountResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DeleteAccountResponse.Marshal(b, m, deterministic)
}
func (m *DeleteAccountResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DeleteAccountResponse.Merge(m, src)
}
func (m *DeleteAccountResponse) XXX_Size() int {
	return xxx_messageInfo_DeleteAccountResponse.Size(m)
}
func (m *DeleteAccountResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_DeleteAccountResponse.DiscardUnknown(m)
}

var xxx_messageInfo_DeleteAccountResponse proto.InternalMessageInfo

func init() {
	proto.RegisterType((*DesiredStateRequest)(nil), "v1alpha1.DesiredStateRequest")
	proto.RegisterType((*DesiredStateResponse)(nil), "v1alpha1.DesiredStateResponse")
	proto.RegisterMapType((map[string]string)(nil), "v1alpha1.DesiredStateResponse.CredentialsEntry")
	proto.RegisterType((*DeleteAccountRequest)(nil), "v1alpha1.DeleteAccountRequest")
	proto.RegisterType((*DeleteAccountResponse)(nil), "v1alpha1.DeleteAccountResponse")
}

func init() { proto.RegisterFile("config.proto", fileDescriptor_3eaf2c85e69e9ea4) }

var fileDescriptor_3eaf2c85e69e9ea4 = []byte{
	// 288 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x84, 0x92, 0x41, 0x4b, 0xc3, 0x40,
	0x10, 0x85, 0xd9, 0x16, 0xb5, 0x9d, 0xa6, 0x50, 0xc6, 0x8a, 0x21, 0xa0, 0x86, 0x9c, 0x72, 0x8a,
	0xb6, 0x5e, 0x44, 0x41, 0xd0, 0xea, 0x49, 0x04, 0x4d, 0x6f, 0xde, 0xd6, 0x74, 0xd4, 0x60, 0xdc,
	0x8d, 0xbb, 0x9b, 0x40, 0x7f, 0x92, 0x37, 0x7f, 0xa2, 0x24, 0x31, 0xda, 0x54, 0xab, 0xb7, 0x9d,
	0xdd, 0x79, 0x6f, 0xde, 0x37, 0x2c, 0x58, 0x91, 0x14, 0x0f, 0xf1, 0x63, 0x90, 0x2a, 0x69, 0x24,
	0x76, 0xf2, 0x11, 0x4f, 0xd2, 0x27, 0x3e, 0xf2, 0xae, 0x60, 0xf3, 0x82, 0x74, 0xac, 0x68, 0x36,
	0x35, 0xdc, 0x50, 0x48, 0xaf, 0x19, 0x69, 0x83, 0x36, 0x6c, 0xf0, 0x28, 0x92, 0x99, 0x30, 0x36,
	0x73, 0x99, 0xdf, 0x0d, 0xeb, 0x12, 0x1d, 0xe8, 0x64, 0x9a, 0x94, 0xe0, 0x2f, 0x64, 0xb7, 0xdc,
	0xb6, 0xdf, 0x0d, 0xbf, 0x6a, 0xef, 0x8d, 0xc1, 0xb0, 0xe9, 0xa6, 0x53, 0x29, 0x34, 0xe1, 0x2d,
	0xf4, 0x22, 0x45, 0x33, 0x12, 0x26, 0xe6, 0x89, 0xb6, 0x99, 0xdb, 0xf6, 0x7b, 0xe3, 0xfd, 0xa0,
	0x4e, 0x11, 0xfc, 0x26, 0x0a, 0x26, 0xdf, 0x8a, 0x4b, 0x61, 0xd4, 0x3c, 0x5c, 0xf4, 0x70, 0x4e,
	0x61, 0xb0, 0xdc, 0x80, 0x03, 0x68, 0x3f, 0xd3, 0xfc, 0x33, 0x71, 0x71, 0xc4, 0x21, 0xac, 0xe5,
	0x3c, 0xc9, 0x8a, 0xa8, 0xc5, 0x5d, 0x55, 0x1c, 0xb7, 0x8e, 0x98, 0x77, 0x50, 0x44, 0x4d, 0xc8,
	0xd0, 0x59, 0x05, 0xf6, 0x2f, 0xb9, 0xb7, 0x0d, 0x5b, 0x4b, 0x8a, 0x2a, 0xe8, 0xf8, 0x9d, 0x41,
	0x7f, 0x52, 0xae, 0x77, 0x4a, 0x2a, 0x8f, 0x23, 0xc2, 0x6b, 0xb0, 0x16, 0x91, 0x70, 0x67, 0x15,
	0x6a, 0x39, 0xd3, 0xd9, 0xfd, 0x7b, 0x13, 0x78, 0x03, 0xfd, 0xc6, 0x64, 0x6c, 0x08, 0x7e, 0x42,
	0x38, 0x7b, 0x2b, 0xdf, 0x2b, 0xc7, 0x73, 0xeb, 0x0e, 0x82, 0x93, 0xba, 0xe7, 0x7e, 0xbd, 0xfc,
	0x15, 0x87, 0x1f, 0x01, 0x00, 0x00, 0xff, 0xff, 0xee, 0x64, 0x20, 0xd0, 0x25, 0x02, 0x00, 0x00,
}
