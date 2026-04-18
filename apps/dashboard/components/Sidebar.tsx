"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  LayoutGrid,
  BarChart3,
  Settings,
  Code2,
  Megaphone,
  Building2,
  List,
} from "lucide-react";
import type { OrgType } from "@/types/org";

interface NavItem {
  label: string;
  href: string;
  icon: React.ComponentType<{ className?: string }>;
}

const PUBLISHER_NAV: NavItem[] = [
  { label: "Apps", href: "/apps", icon: LayoutGrid },
  { label: "Rules", href: "/rules", icon: List },
  { label: "Analytics", href: "/analytics", icon: BarChart3 },
];

const BRAND_NAV: NavItem[] = [
  { label: "Campaigns", href: "/campaigns", icon: Megaphone },
  { label: "Analytics", href: "/analytics", icon: BarChart3 },
];

const ADMIN_NAV: NavItem[] = [
  { label: "Organizations", href: "/admin/organizations", icon: Building2 },
  { label: "All Campaigns", href: "/admin/campaigns", icon: Megaphone },
  { label: "All Apps", href: "/admin/apps", icon: Code2 },
  { label: "Analytics", href: "/admin/analytics", icon: BarChart3 },
  { label: "Settings", href: "/admin/settings", icon: Settings },
];

function getNavItems(orgType: OrgType | null): NavItem[] {
  if (orgType === "admin") return ADMIN_NAV;
  if (orgType === "brand") return BRAND_NAV;
  return PUBLISHER_NAV;
}

interface SidebarProps {
  orgType: OrgType | null;
}

export function Sidebar({ orgType }: SidebarProps) {
  const pathname = usePathname();
  const navItems = getNavItems(orgType);

  return (
    <aside className="flex h-full w-60 flex-col border-r bg-background px-3 py-4">
      <nav className="flex flex-col gap-1">
        {navItems.map((item) => {
          const Icon = item.icon;
          const isActive =
            pathname === item.href || pathname.startsWith(item.href + "/");
          return (
            <Button
              key={item.href}
              variant="ghost"
              asChild
              className={cn(
                "w-full justify-start gap-3",
                isActive && "bg-accent text-accent-foreground font-medium"
              )}
            >
              <Link href={item.href}>
                <Icon className="h-4 w-4 shrink-0" />
                {item.label}
              </Link>
            </Button>
          );
        })}
      </nav>
    </aside>
  );
}
