package finetune

import "github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/llm"

// HardenedSystemPrompt is a defense-in-depth rewrite of llm.SystemPrompt
// that resists role-play injection, instruction override, and system prompt
// extraction. Filled in during Day 11 step 4.
//
// Until then it equals the production prompt so the attack test compiles.
var HardenedSystemPrompt = llm.SystemPrompt
