package agent

import (
	"context"
	"fmt"
	"time"

	openai "github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
)

// runOneToolCall 执行单次工具调用（含 tool_search、循环检测、内置/自定义工具），供阻塞式与流式执行共用。
func (e *Executor) runOneToolCall(
	ctx context.Context,
	ec *execContext,
	tc openai.ToolCall,
	toolMap map[string]Tool,
	tsMode bool,
	allToolDefs []openai.Tool,
	discovered map[string]bool,
	loopDet *toolLoopDetector,
	calledTools map[string]bool,
) (toolMsg openai.ChatCompletionMessage, tr ToolResult, fileParts []openai.ChatMessagePart) {
	toolName := tc.Function.Name
	toolArgs := tc.Function.Arguments

	if tsMode && toolName == toolSearchName {
		if blocked, guardMsg := loopDet.check(toolName, toolArgs); blocked {
			ec.l.WithField("tool", toolName).Warn("[LoopGuard] blocked tool_search")
			toolMsg = openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    guardMsg,
				ToolCallID: tc.ID,
				Name:       toolName,
			}
			return toolMsg, ToolResult{ToolCallID: tc.ID, ToolName: toolName, Content: guardMsg}, nil
		}
		loopDet.record(toolName, toolArgs)
		toolMsg = e.handleToolSearch(ctx, ec, tc, allToolDefs, discovered)
		return toolMsg, ToolResult{ToolCallID: tc.ID, ToolName: toolName, Content: toolMsg.Content}, nil
	}

	tool, ok := toolMap[toolName]
	if !ok {
		errMsg := fmt.Sprintf("tool %q not found", toolName)
		ec.l.WithField("tool", toolName).Warn("[Tool] tool not registered, skipping")
		toolMsg = openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			Content:    errMsg,
			ToolCallID: tc.ID,
			Name:       toolName,
		}
		return toolMsg, ToolResult{ToolCallID: tc.ID, ToolName: toolName, Content: errMsg}, nil
	}

	if blocked, guardMsg := loopDet.check(toolName, toolArgs); blocked {
		ec.l.WithFields(log.Fields{"tool": toolName, "args": truncateLog(toolArgs, 120)}).Warn("[LoopGuard] blocked")
		toolMsg = openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			Content:    guardMsg,
			ToolCallID: tc.ID,
			Name:       toolName,
		}
		return toolMsg, ToolResult{ToolCallID: tc.ID, ToolName: toolName, Content: guardMsg}, nil
	}
	loopDet.record(toolName, toolArgs)

	ec.l.WithFields(log.Fields{"tool": toolName, "args": truncateLog(toolArgs, 200)}).Info("[Tool] >> invoke")
	calledTools[toolName] = true
	callStart := time.Now()
	output, callErr := tool.Call(ctx, toolArgs)
	callDur := time.Since(callStart)
	toolResult := output
	if callErr != nil {
		toolResult = fmt.Sprintf("error: %s", callErr)
		ec.l.WithFields(log.Fields{"tool": toolName, "duration": callDur}).WithError(callErr).Error("[Tool] << failed")
	} else {
		ec.l.WithFields(log.Fields{"tool": toolName, "duration": callDur, "preview": truncateLog(output, 200)}).Info("[Tool] << ok")
	}

	toolMsg, fileParts = e.buildToolResponseParts(ctx, tc.ID, toolName, toolResult, callErr == nil, ec.l)
	return toolMsg, ToolResult{ToolCallID: tc.ID, ToolName: toolName, Content: toolMsg.Content}, fileParts
}

// appendAssistantToolRound 追加助手 tool_calls 消息、执行工具、持久化该轮工具结果，并在有文件时追加用户侧文件提示消息。
func (e *Executor) appendAssistantToolRound(
	ctx context.Context,
	ec *execContext,
	messages []openai.ChatCompletionMessage,
	assistant openai.ChatCompletionMessage,
	toolMap map[string]Tool,
	tsMode bool,
	allToolDefs []openai.Tool,
	discovered map[string]bool,
	loopDet *toolLoopDetector,
	calledTools map[string]bool,
) []openai.ChatCompletionMessage {
	messages = append(messages, assistant)
	var toolResults []ToolResult
	var pendingParts []openai.ChatMessagePart
	for _, tc := range assistant.ToolCalls {
		toolMsg, tr, fps := e.runOneToolCall(ctx, ec, tc, toolMap, tsMode, allToolDefs, discovered, loopDet, calledTools)
		messages = append(messages, toolMsg)
		toolResults = append(toolResults, tr)
		pendingParts = append(pendingParts, fps...)
	}
	if err := e.memory.SaveToolCallRound(ctx, ec.conv.ID, assistant.Content, assistant.ToolCalls, toolResults); err != nil {
		ec.l.WithError(err).Warn("[Memory] save tool call round failed")
	}
	if len(pendingParts) > 0 {
		parts := append([]openai.ChatMessagePart{
			{Type: openai.ChatMessagePartTypeText, Text: "工具返回了以下文件:"},
		}, pendingParts...)
		messages = append(messages, openai.ChatCompletionMessage{
			Role:         openai.ChatMessageRoleUser,
			MultiContent: parts,
		})
	}
	return messages
}
