package config

// BuildModeJSON builds a SuperWhisper custom mode document (type custom, version 1).
func BuildModeJSON(key, name, lang, voiceModel, languageModel, prompt string) map[string]any {
	return map[string]any{
		"activationApps":                 []any{},
		"activationSites":                []any{},
		"autocapitalizeInsert":           true,
		"contextFromActiveApplication":   false,
		"contextFromClipboard":           false,
		"contextFromSelection":           false,
		"contextTemplate":                "Use the copied text as context to complete this task.\n\nCopied text: ",
		"description":                    "",
		"diarize":                        false,
		"iconName":                       "note.text",
		"key":                            key,
		"language":                       lang,
		"languageModelID":                languageModel,
		"literalPunctuation":             false,
		"name":                           name,
		"prompt":                         prompt,
		"promptExamples":                 []any{},
		"realtimeOutput":                 false,
		"script":                         "",
		"scriptEnabled":                  false,
		"translateToEnglish":             false,
		"type":                           "custom",
		"useSystemAudio":                 false,
		"version":                        1,
		"voiceModelID":                   voiceModel,
	}
}
