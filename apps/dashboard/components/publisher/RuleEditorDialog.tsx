"use client";

import { useState, useEffect } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useCreateRule } from "@/hooks/useCreateRule";
import { useUpdateRule } from "@/hooks/useUpdateRule";
import type {
  RuleType,
  PublisherRule,
  BlocklistAllowlistConfig,
  FrequencyCapConfig,
  GeoFilterConfig,
  PlatformFilterConfig,
} from "@/types/publisher-rule";

const RULE_TYPE_LABELS: Record<RuleType, string> = {
  blocklist: "Blocklist",
  allowlist: "Allowlist",
  frequency_cap: "Frequency Cap",
  geo_filter: "Geo Filter",
  platform_filter: "Platform Filter",
};

const RULE_TYPES: RuleType[] = [
  "blocklist",
  "allowlist",
  "frequency_cap",
  "geo_filter",
  "platform_filter",
];

const COMMON_COUNTRY_CODES = [
  "US", "GB", "DE", "FR", "JP", "CN", "RU", "BR", "CA", "AU",
  "IN", "KR", "MX", "ES", "IT", "NL", "PL", "SE", "NO", "CH",
];

const PLATFORMS: Array<"ios" | "android" | "web"> = ["ios", "android", "web"];

interface RuleEditorDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  appId: string;
  editRule?: PublisherRule | null;
}

