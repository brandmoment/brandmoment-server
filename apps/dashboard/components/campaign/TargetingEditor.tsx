"use client";

import { useState } from "react";
import { X } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import type { CampaignTargeting } from "@/types/campaign";

interface TargetingEditorProps {
  value: Partial<CampaignTargeting>;
  onChange: (targeting: Partial<CampaignTargeting>) => void;
  disabled?: boolean;
}

function TagInput({
  tags,
  onAdd,
  onRemove,
  placeholder,
  disabled,
}: {
  tags: string[];
  onAdd: (tag: string) => void;
  onRemove: (tag: string) => void;
  placeholder: string;
  disabled?: boolean;
}) {
  const [input, setInput] = useState("");

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if ((e.key === "Enter" || e.key === ",") && input.trim()) {
      e.preventDefault();
      const tag = input.trim().toUpperCase();
      if (!tags.includes(tag)) {
        onAdd(tag);
      }
      setInput("");
    }
  }

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap gap-1 min-h-8">
        {tags.map((tag) => (
          <Badge key={tag} variant="secondary" className="gap-1">
            {tag}
            {!disabled && (
              <button
                type="button"
                onClick={() => onRemove(tag)}
                className="hover:text-destructive"
              >
                <X className="h-3 w-3" />
              </button>
            )}
          </Badge>
        ))}
      </div>
      {!disabled && (
        <Input
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          className="h-8 text-sm"
        />
      )}
    </div>
  );
}

export function TargetingEditor({ value, onChange, disabled }: TargetingEditorProps) {
  const geo = value.geo ?? [];
  const platforms = value.platforms ?? [];
  const interests = value.interests ?? [];
  const ageRange = value.age_range;

  function updateGeo(geo: string[]) {
    onChange({ ...value, geo });
  }

  function updatePlatforms(platforms: string[]) {
    onChange({ ...value, platforms });
  }

  function updateInterests(interests: string[]) {
    onChange({ ...value, interests });
  }

  function updateAgeRange(field: "min" | "max", val: string) {
    const num = parseInt(val, 10);
    const current = ageRange ?? { min: 18, max: 65 };
    onChange({
      ...value,
      age_range: { ...current, [field]: isNaN(num) ? current[field] : num },
    });
  }

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label className="text-sm font-medium">Geo (country codes)</Label>
        <TagInput
          tags={geo}
          onAdd={(tag) => updateGeo([...geo, tag])}
          onRemove={(tag) => updateGeo(geo.filter((g) => g !== tag))}
          placeholder="Type US, CA... and press Enter"
          disabled={disabled}
        />
      </div>

      <div className="space-y-2">
        <Label className="text-sm font-medium">Platforms</Label>
        <TagInput
          tags={platforms}
          onAdd={(tag) => updatePlatforms([...platforms, tag.toLowerCase()])}
          onRemove={(tag) => updatePlatforms(platforms.filter((p) => p !== tag))}
          placeholder="Type ios, android, web... and press Enter"
          disabled={disabled}
        />
      </div>

      <div className="space-y-2">
        <Label className="text-sm font-medium">Age range (optional)</Label>
        <div className="flex items-center gap-2">
          <Input
            type="number"
            min={1}
            max={120}
            placeholder="Min"
            className="w-20 h-8 text-sm"
            value={ageRange?.min ?? ""}
            onChange={(e) => updateAgeRange("min", e.target.value)}
            disabled={disabled}
          />
          <span className="text-muted-foreground text-sm">to</span>
          <Input
            type="number"
            min={1}
            max={120}
            placeholder="Max"
            className="w-20 h-8 text-sm"
            value={ageRange?.max ?? ""}
            onChange={(e) => updateAgeRange("max", e.target.value)}
            disabled={disabled}
          />
          {ageRange && !disabled && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => onChange({ ...value, age_range: undefined })}
            >
              <X className="h-4 w-4" />
            </Button>
          )}
        </div>
      </div>

      <div className="space-y-2">
        <Label className="text-sm font-medium">Interests</Label>
        <TagInput
          tags={interests}
          onAdd={(tag) => updateInterests([...interests, tag.toLowerCase()])}
          onRemove={(tag) => updateInterests(interests.filter((i) => i !== tag))}
          placeholder="Type sports, music... and press Enter"
          disabled={disabled}
        />
      </div>
    </div>
  );
}
