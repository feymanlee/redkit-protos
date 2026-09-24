package main

// 本文件用合成描述符证明单 App 守卫拒绝代表性违规，并放行与平台 App 身份无关的 app_* 字段。

import (
	"testing"

	"google.golang.org/protobuf/types/descriptorpb"
)

// fieldFixture 构造一个用于检查器测试的消息字段描述符。
func fieldFixture(name string, number int32, typeName string) *descriptorpb.FieldDescriptorProto {
	field := &descriptorpb.FieldDescriptorProto{
		Name:   new(name),
		Number: new(number),
		Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
	}
	if typeName != "" {
		field.TypeName = new(typeName)
	}
	return field
}

// messageFixture 构造一个属于给定 package 的文件，只包含一个消息。
func messageFixture(packageName, messageName string, fields ...*descriptorpb.FieldDescriptorProto) *descriptorpb.FileDescriptorSet {
	return &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{
		Name:    new(packageName + "/v1/fixture.proto"),
		Package: new(packageName + ".v1"),
		MessageType: []*descriptorpb.DescriptorProto{{
			Name:  new(messageName),
			Field: fields,
		}},
	}}}
}

// TestInspectRejectsAppIdentityFields 验证检查器拒绝任何仍携带平台 App 身份的字段。
func TestInspectRejectsAppIdentityFields(t *testing.T) {
	for name, descriptorSet := range map[string]*descriptorpb.FileDescriptorSet{
		"raw app_id":        messageFixture("wallet", "Wallet", fieldFixture("app_id", 1, "")),
		"platform app name": messageFixture("core.operator", "Operator", fieldFixture("app_name", 2, "")),
		"nested app id":     messageFixture("user", "Request", fieldFixture("previous_app_id", 3, "")),
		"canonical enum":    messageFixture("user", "Request", fieldFixture("app_id", 1, ".common.v1.AppId")),
	} {
		if violations := inspect(descriptorSet); len(violations) == 0 {
			t.Errorf("inspect() accepted a contract that still carries App identity: %s", name)
		}
	}
}

// TestInspectRejectsAppSelectionServiceAndErrors 验证检查器拒绝 App Selection 服务、App 枚举与 App Scope 错误原因。
func TestInspectRejectsAppSelectionServiceAndErrors(t *testing.T) {
	serviceSet := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{
		Name:    new("admin/platform/v1/app_selection.proto"),
		Package: new("admin.platform.v1"),
		Service: []*descriptorpb.ServiceDescriptorProto{{Name: new("AppSelectionService")}},
	}}}
	if violations := inspect(serviceSet); len(violations) == 0 {
		t.Error("inspect() accepted a contract that still exposes App Selection")
	}

	enumSet := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{
		Name:    new("common/v1/common.proto"),
		Package: new("common.v1"),
		EnumType: []*descriptorpb.EnumDescriptorProto{{
			Name: new("AppId"),
			Value: []*descriptorpb.EnumValueDescriptorProto{
				{Name: new("APP_ID_UNSPECIFIED"), Number: new(int32(0))},
			},
		}},
	}}}
	if violations := inspect(enumSet); len(violations) == 0 {
		t.Error("inspect() accepted a contract that still declares the AppId enum")
	}

	errorSet := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{
		Name:    new("common/v1/common_error.proto"),
		Package: new("common.v1"),
		EnumType: []*descriptorpb.EnumDescriptorProto{{
			Name: new("ErrorReason"),
			Value: []*descriptorpb.EnumValueDescriptorProto{
				{Name: new("APP_SCOPE_REQUIRED"), Number: new(int32(4001))},
			},
		}},
	}}}
	if violations := inspect(errorSet); len(violations) == 0 {
		t.Error("inspect() accepted a contract that still declares App scope error reasons")
	}
}

// TestInspectAllowsNonAppIdentityFields 验证检查器放行外部 Store / 渠道应用身份与客户端应用名。
func TestInspectAllowsNonAppIdentityFields(t *testing.T) {
	for name, descriptorSet := range map[string]*descriptorpb.FileDescriptorSet{
		"store app":   messageFixture("wallet", "RechargeProduct", fieldFixture("store_app_id", 1, "")),
		"channel app": messageFixture("payment", "PaymentChannel", fieldFixture("channel_app_id", 6, "")),
		"app version": messageFixture("user.types", "Device", fieldFixture("app_version", 4, "")),
	} {
		if violations := inspect(descriptorSet); len(violations) != 0 {
			t.Errorf("inspect() rejected a contract unrelated to platform App identity: %s %v", name, violations)
		}
	}

	clientAppName := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{
		Name:    new("core/audit/v1/device_info.proto"),
		Package: new("core.audit.v1"),
		MessageType: []*descriptorpb.DescriptorProto{{
			Name:  new("DeviceInfo"),
			Field: []*descriptorpb.FieldDescriptorProto{fieldFixture("app_name", 11, "")},
		}},
	}}}
	if violations := inspect(clientAppName); len(violations) != 0 {
		t.Errorf("inspect() rejected the client application name: %v", violations)
	}
}
