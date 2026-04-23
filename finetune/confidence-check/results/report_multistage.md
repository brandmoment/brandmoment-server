# Multi-Stage vs Monolithic Benchmark Report

**Model**: gpt-4o-mini  
**Date**: 2026-04-23  
**Corpus**: 30 phrases

## Per-Phrase Results

| # | Phrase | Group | Mono Conf | Multi Conf | Mono ms | Multi ms | Mono Tok | Multi Tok |
|---|--------|-------|-----------|-----------|---------|---------|---------|----------|
| 1 | Block gambling category | correct | OK | OK | 2087 | 1968 | 412 | 400 |
| 2 | Allow only iOS devices | correct | OK | OK | 1893 | 2008 | 418 | 408 |
| 3 | Show maximum 5 times per hour | correct | OK | OK | 1626 | 2148 | 422 | 424 |
| 4 | Exclude Russia and Kazakhstan | correct | OK | OK | 1640 | 1839 | 420 | 409 |
| 5 | Only show ads in the United States an... | correct | OK | OK | 818 | 2868 | 424 | 427 |
| 6 | Block domain evil.com | correct | OK | OK | 920 | 2149 | 413 | 403 |
| 7 | Allow only the bundle ID com.goodapp.ios | correct | OK | OK | 739 | 2026 | 423 | 428 |
| 8 | Show no more than 10 impressions per day | correct | OK | OK | 844 | 1816 | 424 | 423 |
| 9 | Allow Android and iOS only | correct | OK | OK | 920 | 2152 | 421 | 423 |
| 10 | Block the bundle com.casino.app | correct | OK | OK | 919 | 2151 | 419 | 411 |
| 11 | Exclude Germany from ad delivery | correct | OK | OK | 1126 | 2149 | 418 | 402 |
| 12 | Cap impressions at 3 per week | correct | OK | OK | 1023 | 1741 | 422 | 416 |
| 13 | Only web and CTV platforms | correct | OK | OK | 868 | 1807 | 422 | 419 |
| 14 | Block adult content domains | correct | OK | OK | 825 | 2026 | 412 | 406 |
| 15 | Allow domains news.com and media.org ... | correct | OK | OK | 1270 | 1801 | 420 | 414 |
| 16 | Do not show casino ads in Russia more... | edge | OK | OK | 1328 | 2864 | 461 | 680 |
| 17 | Premium users only | edge | FAIL | FAIL | 488 | 843 | 397 | 165 |
| 18 | Allow everything except gambling | edge | OK | OK | 1228 | 1739 | 413 | 396 |
| 19 | No more than twice a day and only in ... | edge | OK | OK | 1742 | 2762 | 443 | 672 |
| 20 | Show ads to iOS users in Germany | edge | OK | OK | 1228 | 3787 | 439 | 651 |
| 21 | Block competitors | edge | FAIL | OK | 1227 | 1741 | 396 | 394 |
| 22 | Only show once per session | edge | OK | OK | 720 | 2430 | 420 | 414 |
| 23 | Block gambling in Russia and Germany,... | edge | OK | OK | 1046 | 2866 | 450 | 667 |
| 24 | Allow only mobile | edge | OK | OK | 895 | 3097 | 418 | 411 |
| 25 | Limit frequency but only on weekends | edge | FAIL | OK | 612 | 1843 | 400 | 408 |
| 26 | kakashki | noisy | FAIL | FAIL | 761 | 775 | 397 | 165 |
| 27 | Block and allow gambling at the same ... | noisy | FAIL | OK | 1227 | 3276 | 402 | 635 |
| 28 | Show ads | noisy | FAIL | FAIL | 716 | 818 | 396 | 164 |
| 29 | xyzzy foo bar 123!!! | noisy | FAIL | FAIL | 819 | 819 | 401 | 169 |
| 30 | Do absolutely nothing with all ads ev... | noisy | FAIL | FAIL | 716 | 757 | 402 | 170 |

## Summary

| Metric | Monolithic | Multi-Stage |
|--------|-----------|-------------|
| OK | 22 | 25 |
| UNSURE | 0 | 0 |
| FAIL | 8 | 5 |
| Avg latency | 1.08s | 2.04s |
| Total tokens | 12525 | 12374 |

**Multi-stage wins on confidence**: 3 phrases  
**Monolithic wins on confidence**: 0 phrases
