// Code generated by protoc-gen-go. DO NOT EDIT.
// source: hapi/chart/metadata.proto

package chart

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type Metadata_Engine int32

const (
	Metadata_UNKNOWN Metadata_Engine = 0
	Metadata_GOTPL   Metadata_Engine = 1
)

var Metadata_Engine_name = map[int32]string{
	0: "UNKNOWN",
	1: "GOTPL",
}
var Metadata_Engine_value = map[string]int32{
	"UNKNOWN": 0,
	"GOTPL":   1,
}

func (x Metadata_Engine) String() string {
	return proto.EnumName(Metadata_Engine_name, int32(x))
}
func (Metadata_Engine) EnumDescriptor() ([]byte, []int) { return fileDescriptor2, []int{1, 0} }

// Maintainer describes a Chart maintainer.
type Maintainer struct {
	// Name is a user name or organization name
	Name string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	// Email is an optional email address to contact the named maintainer
	Email string `protobuf:"bytes,2,opt,name=email" json:"email,omitempty"`
	// Url is an optional URL to an address for the named maintainer
	Url string `protobuf:"bytes,3,opt,name=url" json:"url,omitempty"`
}

func (m *Maintainer) Reset()                    { *m = Maintainer{} }
func (m *Maintainer) String() string            { return proto.CompactTextString(m) }
func (*Maintainer) ProtoMessage()               {}
func (*Maintainer) Descriptor() ([]byte, []int) { return fileDescriptor2, []int{0} }

func (m *Maintainer) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Maintainer) GetEmail() string {
	if m != nil {
		return m.Email
	}
	return ""
}

func (m *Maintainer) GetUrl() string {
	if m != nil {
		return m.Url
	}
	return ""
}

