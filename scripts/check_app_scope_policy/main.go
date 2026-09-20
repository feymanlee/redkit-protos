package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// appScopedContexts is bounded-context policy, not a mirrored contract inventory.
// Adding a bounded context requires an architecture decision; adding packages,
// services, or RPCs within one of these contexts requires no checker update.
var appScopedContexts = []string{"user", "wallet", "payment", "gift", "support", "ops"}

// Core still contains legacy platform APIs that are not App-scoped. Keep its
// new business-facing packages explicit until those older contracts retire.
var appScopedPackages = []string{"core.application"}

type violation struct {
	id      string
	message string
}

// main 作为 main 的进程入口，负责在运行边界前完成检查或装配。
func main() {
	descriptorSetPath := flag.String("descriptor-set", "", "FileDescriptorSet to inspect")
	baselinePath := flag.String("baseline", "", "Accepted violation IDs during migration")
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

	baseline, err := readBaseline(*baselinePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	violations, staleBaseline := compareBaseline(inspect(descriptorSet), baseline)
	if len(violations) == 0 && len(staleBaseline) == 0 {
		return
	}
	for _, item := range violations {
		fmt.Fprintf(os.Stderr, "App Scope policy violation: %s [%s]\n", item.message, item.id)
	}
	for _, id := range staleBaseline {
		fmt.Fprintf(os.Stderr, "App Scope policy baseline is stale: %s\n", id)
	}
	os.Exit(1)
}

// readDescriptorSet 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
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

// readBaseline 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func readBaseline(path string) (map[string]bool, error) {
	result := make(map[string]bool)
	if path == "" {
		return result, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open App Scope policy baseline: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if result[line] {
			return nil, fmt.Errorf(
				"App Scope policy baseline contains duplicate entry on line %d: %s",
				lineNumber,
				line,
			)
		}
		result[line] = true
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read App Scope policy baseline: %w", err)
	}
	return result, nil
}

// compareBaseline 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func compareBaseline(violations []violation, baseline map[string]bool) ([]violation, []string) {
	current := make(map[string]bool, len(violations))
	newViolations := make([]violation, 0, len(violations))
	for _, item := range violations {
		current[item.id] = true
		if !baseline[item.id] {
			newViolations = append(newViolations, item)
		}
	}
	var stale []string
	for id := range baseline {
		if !current[id] {
			stale = append(stale, id)
		}
	}
	sort.Strings(stale)
	return newViolations, stale
}

// inspect 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func inspect(descriptorSet *descriptorpb.FileDescriptorSet) []violation {
	messages := indexMessages(descriptorSet)
	violations := make(map[string]violation)
	for _, file := range descriptorSet.GetFile() {
		switch {
		case isAdminPackage(file.GetPackage()):
			for _, service := range file.GetService() {
				for _, method := range service.GetMethod() {
					if !isAdminBusinessMethod(file.GetName(), service.GetName(), method) {
						continue
					}
					request := messages[method.GetInputType()]
					if request == nil {
						continue
					}
					for _, path := range appIdentityPaths(method.GetInputType(), request, messages) {
						requestName := strings.TrimPrefix(method.GetInputType(), ".")
						message := "admin business request " + requestName + " exposes App identity at " + path
						if !strings.Contains(path, ".") {
							message = "admin business request " + requestName + " exposes App identity field " + path
						}
						item := violation{
							id:      "admin-request-app-id:" + requestName + "." + path,
							message: message,
						}
						violations[item.id] = item
					}
					for _, binding := range httpBindings(method) {
						if !containsAppIDVariable(binding.path) {
							continue
						}
						item := violation{
							id:      "admin-http-app-id:" + file.GetPackage() + "." + service.GetName() + "." + method.GetName() + ":" + binding.verb + ":" + binding.path,
							message: "admin HTTP binding " + binding.verb + " " + binding.path + " exposes App identity",
						}
						violations[item.id] = item
					}
				}
			}
		case isBusinessPackage(file.GetPackage()):
			for _, service := range file.GetService() {
				for _, method := range service.GetMethod() {
					request := messages[method.GetInputType()]
					if request == nil {
						continue
					}
					requestName := strings.TrimPrefix(method.GetInputType(), ".")
					for _, item := range inspectInternalRequest(requestName, request, messages) {
						violations[item.id] = item
					}
				}
			}
		}
	}
	for fullName, message := range messages {
		name := strings.TrimPrefix(fullName, ".")
		if strings.HasPrefix(fullName, ".common.") && message.GetName() == "AppScope" {
			item := violation{
				id:      "shallow-app-scope:" + name,
				message: "shared message " + name + " is a shallow App Scope contract",
			}
			violations[item.id] = item
		}
		if !strings.HasPrefix(fullName, ".common.pagination.") {
			continue
		}
		for _, field := range message.GetField() {
			if !isAppIdentityField(field) {
				continue
			}
			item := violation{
				id:      "pagination-app-scope:" + name + "." + field.GetName(),
				message: "pagination/filter contract " + name + " exposes App Scope field " + field.GetName(),
			}
			violations[item.id] = item
		}
	}

	result := make([]violation, 0, len(violations))
	for _, item := range violations {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].id < result[j].id })
	return result
}

