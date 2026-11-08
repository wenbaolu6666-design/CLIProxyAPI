package helps

import (
	"strings"

	"github.com/MoeForge/moego/ojson"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/config"
)

// ShouldDemoteDeveloperRoleForModel reports whether the resolved model under an
// OpenAI compatibility configuration has demote-developer-role set to true.
func ShouldDemoteDeveloperRoleForModel(compat *config.OpenAICompatibility, upstreamModel, requestedModel string) bool {
	if compat == nil {
		return false
	}
	if demote, matched := openAICompatibilityModelDemotesDeveloperRole(compat.Models, upstreamModel); matched {
		return demote
	}
	demote, _ := openAICompatibilityModelDemotesDeveloperRole(compat.Models, requestedModel)
	return demote
}

func openAICompatibilityModelDemotesDeveloperRole(models []config.OpenAICompatibilityModel, model string) (bool, bool) {
	model = normalizeOpenAICompatibilityModelName(model)
	if model == "" {
		return false, false
	}

	for i := range models {
		if strings.EqualFold(model, normalizeOpenAICompatibilityModelName(models[i].Name)) {
			return models[i].DemoteDeveloperRole, true
		}
	}

	for i := range models {
		if strings.EqualFold(model, normalizeOpenAICompatibilityModelName(models[i].Alias)) {
			return models[i].DemoteDeveloperRole, true
		}
	}
	return false, false
}

// DemoteOpenAIDeveloperRoleDoc rewrites developer messages to system messages in
// place on doc for upstreams whose Chat Completions dialect rejects the
// developer role.
func DemoteOpenAIDeveloperRoleDoc(doc *ojson.Doc) bool {
	if doc == nil || doc.Len() == 0 {
		return false
	}
	messages := doc.Get("messages")
	if !messages.Exists() || !messages.IsArray() {
		return false
	}

	modified := false
	messages.ForEach(func(msgKey, msg ojson.Result) bool {
		if msg.Get("role").String() == "developer" {
			if errSet := doc.Set("messages."+msgKey.String()+".role", "system"); errSet == nil {
				modified = true
			}
		}
		return true
	})
	return modified
}
