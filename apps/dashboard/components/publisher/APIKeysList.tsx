"use client";

import { useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useApiKeys } from "@/hooks/useApiKeys";
import { useCreateApiKey } from "@/hooks/useCreateApiKey";
import { useRevokeApiKey } from "@/hooks/useRevokeApiKey";
import { ApiKeyRevealModal } from "@/components/publisher/ApiKeyRevealModal";
import type { APIKey, CreateAPIKeyResponse } from "@/types/api-key";

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

interface CreateKeyDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  appId: string;
  onCreated: (response: CreateAPIKeyResponse) => void;
}

function CreateKeyDialog({ open, onOpenChange, appId, onCreated }: CreateKeyDialogProps) {
  const [name, setName] = useState("");
  const { mutateAsync: createKey, isPending } = useCreateApiKey();

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    try {
      const response = await createKey({ appId, body: { name: name.trim() } });
      onCreated(response);
      setName("");
      onOpenChange(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to create API key");
    }
  }

  function handleOpenChange(next: boolean) {
    if (!next) setName("");
    onOpenChange(next);
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Create API Key</DialogTitle>
          <DialogDescription>
            Give your API key a descriptive name (e.g., &ldquo;Production&rdquo;, &ldquo;Beta&rdquo;).
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="key-name">Key name</Label>
            <Input
              id="key-name"
              placeholder="Production"
              value={name}
              onChange={(e) => setName(e.target.value)}
              disabled={isPending}
              required
            />
          </div>
          <DialogFooter className="gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => handleOpenChange(false)}
              disabled={isPending}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isPending || !name.trim()}>
              {isPending ? "Creating..." : "Create key"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

interface RevokeConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  keyName: string;
  onConfirm: () => void;
  isPending: boolean;
}

function RevokeConfirmDialog({
  open,
  onOpenChange,
  keyName,
  onConfirm,
  isPending,
}: RevokeConfirmDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Revoke API Key</DialogTitle>
          <DialogDescription>
            Are you sure you want to revoke &quot;{keyName}&quot;? Any SDKs using
            this key will stop working immediately.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter className="gap-2">
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isPending}
          >
            Cancel
          </Button>
          <Button variant="destructive" onClick={onConfirm} disabled={isPending}>
            {isPending ? "Revoking..." : "Revoke key"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

interface APIKeysListProps {
  appId: string;
}

export function APIKeysList({ appId }: APIKeysListProps) {
  const [createOpen, setCreateOpen] = useState(false);
  const [revealedKey, setRevealedKey] = useState<CreateAPIKeyResponse | null>(null);
  const [revokeTarget, setRevokeTarget] = useState<APIKey | null>(null);

  const { data, isLoading, isError } = useApiKeys({ appId });
  const { mutateAsync: revokeKey, isPending: isRevoking } = useRevokeApiKey();

  function handleKeyCreated(response: CreateAPIKeyResponse) {
    setRevealedKey(response);
  }

  async function handleRevokeConfirm() {
    if (!revokeTarget) return;
    try {
      await revokeKey({ appId, keyId: revokeTarget.id });
      toast.success(`Key "${revokeTarget.name}" revoked`);
      setRevokeTarget(null);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to revoke key");
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          API keys authenticate your SDK with BrandMoment.
        </p>
        <Button size="sm" onClick={() => setCreateOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Create key
        </Button>
      </div>

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-14 w-full" />
          ))}
        </div>
      ) : isError ? (
        <p className="text-sm text-destructive">Failed to load API keys.</p>
      ) : !data || data.items.length === 0 ? (
        <div className="rounded-md border border-dashed p-8 text-center">
          <p className="text-sm text-muted-foreground">No API keys yet.</p>
          <Button
            variant="outline"
            className="mt-4"
            size="sm"
            onClick={() => setCreateOpen(true)}
          >
            <Plus className="mr-2 h-4 w-4" />
            Create your first key
          </Button>
        </div>
      ) : (
        <div className="rounded-md border divide-y">
          {data.items.map((key) => (
            <div
              key={key.id}
              className="flex items-center justify-between px-4 py-3"
            >
              <div className="space-y-0.5">
                <p className="text-sm font-medium">{key.name}</p>
                <p className="font-mono text-xs text-muted-foreground">
                  {key.key_prefix}...
                </p>
              </div>
              <div className="flex items-center gap-3">
                <span className="text-xs text-muted-foreground">
                  {formatDate(key.created_at)}
                </span>
                <Badge variant={key.is_revoked ? "destructive" : "success"}>
                  {key.is_revoked ? "Revoked" : "Active"}
                </Badge>
                {!key.is_revoked && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setRevokeTarget(key)}
                    className="text-destructive hover:text-destructive hover:bg-destructive/10"
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      <CreateKeyDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        appId={appId}
        onCreated={handleKeyCreated}
      />

      <ApiKeyRevealModal
        open={Boolean(revealedKey)}
        apiKey={revealedKey?.key ?? null}
        keyName={revealedKey?.name ?? ""}
        onConfirm={() => setRevealedKey(null)}
      />

      <RevokeConfirmDialog
        open={Boolean(revokeTarget)}
        onOpenChange={(open) => !open && setRevokeTarget(null)}
        keyName={revokeTarget?.name ?? ""}
        onConfirm={handleRevokeConfirm}
        isPending={isRevoking}
      />
    </div>
  );
}
