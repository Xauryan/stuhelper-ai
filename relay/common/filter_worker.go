package common

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	appcommon "github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/types"
)

const (
	relayFilterWorkerConfigEnv        = "RELAY_FILTER_WORKER_CONFIG"
	relayFilterWorkerEnabledEnv       = "RELAY_FILTER_WORKER_ENABLED"
	relayFilterWorkerMaxResponseMBEnv = "RELAY_FILTER_WORKER_MAX_RESPONSE_MB"

	relayFilterStageRequest        = "request"
	relayFilterStageResponse       = "response"
	relayFilterStageStreamResponse = "stream_response"
)

type relayFilterWorkerConfig struct {
	Enabled        *bool                  `json:"enabled,omitempty"`
	Request        relayFilterRuleList    `json:"request,omitempty"`
	Response       relayFilterRuleList    `json:"response,omitempty"`
	StreamResponse relayFilterRuleList    `json:"stream_response,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

type relayFilterRuleList []relayFilterRule

type relayFilterRule struct {
	Name       string                 `json:"name,omitempty"`
	Enabled    *bool                  `json:"enabled,omitempty"`
	Conditions []ConditionOperation   `json:"conditions,omitempty"`
	Logic      string                 `json:"logic,omitempty"`
	Operations []ParamOperation       `json:"operations,omitempty"`
	Override   map[string]interface{} `json:"override,omitempty"`
}

func (rules *relayFilterRuleList) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*rules = nil
		return nil
	}
	if trimmed[0] == '[' {
		var parsed []relayFilterRule
		if err := appcommon.Unmarshal(trimmed, &parsed); err != nil {
			return err
		}
		*rules = parsed
		return nil
	}
	if trimmed[0] == '{' {
		var parsed relayFilterRule
		if err := appcommon.Unmarshal(trimmed, &parsed); err != nil {
			return err
		}
		*rules = []relayFilterRule{parsed}
		return nil
	}
	return fmt.Errorf("relay filter worker rules must be object or array")
}

var relayFilterWorkerCache struct {
	sync.Mutex
	raw    string
	config *relayFilterWorkerConfig
	err    error
}

func getRelayFilterWorkerConfig() (*relayFilterWorkerConfig, error) {
	if !appcommon.GetEnvOrDefaultBool(relayFilterWorkerEnabledEnv, true) {
		return nil, nil
	}

	raw := strings.TrimSpace(appcommon.GetEnvOrDefaultString(relayFilterWorkerConfigEnv, ""))
	if raw == "" {
		return nil, nil
	}

	relayFilterWorkerCache.Lock()
	defer relayFilterWorkerCache.Unlock()
	if raw == relayFilterWorkerCache.raw {
		return relayFilterWorkerCache.config, relayFilterWorkerCache.err
	}

	var config relayFilterWorkerConfig
	err := appcommon.UnmarshalJsonStr(raw, &config)
	if err != nil {
		relayFilterWorkerCache.raw = raw
		relayFilterWorkerCache.config = nil
		relayFilterWorkerCache.err = err
		return nil, err
	}
	if config.Enabled != nil && !*config.Enabled {
		relayFilterWorkerCache.raw = raw
		relayFilterWorkerCache.config = nil
		relayFilterWorkerCache.err = nil
		return nil, nil
	}
	relayFilterWorkerCache.raw = raw
	relayFilterWorkerCache.config = &config
	relayFilterWorkerCache.err = nil
	return &config, nil
}

// ApplyRelayFilterWorkerRequest applies configured request-stage JSON workers
// after channel conversion and channel ParamOverride, immediately before the
// outbound upstream request body is created.
func ApplyRelayFilterWorkerRequest(jsonData []byte, info *RelayInfo) ([]byte, error) {
	config, err := getRelayFilterWorkerConfig()
	if err != nil || config == nil || len(config.Request) == 0 {
		return jsonData, err
	}
	return applyRelayFilterRules(relayFilterStageRequest, jsonData, info, nil, config.Request)
}

func BuildRelayFilterWorkerRequestBody(storage appcommon.BodyStorage, info *RelayInfo) (io.Reader, int64, io.Closer, error) {
	if storage == nil {
		return nil, 0, nil, fmt.Errorf("request body storage is nil")
	}
	config, err := getRelayFilterWorkerConfig()
	if err != nil {
		return nil, 0, nil, err
	}
	if config == nil || len(config.Request) == 0 {
		return appcommon.ReaderOnly(storage), storage.Size(), noopCloser{}, nil
	}
	body, err := storage.Bytes()
	if err != nil {
		return nil, 0, nil, err
	}
	if !isJSONPayload(body) {
		if _, seekErr := storage.Seek(0, io.SeekStart); seekErr != nil {
			return nil, 0, nil, seekErr
		}
		return appcommon.ReaderOnly(storage), storage.Size(), noopCloser{}, nil
	}
	filtered, err := applyRelayFilterRules(relayFilterStageRequest, body, info, nil, config.Request)
	if err != nil {
		return nil, 0, nil, err
	}
	if bytes.Equal(filtered, body) {
		if _, seekErr := storage.Seek(0, io.SeekStart); seekErr != nil {
			return nil, 0, nil, seekErr
		}
		return appcommon.ReaderOnly(storage), storage.Size(), noopCloser{}, nil
	}
	filteredStorage, err := appcommon.CreateBodyStorage(filtered)
	if err != nil {
		return nil, 0, nil, err
	}
	return appcommon.ReaderOnly(filteredStorage), filteredStorage.Size(), filteredStorage, nil
}

// ApplyRelayFilterWorkerResponse rewrites non-stream JSON response bodies and
// wraps SSE bodies so configured stream_response rules run on each JSON data
// chunk before provider adaptors read the response.
func ApplyRelayFilterWorkerResponse(info *RelayInfo, httpResp *http.Response) error {
	config, err := getRelayFilterWorkerConfig()
	if err != nil || config == nil || httpResp == nil || httpResp.Body == nil {
		return err
	}

	contentType := strings.ToLower(httpResp.Header.Get("Content-Type"))
	if strings.HasPrefix(contentType, "text/event-stream") {
		rules := config.StreamResponse
		if len(rules) == 0 {
			rules = config.Response
		}
		if len(rules) == 0 {
			return nil
		}
		httpResp.Body = newRelayFilterSSEReadCloser(httpResp.Body, info, httpResp, rules)
		return nil
	}
	if !strings.Contains(contentType, "json") || len(config.Response) == 0 {
		return nil
	}

	originalBody := httpResp.Body
	body, overLimit, readErr := readResponseBodyForRelayFilter(originalBody, relayFilterWorkerMaxResponseBytes())
	if readErr != nil {
		return readErr
	}
	if overLimit {
		httpResp.Body = &relayFilterMultiReadCloser{
			Reader: io.MultiReader(bytes.NewReader(body), originalBody),
			Closer: originalBody,
		}
		return nil
	}
	if !isJSONPayload(body) {
		_ = originalBody.Close()
		httpResp.Body = io.NopCloser(bytes.NewReader(body))
		httpResp.ContentLength = int64(len(body))
		httpResp.Header.Set("Content-Length", strconv.Itoa(len(body)))
		return nil
	}

	filtered, err := applyRelayFilterRules(relayFilterStageResponse, body, info, httpResp, config.Response)
	if err != nil {
		return err
	}
	_ = originalBody.Close()
	httpResp.Body = io.NopCloser(bytes.NewReader(filtered))
	httpResp.ContentLength = int64(len(filtered))
	httpResp.Header.Set("Content-Length", strconv.Itoa(len(filtered)))
	return nil
}

func StuHelperAIErrorFromRelayFilterWorker(err error) *types.StuHelperAIError {
	if fixedErr, ok := AsParamOverrideReturnError(err); ok {
		return StuHelperAIErrorFromParamOverride(fixedErr)
	}
	return types.NewError(err, types.ErrorCodeChannelParamOverrideInvalid, types.ErrOptionWithSkipRetry())
}

func relayFilterWorkerMaxResponseBytes() int64 {
	mb := appcommon.GetEnvOrDefault(relayFilterWorkerMaxResponseMBEnv, 16)
	if mb <= 0 {
		mb = 16
	}
	return int64(mb) << 20
}

func readResponseBodyForRelayFilter(body io.Reader, maxBytes int64) ([]byte, bool, error) {
	if maxBytes <= 0 {
		maxBytes = 16 << 20
	}
	data, err := io.ReadAll(io.LimitReader(body, maxBytes+1))
	if err != nil {
		return nil, false, err
	}
	if int64(len(data)) > maxBytes {
		return data, true, nil
	}
	return data, false, nil
}

func applyRelayFilterRules(stage string, jsonData []byte, info *RelayInfo, httpResp *http.Response, rules relayFilterRuleList) ([]byte, error) {
	if len(rules) == 0 || !isJSONPayload(jsonData) {
		return jsonData, nil
	}
	context := BuildParamOverrideContext(info)
	if context == nil {
		context = map[string]interface{}{}
	}
	context["filter_stage"] = stage
	context["relay_filter_stage"] = stage
	if httpResp != nil {
		context["response_status_code"] = httpResp.StatusCode
		context["response_content_type"] = httpResp.Header.Get("Content-Type")
		context["is_stream"] = strings.HasPrefix(strings.ToLower(httpResp.Header.Get("Content-Type")), "text/event-stream")
	}

	result := jsonData
	for _, rule := range rules {
		if rule.Enabled != nil && !*rule.Enabled {
			continue
		}
		if len(rule.Conditions) > 0 {
			contextJSON, err := marshalContextJSON(context)
			if err != nil {
				return nil, err
			}
			ok, err := checkConditions(result, contextJSON, rule.Conditions, rule.Logic)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
		}
		overrideMap, err := rule.toParamOverrideMap()
		if err != nil {
			return nil, err
		}
		if len(overrideMap) == 0 {
			continue
		}
		result, err = ApplyParamOverride(result, overrideMap, context)
		if err != nil {
			return nil, err
		}
		if stage == relayFilterStageRequest {
			syncRuntimeHeaderOverrideFromContext(info, context)
		}
		recordRelayFilterAudit(info, stage, rule.Name)
	}
	return result, nil
}

func (rule relayFilterRule) toParamOverrideMap() (map[string]interface{}, error) {
	if len(rule.Override) > 0 {
		return rule.Override, nil
	}
	if len(rule.Operations) == 0 {
		return nil, nil
	}
	raw, err := appcommon.Marshal(rule.Operations)
	if err != nil {
		return nil, err
	}
	var operations []interface{}
	if err := appcommon.Unmarshal(raw, &operations); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"operations": operations,
	}, nil
}

func recordRelayFilterAudit(info *RelayInfo, stage string, name string) {
	if info == nil {
		return
	}
	stage = strings.TrimSpace(stage)
	name = strings.TrimSpace(name)
	line := stage
	if name != "" {
		line = stage + ":" + name
	}
	for _, existing := range info.RelayFilterAudit {
		if existing == line {
			return
		}
	}
	info.RelayFilterAudit = append(info.RelayFilterAudit, line)
}

func isJSONPayload(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return false
	}
	return trimmed[0] == '{' || trimmed[0] == '['
}

type relayFilterMultiReadCloser struct {
	io.Reader
	io.Closer
}

type noopCloser struct{}

func (noopCloser) Close() error {
	return nil
}

type relayFilterSSEReadCloser struct {
	source   io.ReadCloser
	reader   *bufio.Reader
	pending  []byte
	info     *RelayInfo
	resp     *http.Response
	rules    relayFilterRuleList
	closeErr error
}

func newRelayFilterSSEReadCloser(source io.ReadCloser, info *RelayInfo, resp *http.Response, rules relayFilterRuleList) io.ReadCloser {
	return &relayFilterSSEReadCloser{
		source: source,
		reader: bufio.NewReader(source),
		info:   info,
		resp:   resp,
		rules:  rules,
	}
}

func (r *relayFilterSSEReadCloser) Read(p []byte) (int, error) {
	if len(r.pending) == 0 {
		line, err := r.reader.ReadBytes('\n')
		if len(line) > 0 {
			r.pending = r.filterLine(line)
		}
		if len(r.pending) == 0 {
			return 0, err
		}
		if err != nil && err != io.EOF {
			return 0, err
		}
	}
	n := copy(p, r.pending)
	r.pending = r.pending[n:]
	return n, nil
}

func (r *relayFilterSSEReadCloser) Close() error {
	if r.closeErr != nil {
		return r.closeErr
	}
	r.closeErr = r.source.Close()
	return r.closeErr
}

func (r *relayFilterSSEReadCloser) filterLine(line []byte) []byte {
	trimmedLine := bytes.TrimSpace(line)
	if !bytes.HasPrefix(trimmedLine, []byte("data:")) {
		return line
	}

	lineEnding := []byte{}
	content := line
	if bytes.HasSuffix(content, []byte("\r\n")) {
		lineEnding = []byte("\r\n")
		content = content[:len(content)-2]
	} else if bytes.HasSuffix(content, []byte("\n")) {
		lineEnding = []byte("\n")
		content = content[:len(content)-1]
	}

	dataIdx := bytes.Index(content, []byte("data:"))
	if dataIdx < 0 {
		return line
	}
	prefix := content[:dataIdx+len("data:")]
	payloadWithSpace := content[dataIdx+len("data:"):]
	leadingSpaceLen := len(payloadWithSpace) - len(bytes.TrimLeft(payloadWithSpace, " \t"))
	leadingSpace := payloadWithSpace[:leadingSpaceLen]
	payload := bytes.TrimSpace(payloadWithSpace)
	if len(payload) == 0 || bytes.Equal(payload, []byte("[DONE]")) || !isJSONPayload(payload) {
		return line
	}

	filtered, err := applyRelayFilterRules(relayFilterStageStreamResponse, payload, r.info, r.resp, r.rules)
	if err != nil {
		appcommon.SysError("relay filter worker stream response failed: " + err.Error())
		return line
	}

	out := make([]byte, 0, len(prefix)+len(leadingSpace)+len(filtered)+len(lineEnding))
	out = append(out, prefix...)
	out = append(out, leadingSpace...)
	out = append(out, filtered...)
	out = append(out, lineEnding...)
	return out
}
