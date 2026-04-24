# Micro-First Benchmark Report

**Date**: 2026-04-24  
**Corpus**: 30 phrases

## Per-Phrase Results

| # | Phrase | Group | Expected | Micro Intent | Margin | Route | Baseline Conf | Two-Level Conf | Baseline ms | Two-Level ms | Baseline LLM | TwoLevel LLM |
|---|--------|-------|----------|--------------|--------|-------|--------------|----------------|------------|-------------|--------------|-------------|
| 1 | Block gambling category | correct | OK | valid | 0.014 | llm_with_check | OK | OK | 3152 | 6294 | 2 | 4 |
| 2 | Allow only iOS devices | correct | OK | valid | 0.081 | llm_direct | OK | OK | 2120 | 2663 | 2 | 2 |
| 3 | Show maximum 5 times per hour | correct | OK | ambiguous | 0.074 | llm_with_check | OK | OK | 2560 | 4324 | 2 | 4 |
| 4 | Exclude Russia and Kazakhstan | correct | OK | valid | 0.030 | llm_with_check | OK | OK | 2329 | 5941 | 2 | 4 |
| 5 | Only show ads in the United Stat... | correct | OK | valid | 0.068 | llm_direct | OK | OK | 2559 | 2148 | 2 | 2 |
| 6 | Block domain evil.com | correct | OK | valid | 0.189 | llm_direct | OK | OK | 3276 | 2868 | 2 | 2 |
| 7 | Allow only the bundle ID com.goo... | correct | OK | valid | 0.145 | llm_direct | OK | OK | 2235 | 2682 | 2 | 2 |
| 8 | Show no more than 10 impressions... | correct | OK | ambiguous | 0.007 | llm_with_check | OK | OK | 2250 | 5221 | 2 | 4 |
| 9 | Allow Android and iOS only | correct | OK | valid | 0.080 | llm_direct | OK | OK | 1934 | 2366 | 2 | 2 |
| 10 | Block the bundle com.casino.app | correct | OK | valid | 0.118 | llm_direct | OK | OK | 2118 | 2898 | 2 | 2 |
| 11 | Exclude Germany from ad delivery | correct | OK | valid | 0.066 | llm_direct | OK | OK | 1638 | 2151 | 2 | 2 |
| 12 | Cap impressions at 3 per week | correct | OK | valid | 0.017 | llm_with_check | OK | OK | 1944 | 4994 | 2 | 4 |
| 13 | Only web and CTV platforms | correct | OK | valid | 0.142 | llm_direct | OK | OK | 2173 | 2004 | 2 | 2 |
| 14 | Block adult content domains | correct | OK | valid | 0.163 | llm_direct | OK | OK | 2710 | 2760 | 2 | 2 |
| 15 | Allow domains news.com and media... | correct | OK | valid | 0.172 | llm_direct | OK | OK | 2151 | 2667 | 2 | 2 |
| 16 | Do not show casino ads in Russia... | edge | UNSURE | ambiguous | 0.130 | llm_with_check | OK | UNSURE | 3987 | 6552 | 3 | 5 |
| 17 | Premium users only | edge | UNSURE | ambiguous | 0.073 | llm_with_check | FAIL | FAIL | 822 | 703 | 1 | 3 |
| 18 | Allow everything except gambling | edge | UNSURE | ambiguous | 0.052 | llm_with_check | OK | OK | 1713 | 4033 | 2 | 4 |
| 19 | No more than twice a day and onl... | edge | UNSURE | ambiguous | 0.206 | llm_with_check | OK | OK | 3174 | 6041 | 3 | 5 |
| 20 | Show ads to iOS users in Germany | edge | UNSURE | valid | 0.019 | llm_with_check | OK | OK | 2968 | 7988 | 3 | 5 |
| 21 | Block competitors | edge | UNSURE | ambiguous | 0.002 | llm_with_check | OK | UNSURE | 2354 | 5741 | 2 | 4 |
| 22 | Only show once per session | edge | UNSURE | ambiguous | 0.116 | llm_with_check | OK | UNSURE | 2347 | 4711 | 2 | 4 |
| 23 | Block gambling in Russia and Ger... | edge | UNSURE | ambiguous | 0.151 | llm_with_check | OK | OK | 3993 | 8111 | 3 | 5 |
| 24 | Allow only mobile | edge | UNSURE | ambiguous | 0.016 | llm_with_check | OK | UNSURE | 2026 | 5447 | 2 | 4 |
| 25 | Limit frequency but only on week... | edge | UNSURE | ambiguous | 0.147 | llm_with_check | OK | UNSURE | 2027 | 4606 | 2 | 4 |
| 26 | kakashki | noisy | FAIL | gibberish | 0.332 | micro_early_fail | FAIL | FAIL | 709 | 319 | 1 | 0 |
| 27 | Block and allow gambling at the ... | noisy | FAIL | gibberish | 0.026 | llm_with_check | FAIL | UNSURE | 816 | 5120 | 1 | 5 |
| 28 | Show ads | noisy | FAIL | gibberish | 0.157 | micro_early_fail | FAIL | FAIL | 818 | 129 | 1 | 0 |
| 29 | xyzzy foo bar 123!!! | noisy | FAIL | gibberish | 0.296 | micro_early_fail | FAIL | FAIL | 791 | 126 | 1 | 0 |
| 30 | Do absolutely nothing with all a... | noisy | FAIL | gibberish | 0.180 | micro_early_fail | FAIL | FAIL | 893 | 120 | 1 | 0 |

## Summary

**LLM calls saved**: 58 → 88 (-51.7% reduction)

### Route Distribution

| Route | Count | % |
|-------|-------|---|
| micro_early_fail | 4 | 13% |
| llm_direct | 10 | 33% |
| llm_with_check | 16 | 53% |

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
| Two-Level (MicroFirst) | 24 | 30 | 80% |
