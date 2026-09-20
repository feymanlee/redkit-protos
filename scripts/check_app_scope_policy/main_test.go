package main

// 本文件覆盖 App Scope 契约检查器对内部 service 请求身份类型的拒绝规则，防止协议回退为裸数值 app_id。

import (
	"testing"

	"google.golang.org/protobuf/types/descriptorpb"
)

// TestOpsContractsUseCanonicalAppIdentity 验证Proto 契约边界检查中 main 的预期行为、拒绝条件与回归边界。
func TestOpsContractsUseCanonicalAppIdentity(t *testing.T) {
	requestName := "GrantRewardRequest"
	serviceName := "RewardService"
	methodName := "GrantReward"
	inputType := ".ops.reward.v1." + requestName
	outputType := ".ops.reward.v1.GrantRewardResponse"
	fieldName := "app_id"
	packageName := "ops.reward.v1"
	fileName := "ops/reward/v1/reward.proto"
	fieldNumber := int32(1)
	violations := inspect(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{
		Name:    &fileName,
		Package: &packageName,
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: &requestName, Field: []*descriptorpb.FieldDescriptorProto{{
				Name: &fieldName, Number: &fieldNumber,
				Type: descriptorpb.FieldDescriptorProto_TYPE_UINT32.Enum(),
			}}},
			{Name: new("GrantRewardResponse")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: &serviceName,
			Method: []*descriptorpb.MethodDescriptorProto{{
				Name: &methodName, InputType: &inputType, OutputType: &outputType,
			}},
		}},
	}}})

	if len(violations) != 1 {
		t.Fatalf("inspect() violations = %v, want one canonical App identity violation", violations)
	}
	if got, want := violations[0].id, "internal-app-id-type:ops.reward.v1.GrantRewardRequest.app_id"; got != want {
		t.Fatalf("violation id = %q, want %q", got, want)
	}
}

// TestCoreApplicationContractsUseCanonicalAppIdentity 验证Proto 契约边界检查中 main 的预期行为、拒绝条件与回归边界。
func TestCoreApplicationContractsUseCanonicalAppIdentity(t *testing.T) {
	requestName := "GetBusinessTimeZoneRequest"
	inputType := ".core.application.v1." + requestName
	outputType := ".core.application.v1.AppBusinessTimeZone"
	fieldName := "app_id"
	packageName := "core.application.v1"
	fileName := "core/application/v1/business_time_zone.proto"
	fieldNumber := int32(1)
	serviceName := "BusinessTimeZoneService"
	methodName := "Get"
	violations := inspect(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{
		Name:    &fileName,
		Package: &packageName,
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: &requestName, Field: []*descriptorpb.FieldDescriptorProto{{
				Name: &fieldName, Number: &fieldNumber,
				Type: descriptorpb.FieldDescriptorProto_TYPE_UINT32.Enum(),
			}}},
			{Name: new("AppBusinessTimeZone")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: &serviceName,
			Method: []*descriptorpb.MethodDescriptorProto{{
				Name: &methodName, InputType: &inputType, OutputType: &outputType,
			}},
		}},
	}}})

	if len(violations) != 1 {
		t.Fatalf("inspect() violations = %v, want one canonical App identity violation", violations)
	}
	if got, want := violations[0].id, "internal-app-id-type:core.application.v1.GetBusinessTimeZoneRequest.app_id"; got != want {
		t.Fatalf("violation id = %q, want %q", got, want)
	}
}

// stringPointer 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
//
//go:fix inline
func stringPointer(value string) *string { return new(value) }
