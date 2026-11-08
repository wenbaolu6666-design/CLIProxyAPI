package executor

import (
	"context"
	"github.com/MoeForge/moego/ojson"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v8/internal/config"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v8/sdk/translator"
)

func TestOpenAICompatExecutorDemotesDeveloperRoleWhenConfigured(t *testing.T) {
	bodies := make(chan []byte, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, errRead := io.ReadAll(request.Body)
		if errRead != nil {
			http.Error(writer, "cannot read request", http.StatusBadRequest)
			return
		}
		bodies <- body
		writer.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(writer, `{"id":"chatcmpl-test","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`)
	}))
	defer server.Close()
	cfg := &config.Config{
		OpenAICompatibility: []config.OpenAICompatibility{{
			Name:    "test",
			BaseURL: server.URL + "/v1",
			Models: []config.OpenAICompatibilityModel{{
				Name:               "test-model",
				DemoteDeveloperRole: true,
			}},
		}},
	}
	executor := NewOpenAICompatExecutor("openai-compatibility", cfg)
	auth := &cliproxyauth.Auth{
		Provider:   "openai-compatibility",
		Attributes: map[string]string{"source": "config:test", "base_url": server.URL + "/v1", "api_key": "test-key", "config_index": "0"},
	}
	request := cliproxyexecutor.Request{
		Model: "test-model",
		Payload: []byte(`{"instructions":"base instructions","input":[
			{"role":"developer","content":"mode: plan"},
			{"role":"user","content":"hello"}
		]}`),
	}
	options := cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatOpenAIResponse, ResponseFormat: sdktranslator.FormatOpenAIResponse}
	if _, errExec := executor.Execute(context.Background(), auth, request, options); errExec != nil {
		t.Fatal(errExec)
	}
	body := <-bodies
	messages := ojson.GetBytes(body, "messages").Array()
	wantRoles := []string{"system", "system", "user"}
	if len(messages) != len(wantRoles) {
		t.Fatalf("message count = %d, want %d; body=%s", len(messages), len(wantRoles), body)
	}
	for index, message := range messages {
		if role := message.Get("role").String(); role != wantRoles[index] {
			t.Errorf("message %d role = %q, want %q", index, role, wantRoles[index])
		}
	}
}
