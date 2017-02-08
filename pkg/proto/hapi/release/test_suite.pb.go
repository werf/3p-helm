// Code generated by protoc-gen-go.
// source: hapi/release/test_suite.proto
// DO NOT EDIT!

package release

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import google_protobuf "github.com/golang/protobuf/ptypes/timestamp"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// TestSuite comprises of the last run of the pre-defined test suite of a release version
type TestSuite struct {
	// StartedAt indicates the date/time this test suite was kicked off
	StartedAt *google_protobuf.Timestamp `protobuf:"bytes,1,opt,name=started_at,json=startedAt" json:"started_at,omitempty"`
	// CompletedAt indicates the date/time this test suite was completed
	CompletedAt *google_protobuf.Timestamp `protobuf:"bytes,2,opt,name=completed_at,json=completedAt" json:"completed_at,omitempty"`
	// Results are the results of each segment of the test
	Results []*TestRun `protobuf:"bytes,3,rep,name=results" json:"results,omitempty"`
}

func (m *TestSuite) Reset()                    { *m = TestSuite{} }
func (m *TestSuite) String() string            { return proto.CompactTextString(m) }
func (*TestSuite) ProtoMessage()               {}
func (*TestSuite) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{0} }

func (m *TestSuite) GetStartedAt() *google_protobuf.Timestamp {
	if m != nil {
		return m.StartedAt
	}
	return nil
}

func (m *TestSuite) GetCompletedAt() *google_protobuf.Timestamp {
	if m != nil {
		return m.CompletedAt
	}
	return nil
}

func (m *TestSuite) GetResults() []*TestRun {
	if m != nil {
		return m.Results
	}
	return nil
}

func init() {
	proto.RegisterType((*TestSuite)(nil), "hapi.release.TestSuite")
}

func init() { proto.RegisterFile("hapi/release/test_suite.proto", fileDescriptor5) }

var fileDescriptor5 = []byte{
	// 205 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xe2, 0x92, 0xcd, 0x48, 0x2c, 0xc8,
	0xd4, 0x2f, 0x4a, 0xcd, 0x49, 0x4d, 0x2c, 0x4e, 0xd5, 0x2f, 0x49, 0x2d, 0x2e, 0x89, 0x2f, 0x2e,
	0xcd, 0x2c, 0x49, 0xd5, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0xe2, 0x01, 0x49, 0xeb, 0x41, 0xa5,
	0xa5, 0xe4, 0xd3, 0xf3, 0xf3, 0xd3, 0x73, 0x52, 0xf5, 0xc1, 0x72, 0x49, 0xa5, 0x69, 0xfa, 0x25,
	0x99, 0xb9, 0x40, 0x1d, 0x89, 0xb9, 0x05, 0x10, 0xe5, 0x52, 0xd2, 0x98, 0xa6, 0x15, 0x95, 0xe6,
	0x41, 0x24, 0x95, 0xb6, 0x31, 0x72, 0x71, 0x86, 0x00, 0x85, 0x82, 0x41, 0xe6, 0x0b, 0x59, 0x72,
	0x71, 0x01, 0x75, 0x16, 0x95, 0xa4, 0xa6, 0xc4, 0x27, 0x96, 0x48, 0x30, 0x2a, 0x30, 0x6a, 0x70,
	0x1b, 0x49, 0xe9, 0x41, 0x2c, 0xd0, 0x83, 0x59, 0xa0, 0x17, 0x02, 0xb3, 0x20, 0x88, 0x13, 0xaa,
	0xda, 0xb1, 0x44, 0xc8, 0x96, 0x8b, 0x27, 0x39, 0x3f, 0xb7, 0x20, 0x27, 0x15, 0xaa, 0x99, 0x89,
	0xa0, 0x66, 0x6e, 0xb8, 0x7a, 0xa0, 0x76, 0x7d, 0x2e, 0xf6, 0xa2, 0xd4, 0xe2, 0xd2, 0x9c, 0x92,
	0x62, 0x09, 0x66, 0x05, 0x66, 0xa0, 0x4e, 0x51, 0x3d, 0x64, 0x5f, 0xea, 0x81, 0xdc, 0x18, 0x54,
	0x9a, 0x17, 0x04, 0x53, 0xe5, 0xc4, 0x19, 0xc5, 0x0e, 0x95, 0x4b, 0x62, 0x03, 0x1b, 0x6e, 0x0c,
	0x08, 0x00, 0x00, 0xff, 0xff, 0x8c, 0x59, 0x65, 0x4f, 0x37, 0x01, 0x00, 0x00,
}
