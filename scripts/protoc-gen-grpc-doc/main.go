package main

// 本文件把非 Admin gRPC Proto 描述转换为按 bounded context 组织的 Markdown 文档。

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

type document struct {
	group string
	files []*descriptorpb.FileDescriptorProto
}

type renderer struct {
	file *descriptorpb.FileDescriptorProto
}

func main() {
	input, err := io.ReadAll(os.Stdin)
	check(err)

	request := &pluginpb.CodeGeneratorRequest{}
	check(proto.Unmarshal(input, request))

	documents := groupFiles(request)
	response := &pluginpb.CodeGeneratorResponse{
		SupportedFeatures: proto.Uint64(uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)),
	}
	for _, doc := range documents {
		content := renderDocument(doc)
		response.File = append(response.File, &pluginpb.CodeGeneratorResponse_File{
			Name:    proto.String(doc.group + ".md"),
			Content: proto.String(content),
		})
	}

	output, err := proto.Marshal(response)
	check(err)
	_, err = os.Stdout.Write(output)
	check(err)
}

// groupFiles 按 Proto 路径的 bounded context 分组，忽略 Admin HTTP 契约和仅供导入的描述文件。
func groupFiles(request *pluginpb.CodeGeneratorRequest) []document {
	grouped := make(map[string][]*descriptorpb.FileDescriptorProto)
	for _, name := range request.GetFileToGenerate() {
		parts := strings.SplitN(name, "/", 2)
		if len(parts) != 2 || parts[0] == "admin" {
			continue
		}
		for _, file := range request.GetProtoFile() {
			if file.GetName() == name {
				grouped[parts[0]] = append(grouped[parts[0]], file)
				break
			}
		}
	}

	groups := make([]string, 0, len(grouped))
	for group := range grouped {
		groups = append(groups, group)
	}
	sort.Strings(groups)

	documents := make([]document, 0, len(groups))
	for _, group := range groups {
		files := grouped[group]
		sort.Slice(files, func(i, j int) bool { return files[i].GetName() < files[j].GetName() })
		documents = append(documents, document{group: group, files: files})
	}
	return documents
}

// renderDocument 将一个 bounded context 的文件描述渲染为可审阅的 Markdown 文档。
func renderDocument(doc document) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s gRPC API\n\n", titleCase(doc.group))
	b.WriteString("本文档由 `backend/api` 中的 Proto 契约自动生成，请修改 Proto 后重新运行 `make grpc-docs`。\n\n")
	b.WriteString("## Source Files\n\n")
	for _, file := range doc.files {
		fmt.Fprintf(&b, "- `%s`\n", file.GetName())
	}

	b.WriteString("\n## Services\n")
	for _, file := range doc.files {
		r := renderer{file: file}
		for serviceIndex, service := range file.GetService() {
			fmt.Fprintf(&b, "\n### `%s.%s`\n\n", file.GetPackage(), service.GetName())
			writeComment(&b, r.comment([]int{6, serviceIndex}))
			for methodIndex, method := range service.GetMethod() {
				fmt.Fprintf(&b, "\n#### `%s`\n\n", method.GetName())
				writeComment(&b, r.comment([]int{6, serviceIndex, 2, methodIndex}))
				streaming := ""
				switch {
				case method.GetClientStreaming() && method.GetServerStreaming():
					streaming = ", 双向流"
				case method.GetClientStreaming():
					streaming = ", Client streaming"
				case method.GetServerStreaming():
					streaming = ", Server streaming"
				}
				fmt.Fprintf(&b, "`rpc %s(%s) returns (%s)`%s\n", method.GetName(), shortType(method.GetInputType()), shortType(method.GetOutputType()), streaming)
			}
		}
	}

	b.WriteString("\n## Messages\n")
	for _, file := range doc.files {
		r := renderer{file: file}
		for messageIndex, message := range file.GetMessageType() {
			r.writeMessage(&b, message, message.GetName(), []int{4, messageIndex})
		}
	}

	b.WriteString("\n## Enums\n")
	for _, file := range doc.files {
		r := renderer{file: file}
		for enumIndex, enum := range file.GetEnumType() {
			fmt.Fprintf(&b, "\n### `%s`\n\n", enum.GetName())
			writeComment(&b, r.comment([]int{5, enumIndex}))
			b.WriteString("| Value | Description |\n| --- | --- |\n")
			for valueIndex, value := range enum.GetValue() {
				fmt.Fprintf(&b, "| `%s` | %s |\n", value.GetName(), escapeTable(r.comment([]int{5, enumIndex, 2, valueIndex})))
			}
		}
	}
	return b.String()
}

// writeMessage 输出消息字段和嵌套消息，保留字段号、类型、重复性和 Proto 注释。
func (r renderer) writeMessage(b *strings.Builder, message *descriptorpb.DescriptorProto, name string, path []int) {
	fmt.Fprintf(b, "\n### `%s`\n\n", name)
	writeComment(b, r.comment(path))
	b.WriteString("| Field | Type | Cardinality | Description |\n| --- | --- | --- | --- |\n")
	for fieldIndex, field := range message.GetField() {
		fieldPath := append(append([]int{}, path...), 2, fieldIndex)
		cardinality := "optional"
		if field.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED {
			cardinality = "repeated"
		}
		fmt.Fprintf(b, "| `%s` (#%d) | `%s` | %s | %s |\n", field.GetName(), field.GetNumber(), fieldType(field), cardinality, escapeTable(r.comment(fieldPath)))
	}
	for nestedIndex, nested := range message.GetNestedType() {
		nestedPath := append(append([]int{}, path...), 3, nestedIndex)
		r.writeMessage(b, nested, name+"."+nested.GetName(), nestedPath)
	}
}

func (r renderer) comment(path []int) string {
	for _, location := range r.file.GetSourceCodeInfo().GetLocation() {
		if samePath(location.GetPath(), path) {
			if text := strings.TrimSpace(location.GetLeadingComments()); text != "" {
				return text
			}
			return strings.TrimSpace(location.GetTrailingComments())
		}
	}
	return ""
}

// samePath 比较源码位置路径，处理 protobuf 使用 int32 而渲染器使用 int 的表示差异。
func samePath(got []int32, want []int) bool {
	if len(got) != len(want) {
		return false
	}
	for i, value := range want {
		if got[i] != int32(value) {
			return false
		}
	}
	return true
}

func fieldType(field *descriptorpb.FieldDescriptorProto) string {
	if field.GetTypeName() != "" {
		return shortType(field.GetTypeName())
	}
	return strings.ToLower(strings.TrimPrefix(field.GetType().String(), "TYPE_"))
}

// shortType 去除 protobuf 类型名的全限定前缀，保留包和消息名用于文档引用。
func shortType(value string) string {
	return strings.TrimPrefix(value, ".")
}

// writeComment 将 Proto 源码注释写入 Markdown，空注释不产生额外内容。
func writeComment(b *strings.Builder, comment string) {
	if comment != "" {
		fmt.Fprintf(b, "%s\n", comment)
	}
}

// escapeTable 转义 Markdown 表格中会改变列结构的字符。
func escapeTable(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "\n", "<br>"), "|", "\\|")
}

// titleCase 将 bounded context 名称格式化为文档标题。
func titleCase(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

// check 统一处理生成器输入、编码和输出错误，并以非零状态结束进程。
func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
