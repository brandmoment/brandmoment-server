"use client";

import { useState } from "react";
import { toast } from "sonner";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useUpdatePublisherApp } from "@/hooks/useUpdatePublisherApp";
import type { PublisherApp } from "@/types/publisher-app";

function formatDate(iso: string) {
  return new Date(iso).toLocaleString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

const PLATFORM_LABELS: Record<PublisherApp["platform"], string> = {
  ios: "iOS",
  android: "Android",
  web: "Web",
};

const editSchema = z.object({
  name: z.string().min(1, "Name is required").max(100, "Max 100 characters"),
});

type EditFormValues = z.infer<typeof editSchema>;

interface AppOverviewTabProps {
  app: PublisherApp;
}

export function AppOverviewTab({ app }: AppOverviewTabProps) {
  const [editing, setEditing] = useState(false);
  const { mutateAsync: updateApp } = useUpdatePublisherApp();

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<EditFormValues>({
    resolver: zodResolver(editSchema),
    defaultValues: { name: app.name },
  });

  async function onSubmit(values: EditFormValues) {
    try {
      await updateApp({ id: app.id, body: { name: values.name } });
      toast.success("App updated");
      setEditing(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to update app");
    }
  }

  async function handleToggleActive() {
    try {
      await updateApp({ id: app.id, body: { is_active: !app.is_active } });
      toast.success(app.is_active ? "App deactivated" : "App activated");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to update app");
    }
  }

  function handleCancelEdit() {
    reset({ name: app.name });
    setEditing(false);
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-base">App Details</CardTitle>
          {!editing && (
            <Button variant="outline" size="sm" onClick={() => setEditing(true)}>
              Edit
            </Button>
          )}
        </CardHeader>
        <CardContent className="space-y-4">
          {editing ? (
            <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="edit-name">App name</Label>
                <Input
                  id="edit-name"
                  disabled={isSubmitting}
                  {...register("name")}
                />
                {errors.name && (
                  <p className="text-xs text-destructive">{errors.name.message}</p>
                )}
              </div>
              <div className="flex gap-2">
                <Button type="button" variant="outline" onClick={handleCancelEdit} disabled={isSubmitting}>
                  Cancel
                </Button>
                <Button type="submit" disabled={isSubmitting}>
                  {isSubmitting ? "Saving..." : "Save changes"}
                </Button>
              </div>
            </form>
          ) : (
            <dl className="space-y-3">
              <div className="flex items-center justify-between">
                <dt className="text-sm text-muted-foreground">Name</dt>
                <dd className="text-sm font-medium">{app.name}</dd>
              </div>
              <div className="flex items-center justify-between">
                <dt className="text-sm text-muted-foreground">Platform</dt>
                <dd>
                  <Badge variant="outline">{PLATFORM_LABELS[app.platform]}</Badge>
                </dd>
              </div>
              <div className="flex items-center justify-between">
                <dt className="text-sm text-muted-foreground">Bundle ID</dt>
                <dd className="font-mono text-xs text-muted-foreground">{app.bundle_id}</dd>
              </div>
              <div className="flex items-center justify-between">
                <dt className="text-sm text-muted-foreground">Status</dt>
                <dd className="flex items-center gap-2">
                  <Badge variant={app.is_active ? "success" : "outline"}>
                    {app.is_active ? "Active" : "Inactive"}
                  </Badge>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleToggleActive}
                    className="text-xs h-auto p-1"
                  >
                    {app.is_active ? "Deactivate" : "Activate"}
                  </Button>
                </dd>
              </div>
              <div className="flex items-center justify-between">
                <dt className="text-sm text-muted-foreground">App ID</dt>
                <dd className="font-mono text-xs text-muted-foreground">{app.id}</dd>
              </div>
              <div className="flex items-center justify-between">
                <dt className="text-sm text-muted-foreground">Created</dt>
                <dd className="text-sm text-muted-foreground">{formatDate(app.created_at)}</dd>
              </div>
              <div className="flex items-center justify-between">
                <dt className="text-sm text-muted-foreground">Updated</dt>
                <dd className="text-sm text-muted-foreground">{formatDate(app.updated_at)}</dd>
              </div>
            </dl>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