// inspectInternalRequest 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func inspectInternalRequest(
	requestName string,
	request *descriptorpb.DescriptorProto,
	messages map[string]*descriptorpb.DescriptorProto,
) []violation {
	var violations []violation
	var walk func(*descriptorpb.DescriptorProto, []string, map[string]bool)
	walk = func(message *descriptorpb.DescriptorProto, path []string, ancestors map[string]bool) {
		for _, field := range message.GetField() {
			fieldPath := append(append([]string(nil), path...), field.GetName())
			if isAppIdentityField(field) {
				switch {
				case len(path) > 0:
					joinedPath := strings.Join(fieldPath, ".")
					violations = append(violations, violation{
						id:      "internal-nested-app-id:" + requestName + "." + joinedPath,
						message: "internal request " + requestName + " contains nested App identity at " + joinedPath,
					})
				case !isCanonicalAppID(field):
					violations = append(violations, violation{
						id:      "internal-app-id-type:" + requestName + "." + field.GetName(),
						message: "internal request " + requestName + " field " + field.GetName() + " must use top-level common.v1.AppId",
					})
				}
			}

			if field.GetType() != descriptorpb.FieldDescriptorProto_TYPE_MESSAGE || ancestors[field.GetTypeName()] {
				continue
			}
			nested := messages[field.GetTypeName()]
			if nested == nil {
				continue
			}
			ancestors[field.GetTypeName()] = true
			walk(nested, fieldPath, ancestors)
			delete(ancestors, field.GetTypeName())
		}
	}
	walk(request, nil, map[string]bool{"." + requestName: true})
	return violations
}

// appIdentityPaths 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func appIdentityPaths(
	requestType string,
	request *descriptorpb.DescriptorProto,
	messages map[string]*descriptorpb.DescriptorProto,
) []string {
	var paths []string
	var walk func(*descriptorpb.DescriptorProto, []string, map[string]bool)
	walk = func(message *descriptorpb.DescriptorProto, path []string, ancestors map[string]bool) {
		for _, field := range message.GetField() {
			fieldPath := append(append([]string(nil), path...), field.GetName())
			if isAppIdentityField(field) {
				paths = append(paths, strings.Join(fieldPath, "."))
			}
			if field.GetType() != descriptorpb.FieldDescriptorProto_TYPE_MESSAGE || ancestors[field.GetTypeName()] {
				continue
			}
			nested := messages[field.GetTypeName()]
			if nested == nil {
				continue
			}
			ancestors[field.GetTypeName()] = true
			walk(nested, fieldPath, ancestors)
			delete(ancestors, field.GetTypeName())
		}
	}
	walk(request, nil, map[string]bool{requestType: true})
	return paths
}

// isAppIdentityField 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func isAppIdentityField(field *descriptorpb.FieldDescriptorProto) bool {
	return field.GetName() == "app_id" ||
		field.GetTypeName() == ".common.v1.AppId" ||
		field.GetTypeName() == ".common.v1.AppScope"
}

// isCanonicalAppID 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func isCanonicalAppID(field *descriptorpb.FieldDescriptorProto) bool {
	return field.GetName() == "app_id" &&
		field.GetType() == descriptorpb.FieldDescriptorProto_TYPE_ENUM &&
		field.GetTypeName() == ".common.v1.AppId"
}

