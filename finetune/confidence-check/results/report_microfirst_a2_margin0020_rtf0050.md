# Micro-First Benchmark Report (A2)

**Date**: 2026-04-25  
**Corpus**: 30 phrases  
**Margin floor**: 0.020  
**Rule-type floor**: 0.050

## Per-Phrase Results

| # | Phrase | Group | Expected | Micro Intent | Margin | Route | Baseline Conf | Two-Level Conf | Baseline ms | Two-Level ms | Baseline LLM | TwoLevel LLM |
|---|--------|-------|----------|--------------|--------|-------|--------------|----------------|------------|-------------|--------------|-------------|
| 1 | Block gambling category | correct | OK | blocklist | 0.196 | micro_answer | OK | OK | 2673 | 1559 | 2 | 1 |
| 2 | Allow only iOS devices | correct | OK | platform_filter | 0.261 | micro_answer | OK | OK | 2354 | 1128 | 2 | 1 |
| 3 | Show maximum 5 times per hour | correct | OK | frequency_cap | 0.243 | micro_answer | OK | OK | 2240 | 1139 | 2 | 1 |
| 4 | Exclude Russia and Kazakhstan | correct | OK | geo_filter | 0.312 | micro_answer | OK | OK | 2352 | 1435 | 2 | 1 |
| 5 | Only show ads in the United Stat... | correct | OK | geo_filter | 0.226 | micro_answer | OK | OK | 2457 | 1181 | 2 | 1 |
| 6 | Block domain evil.com | correct | OK | blocklist | 0.253 | micro_answer | OK | OK | 2095 | 946 | 2 | 1 |
| 7 | Allow only the bundle ID com.goo... | correct | OK | allowlist | 0.275 | micro_answer | OK | OK | 2228 | 1023 | 2 | 1 |
| 8 | Show no more than 10 impressions... | correct | OK | frequency_cap | 0.295 | micro_answer | OK | OK | 3685 | 1125 | 2 | 1 |
| 9 | Allow Android and iOS only | correct | OK | platform_filter | 0.313 | micro_answer | OK | OK | 2561 | 982 | 2 | 1 |
| 10 | Block the bundle com.casino.app | correct | OK | blocklist | 0.144 | micro_answer | OK | OK | 1644 | 1095 | 2 | 1 |
| 11 | Exclude Germany from ad delivery | correct | OK | geo_filter | 0.239 | micro_answer | OK | OK | 2011 | 921 | 2 | 1 |
| 12 | Cap impressions at 3 per week | correct | OK | frequency_cap | 0.422 | micro_answer | OK | OK | 1972 | 1304 | 2 | 1 |
| 13 | Only web and CTV platforms | correct | OK | platform_filter | 0.284 | micro_answer | OK | OK | 2347 | 1431 | 2 | 1 |
| 14 | Block adult content domains | correct | OK | blocklist | 0.262 | micro_answer | OK | OK | 2060 | 859 | 2 | 1 |
| 15 | Allow domains news.com and media... | correct | OK | allowlist | 0.238 | micro_answer | OK | OK | 2106 | 921 | 2 | 1 |
| 16 | Do not show casino ads in Russia... | edge | UNSURE | ambiguous | 0.160 | llm_with_check | OK | UNSURE | 4199 | 6858 | 3 | 5 |
| 17 | Premium users only | edge | UNSURE | ambiguous | 0.115 | llm_with_check | FAIL | FAIL | 709 | 747 | 1 | 3 |
| 18 | Allow everything except gambling | edge | UNSURE | ambiguous | 0.055 | llm_with_check | OK | OK | 2028 | 4194 | 2 | 4 |
| 19 | No more than twice a day and onl... | edge | UNSURE | ambiguous | 0.167 | llm_with_check | OK | OK | 3380 | 6451 | 3 | 5 |
| 20 | Show ads to iOS users in Germany | edge | UNSURE | geo_filter | 0.021 | llm_with_check | OK | OK | 3480 | 6964 | 3 | 5 |
| 21 | Block competitors | edge | UNSURE | blocklist | 0.003 | llm_with_check | OK | UNSURE | 1945 | 4504 | 2 | 4 |
| 22 | Only show once per session | edge | UNSURE | ambiguous | 0.042 | llm_with_check | OK | UNSURE | 2253 | 5836 | 2 | 4 |
| 23 | Block gambling in Russia and Ger... | edge | UNSURE | ambiguous | 0.165 | llm_with_check | OK | OK | 3378 | 5530 | 3 | 5 |
| 24 | Allow only mobile | edge | UNSURE | ambiguous | 0.015 | llm_with_check | OK | UNSURE | 1842 | 4608 | 2 | 4 |
| 25 | Limit frequency but only on week... | edge | UNSURE | ambiguous | 0.182 | llm_with_check | OK | UNSURE | 2050 | 5018 | 2 | 4 |
| 26 | kakashki | noisy | FAIL | invalid | 0.332 | micro_early_fail | FAIL | FAIL | 1023 | 139 | 1 | 0 |
| 27 | Block and allow gambling at the ... | noisy | FAIL | blocklist | 0.026 | llm_with_check | FAIL | UNSURE | 983 | 4917 | 1 | 5 |
| 28 | Show ads | noisy | FAIL | invalid | 0.179 | micro_early_fail | FAIL | FAIL | 818 | 131 | 1 | 0 |
| 29 | xyzzy foo bar 123!!! | noisy | FAIL | invalid | 0.304 | micro_early_fail | FAIL | FAIL | 789 | 134 | 1 | 0 |
| 30 | Do absolutely nothing with all a... | noisy | FAIL | invalid | 0.180 | micro_early_fail | FAIL | FAIL | 632 | 261 | 1 | 0 |

## Summary

**LLM calls saved**: 58 → 63 (-8.6% reduction)

**% handled without full pipeline** (micro_early_fail + micro_answer): 19/30 (63.3%)

### Route Distribution

| Route | Count | % |
|-------|-------|---|
| micro_early_fail | 4 | 13% |
| micro_answer | 15 | 50% |
| llm_with_check | 11 | 37% |
| llm_direct | 0 | 0% |

### Confidence Distribution

| Confidence | Baseline | Two-Level |
|------------|----------|----------|
| OK | 24 | 19 |
| UNSURE | 0 | 6 |
| FAIL | 6 | 5 |

### Accuracy vs expected_confidence

| Parser | Correct | Total | Accuracy |
|--------|---------|-------|----------|
| Baseline (MultiStage) | 20 | 30 | 67% |
| Two-Level (MicroFirst A2) | 24 | 30 | 80% |
