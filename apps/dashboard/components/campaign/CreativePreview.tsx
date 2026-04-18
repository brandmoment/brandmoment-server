"use client";

import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import type { Creative } from "@/types/campaign";

const PREVIEW_SIZES = [
  { label: "320×50 (Mobile Banner)", width: 320, height: 50 },
  { label: "300×250 (Medium Rectangle)", width: 300, height: 250 },
  { label: "728×90 (Leaderboard)", width: 728, height: 90 },
  { label: "160×600 (Wide Skyscraper)", width: 160, height: 600 },
] as const;

type PreviewSizeLabel = (typeof PREVIEW_SIZES)[number]["label"];

interface CreativePreviewProps {
  creative: Creative;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function CreativePreview({
  creative,
  open,
  onOpenChange,
}: CreativePreviewProps) {
  const [selectedSize, setSelectedSize] = useState<PreviewSizeLabel>(
    PREVIEW_SIZES[0].label
  );

  const size =
    PREVIEW_SIZES.find((s) => s.label === selectedSize) ?? PREVIEW_SIZES[0];

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>Preview: {creative.name}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div className="flex items-center gap-3">
            <Label className="shrink-0">Size</Label>
            <Select
              value={selectedSize}
              onValueChange={(v) => setSelectedSize(v as PreviewSizeLabel)}
            >
              <SelectTrigger className="w-64">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {PREVIEW_SIZES.map((s) => (
                  <SelectItem key={s.label} value={s.label}>
                    {s.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="rounded border bg-muted/30 p-4 overflow-auto">
            {creative.preview_url ? (
              <iframe
                src={creative.preview_url}
                sandbox="allow-scripts allow-same-origin"
                width={size.width}
                height={size.height}
                className="border-0 bg-white"
                title={`Preview of ${creative.name}`}
              />
            ) : (
              <div
                className="flex items-center justify-center bg-muted/50 rounded text-muted-foreground text-sm"
                style={{ width: size.width, height: Math.max(size.height, 80) }}
              >
                No preview available
              </div>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
