package main

// 本文件验证 Admin package 扁平化只重写所属消息与 RPC 输入输出，外部领域类型引用必须保持不变。

import (
	"testing"

	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// TestFlattenAdminPackagesRewritesOwnedTypeReferences 验证Proto 契约边界检查中 main 的预期行为、拒绝条件与回归边界。
func TestFlattenAdminPackagesRewritesOwnedTypeReferences(t *testing.T) {
	request := &pluginpb.CodeGeneratorRequest{ProtoFile: []*descriptorpb.FileDescriptorProto{{
		Name:    new("admin/gift/v1/catalog.proto"),
		Package: new("admin.gift.v1"),
		MessageType: []*descriptorpb.DescriptorProto{{
			Name: new("Request"),
			Field: []*descriptorpb.FieldDescriptorProto{{
				Name:     new("nested"),
				TypeName: new(".admin.gift.v1.Request.Nested"),
			}},
		}},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: new("GiftService"),
			Method: []*descriptorpb.MethodDescriptorProto{{
				Name:       new("Get"),
				InputType:  new(".admin.gift.v1.Request"),
				OutputType: new(".gift.v1.Gift"),
			}},
		}},
	}}}

	flattenAdminPackages(request)
	file := request.GetProtoFile()[0]
	if got := file.GetPackage(); got != "admin.v1" {
		t.Fatalf("package = %q, want admin.v1", got)
	}
	if got := file.GetMessageType()[0].GetField()[0].GetTypeName(); got != ".admin.v1.Request.Nested" {
		t.Fatalf("field type = %q, want flattened Admin type", got)
	}
	method := file.GetService()[0].GetMethod()[0]
	if got := method.GetInputType(); got != ".admin.v1.Request" {
		t.Fatalf("input type = %q, want flattened Admin type", got)
	}
	if got := method.GetOutputType(); got != ".gift.v1.Gift" {
		t.Fatalf("output type = %q, want domain type unchanged", got)
	}
}
