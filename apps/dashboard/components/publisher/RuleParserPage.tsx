"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Label } from "@/components/ui/label";
import { useParseRule } from "@/hooks/useParseRule";
import type {
  LLMProvider,
  ApproachName,
  ConfidenceStatus,
  ParseRuleResponse,
  ApproachResults,
} from "@/types/rule-parser";

const ALL_APPROACHES: { value: ApproachName; label: string }[] = [
  { value: "constraint", label: "Constraint" },
  { value: "self_check", label: "Self-Check" },
];

const PROVIDERS: { value: LLMProvider; label: string }[] = [
  { value: "openai", label: "OpenAI (gpt-4o-mini)" },
  { value: "gemini", label: "Gemini (gemini-2.0-flash)" },
];

function confidenceBadgeVariant(
  status: ConfidenceStatus
): "default" | "secondary" | "destructive" | "outline" {
  if (status === "OK") return "default";
  if (status === "UNSURE") return "secondary";
  return "destructive";
}

function confidenceStatusLabel(status: ConfidenceStatus): string {
  return status;
}

function formatMs(ms: number): string {
  if (ms >= 1000) return `${(ms / 1000).toFixed(2)}s`;
  return `${ms}ms`;
}

interface ApproachRowProps {
  name: ApproachName;
  approaches: ApproachResults;
}

function ApproachRow({ name, approaches }: ApproachRowProps) {
  const result = approaches[name];
  if (!result) return null;

  let details = "—";
  if (name === "self_check" && approaches.self_check) {
    details = approaches.self_check.explanation;
  }

  const labelMap: Record<ApproachName, string> = {
    constraint: "Constraint",
    self_check: "Self-Check",
  };

  return (
    <tr className="border-b last:border-0">
      <td className="px-4 py-3 text-sm font-medium">{labelMap[name]}</td>
      <td className="px-4 py-3">
        <Badge variant={confidenceBadgeVariant(result.status)}>
          {confidenceStatusLabel(result.status)}
        </Badge>
      </td>
      <td className="px-4 py-3 text-sm text-muted-foreground">
        {formatMs(result.latency_ms)}
      </td>
      <td className="px-4 py-3 text-sm text-muted-foreground max-w-xs truncate">
        {details}
      </td>
    </tr>
  );
}

interface ResultsPanelProps {
  result: ParseRuleResponse;
}

