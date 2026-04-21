import type { RuleType } from "@/types/publisher-rule";

export type LLMProvider = "openai" | "gemini";

export type ApproachName = "constraint" | "self_check";

export type ConfidenceStatus = "OK" | "UNSURE" | "FAIL";

export interface ConstraintApproachResult {
  status: ConfidenceStatus;
  latency_ms: number;
}

export interface SelfCheckApproachResult {
  status: ConfidenceStatus;
  verified: boolean;
  explanation: string;
  latency_ms: number;
}

export type ApproachResult = ConstraintApproachResult | SelfCheckApproachResult;

export interface ApproachResults {
  constraint?: ConstraintApproachResult;
  self_check?: SelfCheckApproachResult;
}

export interface ConfidenceReport {
  overall: ConfidenceStatus;
  approaches: ApproachResults;
  total_latency_ms: number;
  total_tokens: number;
}

export interface ParsedRule {
  type: RuleType;
  config: Record<string, unknown>;
}

export interface ParseRuleRequest {
  phrase: string;
  provider: LLMProvider;
  approaches: ApproachName[];
}

export interface ParseRuleResponse {
  rules: ParsedRule[];
  confidence: ConfidenceReport;
}