export function RuleEditorDialog({
  open,
  onOpenChange,
  appId,
  editRule,
}: RuleEditorDialogProps) {
  const isEditing = Boolean(editRule);

  const [ruleType, setRuleType] = useState<RuleType | "">("");
  const [domains, setDomains] = useState("");
  const [bundleIds, setBundleIds] = useState("");
  const [maxImpressions, setMaxImpressions] = useState("");
  const [windowSeconds, setWindowSeconds] = useState("");
  const [geoMode, setGeoMode] = useState<"include" | "exclude">("exclude");
  const [selectedCountries, setSelectedCountries] = useState<string[]>([]);
  const [platformMode, setPlatformMode] = useState<"include" | "exclude">("include");
  const [selectedPlatforms, setSelectedPlatforms] = useState<string[]>([]);

  const { mutateAsync: createRule, isPending: isCreating } = useCreateRule();
  const { mutateAsync: updateRule, isPending: isUpdating } = useUpdateRule();
  const isPending = isCreating || isUpdating;

  useEffect(() => {
    if (editRule) {
      setRuleType(editRule.type);
      const cfg = editRule.config;
      if (editRule.type === "blocklist" || editRule.type === "allowlist") {
        const c = cfg as unknown as BlocklistAllowlistConfig;
        setDomains((c.domains ?? []).join("\n"));
        setBundleIds((c.bundle_ids ?? []).join("\n"));
      } else if (editRule.type === "frequency_cap") {
        const c = cfg as unknown as FrequencyCapConfig;
        setMaxImpressions(String(c.max_impressions ?? ""));
        setWindowSeconds(String(c.window_seconds ?? ""));
      } else if (editRule.type === "geo_filter") {
        const c = cfg as unknown as GeoFilterConfig;
        setGeoMode(c.mode ?? "exclude");
        setSelectedCountries(c.country_codes ?? []);
      } else if (editRule.type === "platform_filter") {
        const c = cfg as unknown as PlatformFilterConfig;
        setPlatformMode(c.mode ?? "include");
        setSelectedPlatforms(c.platforms ?? []);
      }
    }
  }, [editRule]);

  function resetForm() {
    setRuleType("");
    setDomains("");
    setBundleIds("");
    setMaxImpressions("");
    setWindowSeconds("");
    setGeoMode("exclude");
    setSelectedCountries([]);
    setPlatformMode("include");
    setSelectedPlatforms([]);
  }

  function handleOpenChange(next: boolean) {
    if (!next) resetForm();
    onOpenChange(next);
  }

  function buildConfig(): Record<string, unknown> {
    switch (ruleType) {
      case "blocklist":
      case "allowlist":
        return {
          domains: domains.split("\n").map((d) => d.trim()).filter(Boolean),
          bundle_ids: bundleIds.split("\n").map((b) => b.trim()).filter(Boolean),
        } satisfies BlocklistAllowlistConfig;
      case "frequency_cap":
        return {
          max_impressions: parseInt(maxImpressions, 10),
          window_seconds: parseInt(windowSeconds, 10),
        } satisfies FrequencyCapConfig;
      case "geo_filter":
        return {
          mode: geoMode,
          country_codes: selectedCountries,
        } satisfies GeoFilterConfig;
      case "platform_filter":
        return {
          mode: platformMode,
          platforms: selectedPlatforms as ("ios" | "android" | "web")[],
        } satisfies PlatformFilterConfig;
      default:
        return {};
    }
  }

  function toggleCountry(code: string) {
    setSelectedCountries((prev) =>
      prev.includes(code) ? prev.filter((c) => c !== code) : [...prev, code]
    );
  }

  function togglePlatform(platform: string) {
    setSelectedPlatforms((prev) =>
      prev.includes(platform)
        ? prev.filter((p) => p !== platform)
        : [...prev, platform]
    );
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!ruleType) return;

    const config = buildConfig();

    try {
      if (isEditing && editRule) {
        await updateRule({ appId, ruleId: editRule.id, body: { config } });
        toast.success("Rule updated");
      } else {
        await createRule({
          appId,
          body: { type: ruleType as RuleType, config },
        });
        toast.success("Rule created");
      }
      handleOpenChange(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to save rule");
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-lg max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEditing ? "Edit Rule" : "Add Rule"}</DialogTitle>
          <DialogDescription>
            Configure how ads are filtered for this app.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label>Rule type</Label>
            <Select
              value={ruleType}
              onValueChange={(v) => setRuleType(v as RuleType)}
              disabled={isEditing || isPending}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select rule type" />
              </SelectTrigger>
              <SelectContent>
                {RULE_TYPES.map((type) => (
                  <SelectItem key={type} value={type}>
                    {RULE_TYPE_LABELS[type]}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {(ruleType === "blocklist" || ruleType === "allowlist") && (
            <>
              <div className="space-y-2">
                <Label>Domains (one per line)</Label>
                <Textarea
                  placeholder="example.com&#10;ads.badsite.net"
                  value={domains}
                  onChange={(e) => setDomains(e.target.value)}
                  disabled={isPending}
                  rows={4}
                />
              </div>
              <div className="space-y-2">
                <Label>Bundle IDs (one per line)</Label>
                <Textarea
                  placeholder="com.example.app&#10;com.competitor.app"
                  value={bundleIds}
                  onChange={(e) => setBundleIds(e.target.value)}
                  disabled={isPending}
                  rows={4}
                />
              </div>
            </>
          )}

          {ruleType === "frequency_cap" && (
            <>
              <div className="space-y-2">
                <Label htmlFor="max-impressions">Max impressions</Label>
                <Input
                  id="max-impressions"
                  type="number"
                  min={1}
                  placeholder="10"
                  value={maxImpressions}
                  onChange={(e) => setMaxImpressions(e.target.value)}
                  disabled={isPending}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="window-seconds">Window (seconds)</Label>
                <Input
                  id="window-seconds"
                  type="number"
                  min={1}
                  placeholder="3600"
                  value={windowSeconds}
                  onChange={(e) => setWindowSeconds(e.target.value)}
                  disabled={isPending}
                  required
                />
                <p className="text-xs text-muted-foreground">
                  e.g., 3600 = 1 hour, 86400 = 1 day
                </p>
              </div>
            </>
          )}

          {ruleType === "geo_filter" && (
            <>
              <div className="space-y-2">
                <Label>Mode</Label>
                <div className="flex gap-4">
                  {(["include", "exclude"] as const).map((m) => (
                    <label key={m} className="flex items-center gap-2 cursor-pointer">
                      <input
                        type="radio"
                        name="geo-mode"
                        value={m}
                        checked={geoMode === m}
                        onChange={() => setGeoMode(m)}
                        disabled={isPending}
                      />
                      <span className="text-sm capitalize">{m}</span>
                    </label>
                  ))}
                </div>
              </div>
              <div className="space-y-2">
                <Label>Country codes</Label>
                <div className="grid grid-cols-5 gap-2">
                  {COMMON_COUNTRY_CODES.map((code) => (
                    <label
                      key={code}
                      className={`flex cursor-pointer items-center justify-center rounded border px-2 py-1 text-xs transition-colors ${
                        selectedCountries.includes(code)
                          ? "border-primary bg-primary/10 text-primary font-medium"
                          : "border-border hover:bg-muted/50"
                      }`}
                    >
                      <input
                        type="checkbox"
                        className="sr-only"
                        checked={selectedCountries.includes(code)}
                        onChange={() => toggleCountry(code)}
                        disabled={isPending}
                      />
                      {code}
                    </label>
                  ))}
                </div>
                {selectedCountries.length > 0 && (
                  <p className="text-xs text-muted-foreground">
                    Selected: {selectedCountries.join(", ")}
                  </p>
                )}
              </div>
            </>
          )}

          {ruleType === "platform_filter" && (
            <>
              <div className="space-y-2">
                <Label>Mode</Label>
                <div className="flex gap-4">
                  {(["include", "exclude"] as const).map((m) => (
                    <label key={m} className="flex items-center gap-2 cursor-pointer">
                      <input
                        type="radio"
                        name="platform-mode"
                        value={m}
                        checked={platformMode === m}
                        onChange={() => setPlatformMode(m)}
                        disabled={isPending}
                      />
                      <span className="text-sm capitalize">{m}</span>
                    </label>
                  ))}
                </div>
              </div>
              <div className="space-y-2">
                <Label>Platforms</Label>
                <div className="flex gap-3">
                  {PLATFORMS.map((platform) => (
                    <label
                      key={platform}
                      className={`flex cursor-pointer items-center gap-2 rounded border px-3 py-2 text-sm transition-colors ${
                        selectedPlatforms.includes(platform)
                          ? "border-primary bg-primary/10 text-primary font-medium"
                          : "border-border hover:bg-muted/50"
                      }`}
                    >
                      <input
                        type="checkbox"
                        className="sr-only"
                        checked={selectedPlatforms.includes(platform)}
                        onChange={() => togglePlatform(platform)}
                        disabled={isPending}
                      />
                      <span className="capitalize">{platform}</span>
                    </label>
                  ))}
                </div>
              </div>
            </>
          )}

          <DialogFooter className="gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => handleOpenChange(false)}
              disabled={isPending}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isPending || !ruleType}>
              {isPending
                ? isEditing
                  ? "Saving..."
                  : "Creating..."
                : isEditing
                  ? "Save changes"
                  : "Add rule"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
