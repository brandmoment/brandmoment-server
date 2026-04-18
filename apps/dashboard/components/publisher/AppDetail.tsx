"use client";

import { ArrowLeft } from "lucide-react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { usePublisherApp } from "@/hooks/usePublisherApp";
import { AppOverviewTab } from "@/components/publisher/AppOverviewTab";
import { APIKeysList } from "@/components/publisher/APIKeysList";
import { RulesList } from "@/components/publisher/RulesList";

interface AppDetailProps {
  appId: string;
}

export function AppDetail({ appId }: AppDetailProps) {
  const { data: app, isLoading, isError } = usePublisherApp(appId);

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-10 w-80" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (isError || !app) {
    return (
      <div className="space-y-4">
        <Button variant="ghost" asChild>
          <Link href="/apps">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to Apps
          </Link>
        </Button>
        <p className="text-sm text-destructive">
          App not found or failed to load.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-start gap-4">
        <Button variant="ghost" size="sm" asChild>
          <Link href="/apps">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Apps
          </Link>
        </Button>
      </div>

      <div className="flex items-center gap-3">
        <h1 className="text-2xl font-semibold tracking-tight">{app.name}</h1>
        <Badge variant={app.is_active ? "success" : "outline"}>
          {app.is_active ? "Active" : "Inactive"}
        </Badge>
        <Badge variant="outline">
          {app.platform.toUpperCase()}
        </Badge>
      </div>

      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="api-keys">API Keys</TabsTrigger>
          <TabsTrigger value="rules">Rules</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="mt-6">
          <AppOverviewTab app={app} />
        </TabsContent>

        <TabsContent value="api-keys" className="mt-6">
          <APIKeysList appId={appId} />
        </TabsContent>

        <TabsContent value="rules" className="mt-6">
          <RulesList appId={appId} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
