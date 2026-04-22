# Routing Benchmark Report

**Primary model**: gpt-4o-mini  
**Fallback model**: gpt-4o  
**Date**: 2026-04-22  
**Corpus**: 30 phrases

## Results

| # | Phrase | Group | Primary Conf | Routed To | Final Conf | Latency |
|---|--------|-------|-------------|-----------|-----------|--------|
| 1 | Block gambling category | correct | OK | primary | OK | 4.0s |
| 2 | Allow only iOS devices | correct | OK | primary | OK | 2.5s |
| 3 | Show maximum 5 times per hour | correct | OK | primary | OK | 2.4s |
| 4 | Exclude Russia and Kazakhstan | correct | OK | primary | OK | 2.4s |
| 5 | Only show ads in the United States an... | correct | OK | primary | OK | 2.5s |
| 6 | Block domain evil.com | correct | OK | primary | OK | 2.4s |
| 7 | Allow only the bundle ID com.goodapp.ios | correct | OK | primary | OK | 2.7s |
| 8 | Show no more than 10 impressions per day | correct | OK | primary | OK | 2.3s |
| 9 | Allow Android and iOS only | correct | OK | primary | OK | 2.1s |
| 10 | Block the bundle com.casino.app | correct | OK | primary | OK | 6.2s |
| 11 | Exclude Germany from ad delivery | correct | OK | primary | OK | 2.1s |
| 12 | Cap impressions at 3 per week | correct | OK | primary | OK | 2.7s |
| 13 | Only web and CTV platforms | correct | OK | primary | OK | 2.5s |
| 14 | Block adult content domains | correct | OK | primary | OK | 2.7s |
| 15 | Allow domains news.com and media.org ... | correct | OK | primary | OK | 2.0s |
| 16 | Do not show casino ads in Russia more... | edge | UNSURE | fallback | UNSURE | 6.5s |
| 17 | Premium users only | edge | FAIL | fallback | FAIL | 4.2s |
| 18 | Allow everything except gambling | edge | OK | primary | OK | 2.2s |
| 19 | No more than twice a day and only in ... | edge | OK | primary | OK | 2.5s |
| 20 | Show ads to iOS users in Germany | edge | OK | primary | OK | 2.4s |
| 21 | Block competitors | edge | FAIL | fallback | FAIL | 3.8s |
| 22 | Only show once per session | edge | UNSURE | fallback | UNSURE | 5.2s |
| 23 | Block gambling in Russia and Germany,... | edge | OK | primary | OK | 3.0s |
| 24 | Allow only mobile | edge | UNSURE | fallback | OK | 5.2s |
| 25 | Limit frequency but only on weekends | edge | FAIL | fallback | FAIL | 3.9s |
| 26 | kakashki | noisy | FAIL | fallback | FAIL | 3.3s |
| 27 | Block and allow gambling at the same ... | noisy | FAIL | fallback | FAIL | 3.5s |
| 28 | Show ads | noisy | FAIL | fallback | FAIL | 3.8s |
| 29 | xyzzy foo bar 123!!! | noisy | FAIL | fallback | FAIL | 3.4s |
| 30 | Do absolutely nothing with all ads ev... | noisy | FAIL | fallback | FAIL | 3.4s |

## Summary

| Metric | Value |
|--------|-------|
| Stayed on primary (gpt-4o-mini) | 19/30 (63%) |
| Escalated to fallback (gpt-4o) | 11/30 (37%) |
| Total input tokens | 20587 |
| Total output tokens | 2062 |