function ResultsPanel({ result }: ResultsPanelProps) {
  const { rules, confidence } = result;
  const approachNames = Object.keys(confidence.approaches) as ApproachName[];

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="text-base">Parse Result</CardTitle>
            <div className="flex items-center gap-2">
              <span className="text-sm text-muted-foreground">
                Overall confidence:
              </span>
              <Badge
                variant={confidenceBadgeVariant(confidence.overall)}
                className="text-sm px-3 py-1"
              >
                {confidenceStatusLabel(confidence.overall)}
              </Badge>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            <p className="text-sm font-medium text-muted-foreground">
              {rules.length === 0
                ? "No rules extracted"
                : `${rules.length} rule${rules.length === 1 ? "" : "s"} extracted`}
            </p>
            {rules.map((rule, idx) => (
              <div
                key={idx}
                className="rounded-md border bg-muted/40 p-4 space-y-2"
              >
                <div className="flex items-center gap-2">
                  <Badge variant="outline">{rule.type}</Badge>
                  <span className="text-xs text-muted-foreground">
                    Rule {idx + 1}
                  </span>
                </div>
                <pre className="text-xs font-mono overflow-x-auto whitespace-pre-wrap break-all">
                  {JSON.stringify(rule.config, null, 2)}
                </pre>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Confidence Breakdown</CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          <table className="w-full">
            <thead>
              <tr className="border-b bg-muted/30">
                <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">
                  Approach
                </th>
                <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">
                  Status
                </th>
                <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">
                  Latency
                </th>
                <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">
                  Details
                </th>
              </tr>
            </thead>
            <tbody>
              {approachNames.map((name) => (
                <ApproachRow
                  key={name}
                  name={name}
                  approaches={confidence.approaches}
                />
              ))}
            </tbody>
          </table>
        </CardContent>
      </Card>

      <div className="flex items-center gap-6 text-sm text-muted-foreground">
        <span>
          Total latency:{" "}
          <span className="font-medium text-foreground">
            {formatMs(confidence.total_latency_ms)}
          </span>
        </span>
        <span>
          Total tokens:{" "}
          <span className="font-medium text-foreground">
            {confidence.total_tokens.toLocaleString()}
          </span>
        </span>
      </div>
    </div>
  );
}

function LoadingSkeleton() {
  return (
    <div className="space-y-6">
      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <Skeleton className="h-5 w-28" />
            <Skeleton className="h-7 w-36" />
          </div>
        </CardHeader>
        <CardContent className="space-y-3">
          <Skeleton className="h-4 w-40" />
          <Skeleton className="h-24 w-full" />
          <Skeleton className="h-24 w-full" />
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="pb-3">
          <Skeleton className="h-5 w-48" />
        </CardHeader>
        <CardContent className="space-y-2">
          <Skeleton className="h-9 w-full" />
          <Skeleton className="h-9 w-full" />
          <Skeleton className="h-9 w-full" />
        </CardContent>
      </Card>
    </div>
  );
}

export function RuleParserPage() {
  const [phrase, setPhrase] = useState("");
  const [provider, setProvider] = useState<LLMProvider>("openai");
  const [selectedApproaches, setSelectedApproaches] = useState<
    Set<ApproachName>
  >(new Set(["constraint", "self_check"]));

  const { mutate, data, error, isPending, isSuccess } = useParseRule();

  function toggleApproach(name: ApproachName) {
    setSelectedApproaches((prev) => {
      const next = new Set(prev);
      if (next.has(name)) {
        if (next.size === 1) return prev; // keep at least one selected
        next.delete(name);
      } else {
        next.add(name);
      }
      return next;
    });
  }

  function handleSubmit() {
    const trimmed = phrase.trim();
    if (!trimmed) return;
    mutate({
      phrase: trimmed,
      provider,
      approaches: Array.from(selectedApproaches),
    });
  }

  return (
    <div className="mx-auto max-w-3xl space-y-8">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Rule Parser</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Describe a publisher rule in natural language and let the LLM parse it
          into structured config.
        </p>
      </div>

      <Card>
        <CardHeader className="pb-4">
          <CardTitle className="text-base">Input</CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="space-y-2">
            <Label htmlFor="phrase">Phrase</Label>
            <Textarea
              id="phrase"
              placeholder='e.g. "Block gambling in Russia no more than 3 times per day"'
              value={phrase}
              onChange={(e) => setPhrase(e.target.value)}
              rows={4}
              className="resize-none"
            />
          </div>

          <div className="space-y-2">
            <Label>Provider</Label>
            <div className="flex gap-6">
              {PROVIDERS.map(({ value, label }) => (
                <label
                  key={value}
                  className="flex cursor-pointer items-center gap-2 text-sm"
                >
                  <input
                    type="radio"
                    name="provider"
                    value={value}
                    checked={provider === value}
                    onChange={() => setProvider(value)}
                    className="h-4 w-4 accent-primary"
                  />
                  {label}
                </label>
              ))}
            </div>
          </div>

          <div className="space-y-2">
            <Label>Confidence approaches</Label>
            <div className="flex flex-wrap gap-4">
              {ALL_APPROACHES.map(({ value, label }) => (
                <label
                  key={value}
                  className="flex cursor-pointer items-center gap-2 text-sm"
                >
                  <input
                    type="checkbox"
                    checked={selectedApproaches.has(value)}
                    onChange={() => toggleApproach(value)}
                    className="h-4 w-4 accent-primary"
                  />
                  {label}
                </label>
              ))}
            </div>
          </div>

          <Button
            onClick={handleSubmit}
            disabled={isPending || phrase.trim().length === 0}
            className="w-full sm:w-auto"
          >
            {isPending ? "Parsing..." : "Parse"}
          </Button>
        </CardContent>
      </Card>

      {isPending && <LoadingSkeleton />}

      {!isPending && error && (
        <div className="rounded-md border border-destructive/50 bg-destructive/10 p-4">
          <p className="text-sm font-medium text-destructive">Parse failed</p>
          <p className="mt-1 text-sm text-destructive/80">{error.message}</p>
        </div>
      )}

      {!isPending && isSuccess && data && <ResultsPanel result={data} />}
    </div>
  );
}
