package agent

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

const (
	toolSearchName       = "tool_search"
	toolSearchMaxResults = 5
)

type toolSearchResult struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func toolSearchDef() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        toolSearchName,
			Description: "搜索可用工具。当你需要完成某项任务但不确定有哪些工具可用时，先调用此工具搜索。系统会将匹配的工具自动加入你的可用工具列表，随后你可以直接调用它们。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "搜索关键词，可以是工具名称、功能描述、任务目标等",
					},
				},
				"required": []string{"query"},
			},
		},
	}
}

// searchTools performs keyword-based scoring on all tool definitions.
// Name matches are weighted higher than description matches.
func searchTools(query string, allDefs []openai.Tool) []toolSearchResult {
	query = strings.ToLower(query)
	keywords := strings.Fields(query)
	if len(keywords) == 0 {
		return nil
	}

	type scored struct {
		name  string
		desc  string
		score int
	}

	var hits []scored
	for _, def := range allDefs {
		if def.Function == nil {
			continue
		}
		name := strings.ToLower(def.Function.Name)
		desc := strings.ToLower(def.Function.Description)

		score := 0
		for _, kw := range keywords {
			if strings.EqualFold(def.Function.Name, kw) {
				score += 5
			} else if strings.Contains(name, kw) {
				score += 3
			}
			if strings.Contains(desc, kw) {
				score += 1
			}
		}

		if score > 0 {
			hits = append(hits, scored{name: def.Function.Name, desc: def.Function.Description, score: score})
		}
	}

	slices.SortFunc(hits, func(a, b scored) int {
		return cmp.Compare(b.score, a.score)
	})

	limit := min(toolSearchMaxResults, len(hits))
	out := make([]toolSearchResult, limit)
	for i := range limit {
		out[i] = toolSearchResult{Name: hits[i].name, Description: hits[i].desc}
	}
	return out
}

func formatToolSearchResults(results []toolSearchResult, totalTools int) string {
	if len(results) == 0 {
		return fmt.Sprintf("未找到匹配的工具。当前共有 %d 个工具可用，请尝试更宽泛的关键词重新搜索。", totalTools)
	}
	resp := struct {
		FoundTools []toolSearchResult `json:"found_tools"`
		Message    string             `json:"message"`
	}{
		FoundTools: results,
		Message:    fmt.Sprintf("找到 %d 个相关工具，已自动加入可用工具列表，你现在可以直接调用它们。", len(results)),
	}
	data, _ := json.Marshal(resp)
	return string(data)
}

// preloadSkillTools pre-populates the discovered set with tools that are
// explicitly associated with skills, so skill-dependent tools are always
// available without requiring a search round-trip.
func preloadSkillTools(toolSkillMap map[string]string, discovered map[string]bool) {
	for toolName := range toolSkillMap {
		discovered[toolName] = true
	}
}

// buildToolSearchDefs creates the LLM-visible tool list for tool search mode:
// always includes tool_search itself, plus any previously discovered tools.
func buildToolSearchDefs(allDefs []openai.Tool, discovered map[string]bool) []openai.Tool {
	result := make([]openai.Tool, 0, 1+len(discovered))
	result = append(result, toolSearchDef())
	for _, def := range allDefs {
		if def.Function != nil && discovered[def.Function.Name] {
			result = append(result, def)
		}
	}
	return result
}