// 	Metadata for a Chart file. This models the structure of a Chart.yaml file.
//
// 	Spec: https://k8s.io/helm/blob/master/docs/design/chart_format.md#the-chart-file
type Metadata struct {
	// The name of the chart
	Name string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	// The URL to a relevant project page, git repo, or contact person
	Home string `protobuf:"bytes,2,opt,name=home" json:"home,omitempty"`
	// Source is the URL to the source code of this chart
	Sources []string `protobuf:"bytes,3,rep,name=sources" json:"sources,omitempty"`
	// A SemVer 2 conformant version string of the chart
	Version string `protobuf:"bytes,4,opt,name=version" json:"version,omitempty"`
	// A one-sentence description of the chart
	Description string `protobuf:"bytes,5,opt,name=description" json:"description,omitempty"`
	// A list of string keywords
	Keywords []string `protobuf:"bytes,6,rep,name=keywords" json:"keywords,omitempty"`
	// A list of name and URL/email address combinations for the maintainer(s)
	Maintainers []*Maintainer `protobuf:"bytes,7,rep,name=maintainers" json:"maintainers,omitempty"`
	// The name of the template engine to use. Defaults to 'gotpl'.
	Engine string `protobuf:"bytes,8,opt,name=engine" json:"engine,omitempty"`
	// The URL to an icon file.
	Icon string `protobuf:"bytes,9,opt,name=icon" json:"icon,omitempty"`
	// The API Version of this chart.
	ApiVersion string `protobuf:"bytes,10,opt,name=apiVersion" json:"apiVersion,omitempty"`
	// The condition to check to enable chart
	Condition string `protobuf:"bytes,11,opt,name=condition" json:"condition,omitempty"`
	// The tags to check to enable chart
	Tags string `protobuf:"bytes,12,opt,name=tags" json:"tags,omitempty"`
	// The version of the application enclosed inside of this chart.
	AppVersion string `protobuf:"bytes,13,opt,name=appVersion" json:"appVersion,omitempty"`
	// Whether or not this chart is deprecated
	Deprecated bool `protobuf:"varint,14,opt,name=deprecated" json:"deprecated,omitempty"`
	// TillerVersion is a SemVer constraints on what version of Tiller is required.
	// See SemVer ranges here: https://github.com/Masterminds/semver#basic-comparisons
	TillerVersion string `protobuf:"bytes,15,opt,name=tillerVersion" json:"tillerVersion,omitempty"`
	// Annotations are additional mappings uninterpreted by Tiller,
	// made available for inspection by other applications.
	Annotations map[string]string `protobuf:"bytes,16,rep,name=annotations" json:"annotations,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

func (m *Metadata) Reset()                    { *m = Metadata{} }
func (m *Metadata) String() string            { return proto.CompactTextString(m) }
func (*Metadata) ProtoMessage()               {}
func (*Metadata) Descriptor() ([]byte, []int) { return fileDescriptor2, []int{1} }

func (m *Metadata) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Metadata) GetHome() string {
	if m != nil {
		return m.Home
	}
	return ""
}

func (m *Metadata) GetSources() []string {
	if m != nil {
		return m.Sources
	}
	return nil
}

func (m *Metadata) GetVersion() string {
	if m != nil {
		return m.Version
	}
	return ""
}

func (m *Metadata) GetDescription() string {
	if m != nil {
		return m.Description
	}
	return ""
}

func (m *Metadata) GetKeywords() []string {
	if m != nil {
		return m.Keywords
	}
	return nil
}

func (m *Metadata) GetMaintainers() []*Maintainer {
	if m != nil {
		return m.Maintainers
	}
	return nil
}

func (m *Metadata) GetEngine() string {
	if m != nil {
		return m.Engine
	}
	return ""
}

func (m *Metadata) GetIcon() string {
	if m != nil {
		return m.Icon
	}
	return ""
}

func (m *Metadata) GetApiVersion() string {
	if m != nil {
		return m.ApiVersion
	}
	return ""
}

func (m *Metadata) GetCondition() string {
	if m != nil {
		return m.Condition
	}
	return ""
}

func (m *Metadata) GetTags() string {
	if m != nil {
		return m.Tags
	}
	return ""
}

func (m *Metadata) GetAppVersion() string {
	if m != nil {
		return m.AppVersion
	}
	return ""
}

func (m *Metadata) GetDeprecated() bool {
	if m != nil {
		return m.Deprecated
	}
	return false
}

func (m *Metadata) GetTillerVersion() string {
	if m != nil {
		return m.TillerVersion
	}
	return ""
}

func (m *Metadata) GetAnnotations() map[string]string {
	if m != nil {
		return m.Annotations
	}
	return nil
}

func init() {
	proto.RegisterType((*Maintainer)(nil), "hapi.chart.Maintainer")
	proto.RegisterType((*Metadata)(nil), "hapi.chart.Metadata")
	proto.RegisterEnum("hapi.chart.Metadata_Engine", Metadata_Engine_name, Metadata_Engine_value)
}

func init() { proto.RegisterFile("hapi/chart/metadata.proto", fileDescriptor2) }

var fileDescriptor2 = []byte{
	// 420 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x52, 0x5b, 0x6b, 0xd4, 0x40,
	0x14, 0x36, 0xcd, 0xde, 0x72, 0x62, 0x35, 0x1c, 0xa4, 0x8c, 0x45, 0x24, 0x2c, 0x0a, 0xfb, 0xb4,
	0x05, 0x7d, 0x29, 0x3e, 0x08, 0x0a, 0xa5, 0x82, 0x76, 0x2b, 0xc1, 0x0b, 0xf8, 0x36, 0x26, 0x87,
	0xee, 0xd0, 0x64, 0x26, 0x4c, 0x66, 0x2b, 0xfb, 0xa3, 0xfd, 0x0f, 0x32, 0x27, 0x49, 0x93, 0x95,
	0xbe, 0x7d, 0x97, 0x99, 0x6f, 0xe6, 0x1c, 0x3e, 0x78, 0xbe, 0x95, 0xb5, 0x3a, 0xcb, 0xb7, 0xd2,
	0xba, 0xb3, 0x8a, 0x9c, 0x2c, 0xa4, 0x93, 0xeb, 0xda, 0x1a, 0x67, 0x10, 0xbc, 0xb5, 0x66, 0x6b,
	0xf9, 0x09, 0xe0, 0x4a, 0x2a, 0xed, 0xa4, 0xd2, 0x64, 0x11, 0x61, 0xa2, 0x65, 0x45, 0x22, 0x48,
	0x83, 0x55, 0x94, 0x31, 0xc6, 0x67, 0x30, 0xa5, 0x4a, 0xaa, 0x52, 0x1c, 0xb1, 0xd8, 0x12, 0x4c,
	0x20, 0xdc, 0xd9, 0x52, 0x84, 0xac, 0x79, 0xb8, 0xfc, 0x3b, 0x81, 0xc5, 0x55, 0xf7, 0xd0, 0x83,
	0x41, 0x08, 0x93, 0xad, 0xa9, 0xa8, 0xcb, 0x61, 0x8c, 0x02, 0xe6, 0x8d, 0xd9, 0xd9, 0x9c, 0x1a,
	0x11, 0xa6, 0xe1, 0x2a, 0xca, 0x7a, 0xea, 0x9d, 0x3b, 0xb2, 0x8d, 0x32, 0x5a, 0x4c, 0xf8, 0x42,
	0x4f, 0x31, 0x85, 0xb8, 0xa0, 0x26, 0xb7, 0xaa, 0x76, 0xde, 0x9d, 0xb2, 0x3b, 0x96, 0xf0, 0x14,
	0x16, 0xb7, 0xb4, 0xff, 0x63, 0x6c, 0xd1, 0x88, 0x19, 0xc7, 0xde, 0x73, 0x3c, 0x87, 0xb8, 0xba,
	0x1f, 0xb8, 0x11, 0xf3, 0x34, 0x5c, 0xc5, 0x6f, 0x4e, 0xd6, 0xc3, 0x4a, 0xd6, 0xc3, 0x3e, 0xb2,
	0xf1, 0x51, 0x3c, 0x81, 0x19, 0xe9, 0x1b, 0xa5, 0x49, 0x2c, 0xf8, 0xc9, 0x8e, 0xf9, 0xb9, 0x54,
	0x6e, 0xb4, 0x88, 0xda, 0xb9, 0x3c, 0xc6, 0x97, 0x00, 0xb2, 0x56, 0x3f, 0xba, 0x01, 0x80, 0x9d,
	0x91, 0x82, 0x2f, 0x20, 0xca, 0x8d, 0x2e, 0x14, 0x4f, 0x10, 0xb3, 0x3d, 0x08, 0x3e, 0xd1, 0xc9,
	0x9b, 0x46, 0x3c, 0x6e, 0x13, 0x3d, 0x6e, 0x13, 0xeb, 0x3e, 0xf1, 0xb8, 0x4f, 0xec, 0x15, 0xef,
	0x17, 0x54, 0x5b, 0xca, 0xa5, 0xa3, 0x42, 0x3c, 0x49, 0x83, 0xd5, 0x22, 0x1b, 0x29, 0xf8, 0x0a,
	0x8e, 0x9d, 0x2a, 0x4b, 0xb2, 0x7d, 0xc4, 0x53, 0x8e, 0x38, 0x14, 0xf1, 0x12, 0x62, 0xa9, 0xb5,
	0x71, 0xd2, 0xff, 0xa3, 0x11, 0x09, 0x6f, 0xe7, 0xf5, 0xc1, 0x76, 0xfa, 0x2e, 0x7d, 0x18, 0xce,
	0x5d, 0x68, 0x67, 0xf7, 0xd9, 0xf8, 0xe6, 0xe9, 0x7b, 0x48, 0xfe, 0x3f, 0xe0, 0x3b, 0x73, 0x4b,
	0xfb, 0xae, 0x13, 0x1e, 0xfa, 0x6e, 0xdd, 0xc9, 0x72, 0xd7, 0x77, 0xa2, 0x25, 0xef, 0x8e, 0xce,
	0x83, 0x65, 0x0a, 0xb3, 0x8b, 0x76, 0xbd, 0x31, 0xcc, 0xbf, 0x6f, 0x3e, 0x6f, 0xae, 0x7f, 0x6e,
	0x92, 0x47, 0x18, 0xc1, 0xf4, 0xf2, 0xfa, 0xdb, 0xd7, 0x2f, 0x49, 0xf0, 0x71, 0xfe, 0x6b, 0xca,
	0x3f, 0xfa, 0x3d, 0xe3, 0x56, 0xbf, 0xfd, 0x17, 0x00, 0x00, 0xff, 0xff, 0x40, 0x4c, 0x34, 0x92,
	0xf2, 0x02, 0x00, 0x00,
}
