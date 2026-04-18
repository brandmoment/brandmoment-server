"use client";

import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { Info } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useCreateCreative } from "@/hooks/useCreateCreative";
import type { CreativeType } from "@/types/campaign";

const createCreativeSchema = z.object({
  name: z.string().min(1, "Name is required").max(200),
  type: z.enum(["html5", "image", "video"], { required_error: "Type is required" }),
  file_url: z.string().min(1, "File URL is required"),
  file_size_bytes: z.string().optional(),
});

type CreateCreativeFormValues = z.infer<typeof createCreativeSchema>;

interface CreativeUploadDialogProps {
  campaignId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function CreativeUploadDialog({
  campaignId,
  open,
  onOpenChange,
}: CreativeUploadDialogProps) {
  const { mutateAsync: createCreative } = useCreateCreative();
  const [selectedType, setSelectedType] = useState<CreativeType | "">("");

  const {
    register,
    handleSubmit,
    setValue,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<CreateCreativeFormValues>({
    resolver: zodResolver(createCreativeSchema),
  });

  async function onSubmit(values: CreateCreativeFormValues) {
    const fileSizeBytes =
      values.file_size_bytes ? parseInt(values.file_size_bytes, 10) : null;
    try {
      await createCreative({
        campaignId,
        body: {
          name: values.name,
          type: values.type,
          file_url: values.file_url,
          file_size_bytes: fileSizeBytes,
          preview_url: values.file_url,
        },
      });
      toast.success("Creative uploaded successfully");
      reset();
      setSelectedType("");
      onOpenChange(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to upload creative");
    }
  }

  function handleTypeChange(value: string) {
    const type = value as CreativeType;
    setSelectedType(type);
    setValue("type", type, { shouldValidate: true });
  }

  function handleOpenChange(next: boolean) {
    if (!next) {
      reset();
      setSelectedType("");
    }
    onOpenChange(next);
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Upload Creative</DialogTitle>
        </DialogHeader>

        <div className="flex items-start gap-2 rounded-md border border-blue-200 bg-blue-50 p-3 text-sm text-blue-800">
          <Info className="h-4 w-4 mt-0.5 shrink-0" />
          <p>
            Actual file upload is coming soon (Phase 4). Enter the file path or URL
            manually for now.
          </p>
        </div>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="creative-name">Creative name</Label>
            <Input
              id="creative-name"
              placeholder="Banner 320x50"
              disabled={isSubmitting}
              {...register("name")}
            />
            {errors.name && (
              <p className="text-xs text-destructive">{errors.name.message}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="creative-type">Type</Label>
            <Select
              value={selectedType}
              onValueChange={handleTypeChange}
              disabled={isSubmitting}
            >
              <SelectTrigger id="creative-type">
                <SelectValue placeholder="Select type" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="html5">HTML5</SelectItem>
                <SelectItem value="image">Image</SelectItem>
                <SelectItem value="video">Video</SelectItem>
              </SelectContent>
            </Select>
            {errors.type && (
              <p className="text-xs text-destructive">{errors.type.message}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="creative-url">File URL / path</Label>
            <Input
              id="creative-url"
              placeholder="s3://brandmoment-creatives/..."
              disabled={isSubmitting}
              {...register("file_url")}
            />
            {errors.file_url && (
              <p className="text-xs text-destructive">{errors.file_url.message}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="creative-size">File size (bytes, optional)</Label>
            <Input
              id="creative-size"
              type="number"
              min="1"
              placeholder="204800"
              disabled={isSubmitting}
              {...register("file_size_bytes")}
            />
            {errors.file_size_bytes && (
              <p className="text-xs text-destructive">
                {errors.file_size_bytes.message}
              </p>
            )}
          </div>

          <DialogFooter className="gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => handleOpenChange(false)}
              disabled={isSubmitting}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? "Uploading..." : "Upload creative"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
