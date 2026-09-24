package main

// 本文件校验 corevia 契约已收敛为单一 App：描述符中不得再出现 App 身份字段、App 枚举、
// App Selection 服务或 App Scope 错误原因。守卫是正向断言，不带 baseline，任何新增违规直接失败。

import (
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// appIdentityEnumNames 列出随平台收敛为单一 App 删除、且不得重新引入的枚举或消息名。
var appIdentityEnumNames = []string{"AppId", "AppScope", "AppBusinessTimeZone"}

// appSelectionServiceNames 列出随 App Selection 删除、且不得重新引入的服务名。
var appSelectionServiceNames = []string{"AppSelectionService"}

// removedErrorReasons 列出随 App Scope 删除、且不得重新引入的错误原因值。
var removedErrorReasons = []string{"APP_SCOPE_REQUIRED", "APP_SCOPE_INVALID"}

// removedAppRoleFieldNames 列出随「App 管理员角色层」或双 App 叙述删除、且不得重新引入的字段名。
var removedAppRoleFieldNames = []string{"is_app_admin", "app_code", "all_apps_ready"}

// allowedAppPrefixedFields 列出名称含 App 但与平台 App 身份无关的字段，
// 其中 store_app_id 与 channel_app_id 标识外部 Store / 支付渠道自有的应用身份。
var allowedAppPrefixedFields = map[string]bool{
	"store_app_id":   true,
	"channel_app_id": true,
}

// allowedAppNameFields 列出唯一允许保留 app_name 的消息全名：客户端应用名，不是平台 App 名称。
var allowedAppNameFields = map[string]bool{
	".core.audit.v1.DeviceInfo": true,
}

// violation 描述一条破坏单 App 不变式的契约位置。
type violation struct {
	location string
	message  string
}

// main 作为 main 的进程入口，负责在运行边界前完成检查或装配。
func main() {
	descriptorSetPath := flag.String("descriptor-set", "", "FileDescriptorSet to inspect")
	flag.Parse()

	if *descriptorSetPath == "" {
		fmt.Fprintln(os.Stderr, "-descriptor-set is required")
		os.Exit(2)
	}

	descriptorSet, err := readDescriptorSet(*descriptorSetPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	violations := inspect(descriptorSet)
	if len(violations) == 0 {
		return
	}
	for _, item := range violations {
		fmt.Fprintf(os.Stderr, "Single App policy violation: %s [%s]\n", item.message, item.location)
	}
	os.Exit(1)
}

// readDescriptorSet 读取并解码由 buf build --as-file-descriptor-set 生成的描述符集合。
func readDescriptorSet(path string) (*descriptorpb.FileDescriptorSet, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read descriptor set: %w", err)
	}
	descriptorSet := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(contents, descriptorSet); err != nil {
		return nil, fmt.Errorf("decode descriptor set: %w", err)
	}
	return descriptorSet, nil
}

// inspect 扫描描述符集合并返回全部破坏单 App 不变式的位置。
func inspect(descriptorSet *descriptorpb.FileDescriptorSet) []violation {
	var violations []violation
	for _, file := range descriptorSet.GetFile() {
		packageName := "." + file.GetPackage()
		for _, enum := range file.GetEnumType() {
			violations = append(violations, inspectEnum(packageName, enum)...)
		}
		for _, service := range file.GetService() {
			violations = append(violations, inspectService(packageName, service)...)
		}
		for _, message := range file.GetMessageType() {
			violations = append(violations, inspectMessage(packageName, message)...)
		}
	}
	return violations
}

// inspectEnum 拒绝重新引入的 App 身份枚举、其值以及 App Scope 错误原因。
func inspectEnum(packageName string, enum *descriptorpb.EnumDescriptorProto) []violation {
	fullName := packageName + "." + enum.GetName()
	var violations []violation
	for _, forbidden := range appIdentityEnumNames {
		if enum.GetName() == forbidden {
			violations = append(violations, violation{
				location: fullName,
				message:  "App identity enum " + fullName + " is no longer part of the contract",
			})
		}
	}
	for _, value := range enum.GetValue() {
		for _, forbidden := range removedErrorReasons {
			if value.GetName() == forbidden {
				violations = append(violations, violation{
					location: fullName + "." + value.GetName(),
					message:  "removed App scope error reason " + value.GetName() + " must not return",
				})
			}
		}
	}
	return violations
}

// inspectService 拒绝重新引入的 App Selection 服务。
func inspectService(packageName string, service *descriptorpb.ServiceDescriptorProto) []violation {
	fullName := packageName + "." + service.GetName()
	var violations []violation
	for _, forbidden := range appSelectionServiceNames {
		if service.GetName() == forbidden {
			violations = append(violations, violation{
				location: fullName,
				message:  "App Selection service " + fullName + " is no longer part of the contract",
			})
		}
	}
	return violations
}

// inspectMessage 递归检查消息字段，拒绝 App 身份字段与 App 枚举类型的字段。
func inspectMessage(packageName string, message *descriptorpb.DescriptorProto) []violation {
	fullName := packageName + "." + message.GetName()
	var violations []violation
	for _, forbidden := range appIdentityEnumNames {
		if message.GetName() == forbidden {
			violations = append(violations, violation{
				location: fullName,
				message:  "App identity message " + fullName + " is no longer part of the contract",
			})
		}
	}
	for _, field := range message.GetField() {
		violations = append(violations, inspectField(fullName, field)...)
	}
	for _, nested := range message.GetNestedType() {
		violations = append(violations, inspectMessage(fullName, nested)...)
	}
	for _, enum := range message.GetEnumType() {
		violations = append(violations, inspectEnum(fullName, enum)...)
	}
	return violations
}

// inspectField 拒绝 App 身份字段，并保留与平台 App 身份无关的 app_* 字段。
func inspectField(messageName string, field *descriptorpb.FieldDescriptorProto) []violation {
	fieldName := field.GetName()
	location := messageName + "." + fieldName
	typeName := strings.TrimPrefix(field.GetTypeName(), ".")

	if typeName == "common.v1.AppId" {
		return []violation{{
			location: location,
			message:  "field " + location + " still uses the removed common.v1.AppId enum",
		}}
	}
	if slices.Contains(removedAppRoleFieldNames, fieldName) {
		return []violation{{
			location: location,
			message:  "field " + location + " belongs to the removed App administrator / dual-App layer",
		}}
	}
	if !strings.Contains(fieldName, "app_id") && !strings.Contains(fieldName, "app_name") {
		return nil
	}
	if allowedAppPrefixedFields[fieldName] {
		return nil
	}
	if fieldName == "app_name" && allowedAppNameFields[messageName] {
		return nil
	}
	return []violation{{
		location: location,
		message:  "field " + location + " still carries platform App identity",
	}}
}