// isBusinessPackage 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func isBusinessPackage(packageName string) bool {
	for _, packagePrefix := range appScopedPackages {
		if packageName == packagePrefix || strings.HasPrefix(packageName, packagePrefix+".") {
			return true
		}
	}
	for _, contextName := range appScopedContexts {
		if packageName == contextName || strings.HasPrefix(packageName, contextName+".") {
			return true
		}
	}
	return false
}

// isAdminPackage 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func isAdminPackage(packageName string) bool {
	return packageName == "admin" || strings.HasPrefix(packageName, "admin.")
}

type httpBinding struct {
	verb string
	path string
}

// httpBindings 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func httpBindings(method *descriptorpb.MethodDescriptorProto) []httpBinding {
	options := method.GetOptions()
	if options == nil || !proto.HasExtension(options, annotations.E_Http) {
		return nil
	}
	rule, ok := proto.GetExtension(options, annotations.E_Http).(*annotations.HttpRule)
	if !ok || rule == nil {
		return nil
	}
	var result []httpBinding
	var collect func(*annotations.HttpRule)
	collect = func(current *annotations.HttpRule) {
		if binding, ok := httpRuleBinding(current); ok {
			result = append(result, binding)
		}
		for _, additional := range current.GetAdditionalBindings() {
			collect(additional)
		}
	}
	collect(rule)
	return result
}

// httpRuleBinding 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func httpRuleBinding(rule *annotations.HttpRule) (httpBinding, bool) {
	switch {
	case rule.GetGet() != "":
		return httpBinding{verb: "GET", path: rule.GetGet()}, true
	case rule.GetPost() != "":
		return httpBinding{verb: "POST", path: rule.GetPost()}, true
	case rule.GetPut() != "":
		return httpBinding{verb: "PUT", path: rule.GetPut()}, true
	case rule.GetPatch() != "":
		return httpBinding{verb: "PATCH", path: rule.GetPatch()}, true
	case rule.GetDelete() != "":
		return httpBinding{verb: "DELETE", path: rule.GetDelete()}, true
	case rule.GetCustom() != nil:
		return httpBinding{verb: strings.ToUpper(rule.GetCustom().GetKind()), path: rule.GetCustom().GetPath()}, true
	default:
		return httpBinding{}, false
	}
}

// containsAppIDVariable 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func containsAppIDVariable(path string) bool {
	for offset := 0; ; {
		start := strings.IndexByte(path[offset:], '{')
		if start < 0 {
			return false
		}
		start += offset
		end := strings.IndexByte(path[start+1:], '}')
		if end < 0 {
			return false
		}
		end += start + 1
		variable, _, _ := strings.Cut(path[start+1:end], "=")
		parts := strings.Split(variable, ".")
		if parts[len(parts)-1] == "app_id" {
			return true
		}
		offset = end + 1
	}
}

// indexMessages 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func indexMessages(descriptorSet *descriptorpb.FileDescriptorSet) map[string]*descriptorpb.DescriptorProto {
	result := make(map[string]*descriptorpb.DescriptorProto)
	for _, file := range descriptorSet.GetFile() {
		prefix := "." + file.GetPackage()
		for _, message := range file.GetMessageType() {
			indexMessage(result, prefix, message)
		}
	}
	return result
}

// indexMessage 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func indexMessage(
	result map[string]*descriptorpb.DescriptorProto,
	prefix string,
	message *descriptorpb.DescriptorProto,
) {
	name := prefix + "." + message.GetName()
	result[name] = message
	for _, nested := range message.GetNestedType() {
		indexMessage(result, name, nested)
	}
}

// isAdminBusinessMethod 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func isAdminBusinessMethod(fileName, serviceName string, method *descriptorpb.MethodDescriptorProto) bool {
	baseName := path.Base(fileName)
	stem := strings.TrimSuffix(baseName, path.Ext(baseName))
	for _, contextName := range appScopedContexts {
		if stem == contextName || strings.HasPrefix(stem, contextName+"_") {
			return true
		}
	}
	values := []string{serviceName, method.GetInputType(), method.GetOutputType()}
	for _, value := range values {
		lower := strings.ToLower(value)
		for _, contextName := range appScopedContexts {
			if strings.Contains(lower, contextName) {
				return true
			}
		}
	}
	return false
}
