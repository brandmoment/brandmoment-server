# Micro-First Benchmark Report

**Date**: 2026-04-24  
**Corpus**: 30 phrases  
**Margin floor**: 0.020

## Per-Phrase Results

| # | Phrase | Group | Expected | Micro Intent | Margin | Route | Baseline Conf | Two-Level Conf | Baseline ms | Two-Level ms | Baseline LLM | TwoLevel LLM |
|---|--------|-------|----------|--------------|--------|-------|--------------|----------------|------------|-------------|--------------|-------------|
| 1 | Block gambling category | correct | OK | valid | 0.014 | llm_with_check | OK | OK | 2131 | 5600 | 2 | 4 |
| 2 | Allow only iOS devices | correct | OK | valid | 0.081 | llm_direct | OK | OK | 2154 | 3275 | 2 | 2 |
| 3 | Show maximum 5 times per hour | correct | OK | ambiguous | 0.074 | llm_with_check | OK | OK | 2353 | 4408 | 2 | 4 |
| 4 | Exclude Russia and Kazakhstan | correct | OK | valid | 0.030 | llm_direct | OK | OK | 2043 | 2456 | 2 | 2 |
| 5 | Only show ads in the United Stat... | correct | OK | valid | 0.068 | llm_direct | OK | OK | 3258 | 2168 | 2 | 2 |
| 6 | Block domain evil.com | correct | OK | valid | 0.189 | llm_direct | OK | OK | 1949 | 2146 | 2 | 2 |
| 7 | Allow only the bundle ID com.goo... | correct | OK | valid | 0.145 | llm_direct | OK | OK | 2150 | 2561 | 2 | 2 |
| 8 | Show no more than 10 impressions... | correct | OK | ambiguous | 0.007 | llm_with_check | OK | OK | 2558 | 4765 | 2 | 4 |
| 9 | Allow Android and iOS only | correct | OK | valid | 0.080 | llm_direct | OK | OK | 2054 | 1994 | 2 | 2 |
| 10 | Block the bundle com.casino.app | correct | OK | valid | 0.118 | llm_direct | OK | OK | 2212 | 2185 | 2 | 2 |
| 11 | Exclude Germany from ad delivery | correct | OK | valid | 0.066 | llm_direct | OK | OK | 3276 | 2662 | 2 | 2 |
| 12 | Cap impressions at 3 per week | correct | OK | valid | 0.017 | llm_with_check | OK | OK | 1885 | 4543 | 2 | 4 |
| 13 | Only web and CTV platforms | correct | OK | valid | 0.142 | llm_direct | OK | OK | 2480 | 2047 | 2 | 2 |
| 14 | Block adult content domains | correct | OK | valid | 0.163 | llm_direct | OK | OK | 1945 | 4608 | 2 | 2 |
| 15 | Allow domains news.com and media... | correct | OK | valid | 0.172 | llm_direct | OK | OK | 1636 | 2050 | 2 | 2 |
| 16 | Do not show casino ads in Russia... | edge | UNSURE | ambiguous | 0.130 | llm_with_check | OK | UNSURE | 2915 | 6708 | 3 | 5 |
| 17 | Premium users only | edge | UNSURE | ambiguous | 0.073 | llm_with_check | FAIL | FAIL | 716 | 1025 | 1 | 3 |
| 18 | Allow everything except gambling | edge | UNSURE | ambiguous | 0.052 | llm_with_check | OK | OK | 1696 | 4816 | 2 | 4 |
| 19 | No more than twice a day and onl... | edge | UNSURE | ambiguous | 0.206 | llm_with_check | OK | OK | 3079 | 5972 | 3 | 5 |
| 20 | Show ads to iOS users in Germany | edge | UNSURE | valid | 0.019 | llm_with_check | OK | OK | 3174 | 6246 | 3 | 5 |
| 21 | Block competitors | edge | UNSURE | ambiguous | 0.002 | llm_with_check | OK | UNSURE | 2207 | 5164 | 2 | 4 |
| 22 | Only show once per session | edge | UNSURE | ambiguous | 0.117 | llm_with_check | OK | UNSURE | 2149 | 5374 | 2 | 4 |
| 23 | Block gambling in Russia and Ger... | edge | UNSURE | ambiguous | 0.151 | llm_with_check | OK | OK | 3433 | 5863 | 3 | 5 |
| 24 | Allow only mobile | edge | UNSURE | ambiguous | 0.016 | llm_with_check | OK | UNSURE | 2248 | 4308 | 2 | 4 |
| 25 | Limit frequency but only on week... | edge | UNSURE | ambiguous | 0.147 | llm_with_check | OK | UNSURE | 2529 | 4708 | 2 | 4 |
| 26 | kakashki | noisy | FAIL | gibberish | 0.332 | micro_early_fail | FAIL | FAIL | 819 | 114 | 1 | 0 |
| 27 | Block and allow gambling at the ... | noisy | FAIL | gibberish | 0.026 | micro_early_fail | OK | FAIL | 3271 | 122 | 3 | 0 |
| 28 | Show ads | noisy | FAIL | gibberish | 0.157 | micro_early_fail | FAIL | FAIL | 1305 | 120 | 1 | 0 |
| 29 | xyzzy foo bar 123!!! | noisy | FAIL | gibberish | 0.296 | micro_early_fail | FAIL | FAIL | 1211 | 195 | 1 | 0 |
| 30 | Do absolutely nothing with all a... | noisy | FAIL | gibberish | 0.180 | micro_early_fail | FAIL | FAIL | 1135 | 107 | 1 | 0 |

## Summary

**LLM calls saved**: 60 → 81 (-35.0% reduction)

### Route Distribution

| Route | Count | % |
|-------|-------|---|
| micro_early_fail | 5 | 17% |
| llm_direct | 11 | 37% |
| llm_with_check | 14 | 47% |

### Confidence Distribution

| Confidence | Baseline | Two-Level |
|------------|----------|----------|
| OK | 25 | 19 |
| UNSURE | 0 | 5 |
| FAIL | 5 | 6 |

### Accuracy vs expected_confidence

| Parser | Correct | Total | Accuracy |
|--------|---------|-------|----------|
| Baseline (MultiStage) | 19 | 30 | 63% |
| Two-Level (MicroFirst) | 25 | 30 | 83% |
