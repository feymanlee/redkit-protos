package main

// 本文件在调用 OpenAPI 生成器前扁平化 Admin 私有 package，保持管理端生成契约使用稳定的 admin.v1 命名空间。

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// main 作为 main 的进程入口，负责在运行边界前完成检查或装配。
func main() {
	input, err := io.ReadAll(os.Stdin)
	check(err)

	request := &pluginpb.CodeGeneratorRequest{}
	check(proto.Unmarshal(input, request))
	options := parseOptions(request.GetParameter())
	if service := options["service"]; service != "" {
		filterServiceFiles(request, service)
		if options["filename"] == "" {
			options["filename"] = service + ".yaml"
		}
	}
	flattenAdminPackages(request)
	request.Parameter = proto.String(strings.Join(optionsWithoutCustom(options), ","))

	normalized, err := proto.Marshal(request)
	check(err)
	command := exec.Command("protoc-gen-openapi")
	command.Stdin = bytes.NewReader(normalized)
	command.Stderr = os.Stderr
	output, err := command.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			_, _ = os.Stderr.Write(exitErr.Stderr)
		}
		check(err)
	}

	response := &pluginpb.CodeGeneratorResponse{}
	check(proto.Unmarshal(output, response))
	if filename := options["filename"]; filename != "" {
		for _, file := range response.File {
			if file.GetName() == "openapi.yaml" {
				file.Name = proto.String(filename)
			}
		}
	}
	encoded, err := proto.Marshal(response)
	check(err)
	_, err = os.Stdout.Write(encoded)
	check(err)
}

// parseOptions 解析 Buf 传入的插件选项，并保留底层 OpenAPI 插件需要的其他选项。
func parseOptions(parameter string) map[string]string {
	options := make(map[string]string)
	for _, item := range strings.Split(parameter, ",") {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		options[key] = value
	}
	return options
}

// optionsWithoutCustom 移除包装器专用选项，避免透传给 protoc-gen-openapi。
func optionsWithoutCustom(options map[string]string) []string {
	keys := make([]string, 0, len(options))
	for key := range options {
		if key == "service" || key == "filename" {
			continue
		}
		keys = append(keys, key+"="+options[key])
	}
	return keys
}

// filterServiceFiles 仅让底层生成器处理指定 Admin 服务的契约文件，依赖描述仍保留在请求中。
func filterServiceFiles(request *pluginpb.CodeGeneratorRequest, service string) {
	prefix := "admin/" + service + "/"
	files := request.FileToGenerate[:0]
	for _, name := range request.FileToGenerate {
		if strings.HasPrefix(name, prefix) {
			files = append(files, name)
		}
	}
	if len(files) == 0 {
		check(fmt.Errorf("no admin proto files found for service %q", service))
	}
	request.FileToGenerate = files
}

// flattenAdminPackages 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func flattenAdminPackages(request *pluginpb.CodeGeneratorRequest) {
	for _, file := range request.GetProtoFile() {
		packageName := file.GetPackage()
		if !strings.HasPrefix(file.GetName(), "admin/") || !isFunctionalAdminPackage(packageName) {
			continue
		}
		from := "." + packageName + "."
		to := ".admin.v1."
		for _, message := range file.GetMessageType() {
			rewriteMessage(message, from, to)
		}
		for _, extension := range file.GetExtension() {
			rewriteField(extension, from, to)
		}
		for _, service := range file.GetService() {
			for _, method := range service.GetMethod() {
				method.InputType = rewriteName(method.InputType, from, to)
				method.OutputType = rewriteName(method.OutputType, from, to)
			}
		}
		file.Package = new("admin.v1")
	}
}

// rewriteMessage 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func rewriteMessage(message *descriptorpb.DescriptorProto, from, to string) {
	for _, field := range message.GetField() {
		rewriteField(field, from, to)
	}
	for _, extension := range message.GetExtension() {
		rewriteField(extension, from, to)
	}
	for _, nested := range message.GetNestedType() {
		rewriteMessage(nested, from, to)
	}
}

// rewriteField 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func rewriteField(field *descriptorpb.FieldDescriptorProto, from, to string) {
	field.TypeName = rewriteName(field.TypeName, from, to)
	field.Extendee = rewriteName(field.Extendee, from, to)
}

// rewriteName 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func rewriteName(value *string, from, to string) *string {
	if value == nil || !strings.HasPrefix(*value, from) {
		return value
	}
	rewritten := to + strings.TrimPrefix(*value, from)
	return &rewritten
}

// isFunctionalAdminPackage 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func isFunctionalAdminPackage(packageName string) bool {
	parts := strings.Split(packageName, ".")
	return len(parts) == 3 && parts[0] == "admin" && parts[2] == "v1"
}

// check 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
