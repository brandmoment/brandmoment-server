import { redirect } from "next/navigation";
import { auth } from "@/lib/auth";
import { headers } from "next/headers";
import { Sidebar } from "@/components/Sidebar";
import { Topbar } from "@/components/Topbar";
import type { OrgType } from "@/types/org";

export default async function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const session = await auth.api.getSession({
    headers: await headers(),
  });

  if (!session) {
    redirect("/login");
  }

  const activeOrg = session.session.activeOrganizationId
    ? await auth.api.getFullOrganization({
        headers: await headers(),
        query: { organizationId: session.session.activeOrganizationId },
      }).catch(() => null)
    : null;

  // Build org list from session. BetterAuth organization plugin exposes
  // the user's memberships. We derive a minimal list for the switcher.
  // In production this would come from GET /v1/me orgs[] array.
  const orgs: Array<{ id: string; name: string; slug: string }> =
    activeOrg
      ? [{ id: activeOrg.id, name: activeOrg.name, slug: activeOrg.slug }]
      : [];

  const orgType = (activeOrg?.metadata?.type as OrgType | undefined) ?? null;

  return (
    <div className="flex h-screen flex-col overflow-hidden">
      <Topbar
        orgs={orgs}
        userName={session.user.name}
        userEmail={session.user.email}
      />
      <div className="flex flex-1 overflow-hidden">
        <Sidebar orgType={orgType} />
        <main className="flex-1 overflow-y-auto bg-muted/10 p-6">
          {children}
        </main>
      </div>
    </div>
  );
}
