"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
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
import { useCreatePublisherApp } from "@/hooks/useCreatePublisherApp";

const createAppSchema = z.object({
  name: z.string().min(1, "Name is required").max(100, "Name must be at most 100 characters"),
  platform: z.enum(["ios", "android", "web"], {
    required_error: "Platform is required",
  }),
  bundle_id: z.string().min(1, "Bundle ID is required"),
});

type CreateAppFormValues = z.infer<typeof createAppSchema>;

interface CreateAppDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function CreateAppDialog({ open, onOpenChange }: CreateAppDialogProps) {
  const router = useRouter();
  const { mutateAsync: createApp } = useCreatePublisherApp();
  const [selectedPlatform, setSelectedPlatform] = useState<"ios" | "android" | "web" | "">("");

  const {
    register,
    handleSubmit,
    setValue,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<CreateAppFormValues>({
    resolver: zodResolver(createAppSchema),
  });

  async function onSubmit(values: CreateAppFormValues) {
    try {
      const app = await createApp(values);
      toast.success(`App "${app.name}" created successfully`);
      reset();
      setSelectedPlatform("");
      onOpenChange(false);
      router.push(`/apps/${app.id}`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to create app");
    }
  }

  function handlePlatformChange(value: string) {
    const platform = value as "ios" | "android" | "web";
    setSelectedPlatform(platform);
    setValue("platform", platform, { shouldValidate: true });
  }

  function handleOpenChange(next: boolean) {
    if (!next) {
      reset();
      setSelectedPlatform("");
    }
    onOpenChange(next);
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Create New App</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="app-name">App name</Label>
            <Input
              id="app-name"
              placeholder="My Awesome App"
              disabled={isSubmitting}
              {...register("name")}
            />
            {errors.name && (
              <p className="text-xs text-destructive">{errors.name.message}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="app-platform">Platform</Label>
            <Select
              value={selectedPlatform}
              onValueChange={handlePlatformChange}
              disabled={isSubmitting}
            >
              <SelectTrigger id="app-platform">
                <SelectValue placeholder="Select platform" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="ios">iOS</SelectItem>
                <SelectItem value="android">Android</SelectItem>
                <SelectItem value="web">Web</SelectItem>
              </SelectContent>
            </Select>
            {errors.platform && (
              <p className="text-xs text-destructive">{errors.platform.message}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="app-bundle-id">Bundle ID</Label>
            <Input
              id="app-bundle-id"
              placeholder="com.example.myapp"
              disabled={isSubmitting}
              {...register("bundle_id")}
            />
            {errors.bundle_id && (
              <p className="text-xs text-destructive">{errors.bundle_id.message}</p>
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
              {isSubmitting ? "Creating..." : "Create app"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
