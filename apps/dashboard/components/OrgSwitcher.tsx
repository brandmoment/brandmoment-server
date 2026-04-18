"use client";

import { useActiveOrg } from "@/hooks/useActiveOrg";
import { authClient } from "@/lib/auth-client";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { ChevronsUpDown, Building2 } from "lucide-react";

interface OrgEntry {
  id: string;
  name: string;
  slug: string;
}

interface OrgSwitcherProps {
  orgs: OrgEntry[];
}

export function OrgSwitcher({ orgs }: OrgSwitcherProps) {
  const { activeOrgId, setActiveOrgId } = useActiveOrg();

  const activeOrg = orgs.find((o) => o.id === activeOrgId) ?? orgs[0] ?? null;

  function handleSelectOrg(orgId: string) {
    setActiveOrgId(orgId);
    // Persist the active org in BetterAuth so server components can read it
    authClient.organization.setActive({ organizationId: orgId });
  }

  if (orgs.length === 0) {
    return null;
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" className="flex items-center gap-2 min-w-40">
          <Building2 className="h-4 w-4 shrink-0" />
          <span className="truncate">{activeOrg?.name ?? "Select org"}</span>
          <ChevronsUpDown className="ml-auto h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-56">
        <DropdownMenuLabel>Organizations</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {orgs.map((org) => (
          <DropdownMenuItem
            key={org.id}
            onSelect={() => handleSelectOrg(org.id)}
            className="cursor-pointer"
          >
            <Building2 className="mr-2 h-4 w-4" />
            <span className="truncate">{org.name}</span>
            {org.id === activeOrgId && (
              <span className="ml-auto text-xs text-muted-foreground">
                active
              </span>
            )}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
