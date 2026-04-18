"use client";

import { useState } from "react";
import { Eye, Upload } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useCreatives } from "@/hooks/useCreatives";
import { CreativePreview } from "@/components/campaign/CreativePreview";
import { CreativeUploadDialog } from "@/components/campaign/CreativeUploadDialog";
import type { Creative, CreativeType } from "@/types/campaign";

const TYPE_LABELS: Record<CreativeType, string> = {
  html5: "HTML5",
  image: "Image",
  video: "Video",
};

function formatFileSize(bytes: number | null) {
  if (bytes === null) return "—";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

interface CreativesListProps {
  campaignId: string;
}

export function CreativesList({ campaignId }: CreativesListProps) {
  const { data, isLoading, isError } = useCreatives(campaignId);
  const [previewCreative, setPreviewCreative] = useState<Creative | null>(null);
  const [uploadOpen, setUploadOpen] = useState(false);

  return (
    <Card>
      <CardHeader className="px-6 py-4">
        <div className="flex items-center justify-between">
          <CardTitle className="text-base">
            {data
              ? `${data.total} creative${data.total !== 1 ? "s" : ""}`
              : "Creatives"}
          </CardTitle>
          <Button size="sm" onClick={() => setUploadOpen(true)}>
            <Upload className="mr-2 h-4 w-4" />
            Upload Creative
          </Button>
        </div>
      </CardHeader>
      <CardContent className="p-0">
        {isLoading ? (
          <div className="px-6 pb-6 space-y-2">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-10 w-full" />
            ))}
          </div>
        ) : isError ? (
          <div className="px-6 pb-6 text-sm text-destructive">
            Failed to load creatives. Please refresh.
          </div>
        ) : !data || data.items.length === 0 ? (
          <div className="px-6 pb-10 pt-6 text-center">
            <p className="text-muted-foreground text-sm">No creatives yet.</p>
            <Button
              variant="outline"
              className="mt-4"
              onClick={() => setUploadOpen(true)}
            >
              <Upload className="mr-2 h-4 w-4" />
              Upload your first creative
            </Button>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/40">
                  <th className="px-6 py-3 text-left font-medium text-muted-foreground">
                    Name
                  </th>
                  <th className="px-6 py-3 text-left font-medium text-muted-foreground">
                    Type
                  </th>
                  <th className="px-6 py-3 text-left font-medium text-muted-foreground">
                    File Size
                  </th>
                  <th className="px-6 py-3 text-left font-medium text-muted-foreground">
                    Active
                  </th>
                  <th className="px-6 py-3 text-left font-medium text-muted-foreground">
                    Created
                  </th>
                  <th className="px-6 py-3 text-left font-medium text-muted-foreground">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {data.items.map((creative) => (
                  <tr key={creative.id} className="hover:bg-muted/20 transition-colors">
                    <td className="px-6 py-3 font-medium">{creative.name}</td>
                    <td className="px-6 py-3">
                      <Badge variant="outline">{TYPE_LABELS[creative.type]}</Badge>
                    </td>
                    <td className="px-6 py-3 text-muted-foreground">
                      {formatFileSize(creative.file_size_bytes)}
                    </td>
                    <td className="px-6 py-3">
                      <Badge variant={creative.is_active ? "success" : "outline"}>
                        {creative.is_active ? "Active" : "Inactive"}
                      </Badge>
                    </td>
                    <td className="px-6 py-3 text-muted-foreground">
                      {formatDate(creative.created_at)}
                    </td>
                    <td className="px-6 py-3">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setPreviewCreative(creative)}
                      >
                        <Eye className="h-4 w-4 mr-1" />
                        Preview
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </CardContent>

      {previewCreative && (
        <CreativePreview
          creative={previewCreative}
          open={Boolean(previewCreative)}
          onOpenChange={(open) => {
            if (!open) setPreviewCreative(null);
          }}
        />
      )}

      <CreativeUploadDialog
        campaignId={campaignId}
        open={uploadOpen}
        onOpenChange={setUploadOpen}
      />
    </Card>
  );
}
