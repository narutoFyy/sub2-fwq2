package apicompat

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

const fallbackFixture = `{"type":"fallback","from":{"model":"claude-opus-5-5"},"to":{"model":"claude-opus-4-8"},"trigger":{"type":"refusal","category":"cyber","future_field":true},"future_metadata":{"value":[1,2]}}`

func TestAnthropicFallbackRoundTrip(t *testing.T) {
	var fallback AnthropicContentBlock
	require.NoError(t, json.Unmarshal([]byte(fallbackFixture), &fallback))
	blocks := []AnthropicContentBlock{
		{Type: "thinking", Signature: "signed-55"},
		{Type: "text", Text: "progress before switch"},
		fallback,
		{Type: "thinking", Thinking: "fallback progress", Signature: "signed-48"},
		{Type: "redacted_thinking", Data: "opaque-redacted"},
		{Type: "tool_use", ID: "toolu_1", Name: "lookup", Input: json.RawMessage(`{}`)},
	}
	for _, streaming := range []bool{false, true} {
		name := "buffered"
		if streaming {
			name = "streaming"
		}
		t.Run(name, func(t *testing.T) {
			var output []ResponsesOutput
			if streaming {
				state := NewAnthropicEventToResponsesState()
				state.PreserveThinkingSignatures = true
				AnthropicEventToResponsesEvents(&AnthropicStreamEvent{Type: "message_start", Message: &AnthropicResponse{ID: "msg_test", Model: "claude-opus-4-8"}}, state)
				var emitted []ResponsesOutput
				for i, block := range blocks {
					start := block
					if block.Type == "thinking" {
						start.Thinking, start.Signature = "", ""
					}
					events := AnthropicEventToResponsesEvents(&AnthropicStreamEvent{Type: "content_block_start", Index: &i, ContentBlock: &start}, state)
					if block.Type == "thinking" {
						events = append(events, AnthropicEventToResponsesEvents(&AnthropicStreamEvent{Type: "content_block_delta", Index: &i, Delta: &AnthropicDelta{Type: "thinking_delta", Thinking: block.Thinking}}, state)...)
						events = append(events, AnthropicEventToResponsesEvents(&AnthropicStreamEvent{Type: "content_block_delta", Index: &i, Delta: &AnthropicDelta{Type: "signature_delta", Signature: block.Signature}}, state)...)
					}
					if block.Type == "text" {
						events = append(events, AnthropicEventToResponsesEvents(&AnthropicStreamEvent{Type: "content_block_delta", Index: &i, Delta: &AnthropicDelta{Type: "text_delta", Text: block.Text}}, state)...)
					}
					events = append(events, AnthropicEventToResponsesEvents(&AnthropicStreamEvent{Type: "content_block_stop", Index: &i}, state)...)
					for _, event := range events {
						if event.Type == "response.output_item.done" {
							emitted = append(emitted, *event.Item)
						}
					}
				}
				AnthropicEventToResponsesEvents(&AnthropicStreamEvent{Type: "message_stop"}, state)
				output = state.Outputs
				require.Equal(t, output, emitted)
			} else {
				output = AnthropicToResponsesResponse(&AnthropicResponse{Model: "claude-opus-4-8", Content: blocks}).Output
			}
			require.Len(t, output, len(blocks))
			require.Contains(t, output[2].EncryptedContent, anthropicFallbackEnvelopePrefix)
			raw, err := json.Marshal(output)
			require.NoError(t, err)
			var history []ResponsesInputItem
			require.NoError(t, json.Unmarshal(raw, &history))
			history = append(history, ResponsesInputItem{Type: "function_call_output", CallID: output[5].CallID, Output: "ok"})
			raw, err = json.Marshal(history)
			require.NoError(t, err)
			replayed, err := ResponsesToAnthropicRequest(&ResponsesRequest{Model: "claude-opus-5-5", Input: raw})
			require.NoError(t, err)
			require.Len(t, replayed.Messages, 2)
			var got []json.RawMessage
			require.NoError(t, json.Unmarshal(replayed.Messages[0].Content, &got))
			require.Len(t, got, len(blocks))
			for i, block := range blocks {
				want, err := json.Marshal(block)
				require.NoError(t, err)
				require.JSONEq(t, string(want), string(got[i]))
			}
		})
	}
}

func TestAnthropicFallbackEnvelopeRejectsInvalidMarkers(t *testing.T) {
	for _, raw := range []string{`{}`, `{"type":"thinking","signature":"foreign"}`, `{"type":"fallback","from":{"model":"a"}}`, `{"type":"fallback","from":4,"to":{"model":"b"}}`} {
		envelope := anthropicFallbackEnvelopePrefix + base64.RawStdEncoding.EncodeToString([]byte(raw))
		input, err := json.Marshal([]ResponsesInputItem{{Type: "reasoning", EncryptedContent: envelope}})
		require.NoError(t, err)
		_, _, err = convertResponsesInputToAnthropic("", input, true)
		require.ErrorContains(t, err, "fallback")
	}
	_, err := decodeAnthropicFallback(anthropicFallbackEnvelopePrefix + "!")
	require.ErrorContains(t, err, "fallback envelope")
}
